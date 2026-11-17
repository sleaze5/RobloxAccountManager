package elevation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const Supported = true

// The helper answers within moments after Windows starts it. Only the
// administrator prompt itself, which ShellExecuteEx waits for, takes longer.
const helperStartTimeout = 15 * time.Second

const (
	// SEE_MASK_NOCLOSEPROCESS returns the helper process handle, so that its
	// exit code can be read. SEE_MASK_NOASYNC finishes the call before returning.
	shellExecuteMaskNoCloseProcess = 0x00000040
	shellExecuteMaskNoAsync        = 0x00000100
)

var procShellExecuteEx = windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteExW")

// shellExecuteInfo is SHELLEXECUTEINFOW.
type shellExecuteInfo struct {
	size                uint32
	mask                uint32
	window              windows.HWND
	verb                *uint16
	file                *uint16
	parameters          *uint16
	directory           *uint16
	show                int32
	instanceApplication windows.Handle
	itemIDList          uintptr
	class               *uint16
	classKey            windows.Handle
	hotKey              uint32
	iconOrMonitor       windows.Handle
	process             windows.Handle
}

func IsElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

// startHelper starts the elevated helper and connects it through two
// anonymous pipes. The pipes have no name, so no other process can find them.
// The helper copies its pipe ends out of this process, which needs the access
// to this process that only the same user's elevated processes have.
func startHelper(ctx context.Context) (*helperConnection, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("find the app executable: %w", err)
	}
	var requestRead, requestWrite, responseRead, responseWrite windows.Handle
	if err := windows.CreatePipe(&requestRead, &requestWrite, nil, 0); err != nil {
		return nil, fmt.Errorf("create the elevated helper request pipe: %w", err)
	}
	if err := windows.CreatePipe(&responseRead, &responseWrite, nil, 0); err != nil {
		windows.CloseHandle(requestRead)
		windows.CloseHandle(requestWrite)
		return nil, fmt.Errorf("create the elevated helper response pipe: %w", err)
	}
	var releaseOnce sync.Once
	releaseHelperEnds := func() {
		releaseOnce.Do(func() {
			windows.CloseHandle(requestRead)
			windows.CloseHandle(responseWrite)
		})
	}
	parameters := windows.ComposeCommandLine([]string{
		helperArgument,
		strconv.FormatUint(uint64(windows.GetCurrentProcessId()), 10),
		strconv.FormatUint(uint64(requestRead), 10),
		strconv.FormatUint(uint64(responseWrite), 10),
	})
	process, err := startElevated(executable, parameters)
	if err != nil {
		releaseHelperEnds()
		windows.CloseHandle(requestWrite)
		windows.CloseHandle(responseRead)
		return nil, err
	}
	defer windows.CloseHandle(process)
	requests := os.NewFile(uintptr(requestWrite), "elevated-helper-requests")
	responses := readResponses(os.NewFile(uintptr(responseRead), "elevated-helper-responses"))
	hasExited := func() bool {
		waitStatus, err := windows.WaitForSingleObject(process, 0)
		return err != nil || waitStatus == windows.WAIT_OBJECT_0
	}
	readyError := waitForReady(ctx, responses, hasExited, releaseHelperEnds)
	// The helper holds its own copies once it is ready. Closing ours lets a
	// read or write fail as soon as the helper exits.
	releaseHelperEnds()
	if readyError != nil {
		_ = requests.Close()
		return nil, readyError
	}
	return &helperConnection{requests: requests, responses: responses}, nil
}

// connectToApplication runs in the helper. It checks that the process that
// started it is this executable, then copies the helper's pipe ends out of it.
// It returns ErrOtherUser, together with the pipes to report it, when the
// helper runs as another Windows user than the application, which happens
// when a standard user types an administrator's password. That user's folders
// and registry would be cleaned instead.
func connectToApplication(arguments []string) (io.ReadCloser, io.WriteCloser, error) {
	if len(arguments) != 3 {
		return nil, nil, errors.New("invalid elevated helper arguments")
	}
	applicationID, idError := strconv.ParseUint(arguments[0], 10, 32)
	requestHandle, requestError := strconv.ParseUint(arguments[1], 10, 64)
	responseHandle, responseError := strconv.ParseUint(arguments[2], 10, 64)
	if err := errors.Join(idError, requestError, responseError); err != nil {
		return nil, nil, fmt.Errorf("parse elevated helper arguments: %w", err)
	}
	application, err := windows.OpenProcess(windows.PROCESS_DUP_HANDLE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(applicationID))
	if err != nil {
		return nil, nil, fmt.Errorf("open the app process: %w", err)
	}
	defer windows.CloseHandle(application)
	if err := requireSameExecutable(application); err != nil {
		return nil, nil, err
	}
	requests, err := duplicatePipe(application, windows.Handle(requestHandle), "elevated-helper-requests")
	if err != nil {
		return nil, nil, err
	}
	responses, err := duplicatePipe(application, windows.Handle(responseHandle), "elevated-helper-responses")
	if err != nil {
		_ = requests.Close()
		return nil, nil, err
	}
	if err := requireSameUser(application); err != nil {
		return requests, responses, err
	}
	return requests, responses, nil
}

func requireSameExecutable(application windows.Handle) error {
	applicationPath, err := processImagePath(application)
	if err != nil {
		return err
	}
	helperPath, err := processImagePath(windows.CurrentProcess())
	if err != nil {
		return err
	}
	if !strings.EqualFold(applicationPath, helperPath) {
		return errors.New("the elevated helper was started by another program")
	}
	return nil
}

func processImagePath(process windows.Handle) (string, error) {
	// 32768 UTF-16 units is the longest Windows path.
	var path [32768]uint16
	size := uint32(len(path))
	if err := windows.QueryFullProcessImageName(process, 0, &path[0], &size); err != nil {
		return "", fmt.Errorf("read process path: %w", err)
	}
	return windows.UTF16ToString(path[:size]), nil
}

func requireSameUser(application windows.Handle) error {
	var token windows.Token
	if err := windows.OpenProcessToken(application, windows.TOKEN_QUERY, &token); err != nil {
		return fmt.Errorf("open the app token: %w", err)
	}
	defer token.Close()
	applicationUser, err := token.GetTokenUser()
	if err != nil {
		return fmt.Errorf("read the app user: %w", err)
	}
	helperUser, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return fmt.Errorf("read the helper user: %w", err)
	}
	if !applicationUser.User.Sid.Equals(helperUser.User.Sid) {
		return ErrOtherUser
	}
	return nil
}

func duplicatePipe(application, source windows.Handle, name string) (*os.File, error) {
	var duplicate windows.Handle
	if err := windows.DuplicateHandle(application, source, windows.CurrentProcess(), &duplicate, 0, false, windows.DUPLICATE_SAME_ACCESS); err != nil {
		return nil, fmt.Errorf("copy the %s pipe: %w", name, err)
	}
	// The application closes its copies when the helper does not answer in
	// time, and Windows may give the number to another handle. Accept only a pipe.
	if fileType, err := windows.GetFileType(duplicate); err != nil || fileType != windows.FILE_TYPE_PIPE {
		windows.CloseHandle(duplicate)
		return nil, fmt.Errorf("the %s handle is not a pipe", name)
	}
	return os.NewFile(uintptr(duplicate), name), nil
}

func startElevated(executable, parameters string) (windows.Handle, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// ShellExecuteEx may use shell extensions, which need COM on the calling
	// thread. S_FALSE still initializes COM and must be balanced by CoUninitialize.
	if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED|windows.COINIT_DISABLE_OLE1DDE); err != nil && !errors.Is(err, syscall.Errno(1)) {
		return 0, fmt.Errorf("initialize COM for the administrator prompt: %w", err)
	}
	defer windows.CoUninitialize()
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return 0, fmt.Errorf("encode elevation verb: %w", err)
	}
	file, err := windows.UTF16PtrFromString(executable)
	if err != nil {
		return 0, fmt.Errorf("encode the app executable: %w", err)
	}
	encodedParameters, err := windows.UTF16PtrFromString(parameters)
	if err != nil {
		return 0, fmt.Errorf("encode elevated helper arguments: %w", err)
	}
	executeInfo := shellExecuteInfo{
		mask:       shellExecuteMaskNoCloseProcess | shellExecuteMaskNoAsync,
		verb:       verb,
		file:       file,
		parameters: encodedParameters,
		show:       windows.SW_HIDE,
	}
	executeInfo.size = uint32(unsafe.Sizeof(executeInfo))
	succeeded, _, callError := procShellExecuteEx.Call(uintptr(unsafe.Pointer(&executeInfo)))
	if succeeded == 0 {
		if errors.Is(callError, windows.ERROR_CANCELLED) {
			return 0, ErrDeclined
		}
		return 0, fmt.Errorf("start the elevated helper: %w", callError)
	}
	if executeInfo.process == 0 {
		return 0, errors.New("the elevated helper started without a process handle")
	}
	return executeInfo.process, nil
}

func readResponses(source io.ReadCloser) <-chan taskResponse {
	responses := make(chan taskResponse, 1)
	go func() {
		defer close(responses)
		// This goroutine owns source. Closing a pipe while a read waits on it
		// does not end the read on Windows, so it closes after the helper exits.
		defer source.Close()
		decoder := json.NewDecoder(source)
		for {
			var response taskResponse
			if decoder.Decode(&response) != nil {
				return
			}
			responses <- response
		}
	}()
	return responses
}

// waitForReady waits for the first answer of a starting helper. When the
// helper exits early, releaseHelperEnds closes the application's copies of the
// helper's pipe ends, so that responses delivers any last answer and then closes.
func waitForReady(ctx context.Context, responses <-chan taskResponse, hasExited func() bool, releaseHelperEnds func()) error {
	ctx, cancel := context.WithTimeout(ctx, helperStartTimeout)
	defer cancel()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	exitChecks := ticker.C
	for {
		select {
		case response, isOpen := <-responses:
			switch {
			case !isOpen:
				return errStopped
			case response.Status == statusReady:
				return nil
			case response.Status == statusOtherUser:
				return ErrOtherUser
			default:
				return fmt.Errorf("the elevated helper answered %q instead of ready", response.Status)
			}
		case <-exitChecks:
			if hasExited() {
				releaseHelperEnds()
				exitChecks = nil
			}
		case <-ctx.Done():
			return fmt.Errorf("wait for the elevated helper: %w", ctx.Err())
		}
	}
}
