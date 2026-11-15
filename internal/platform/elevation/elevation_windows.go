package elevation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const Supported = true

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

func currentUserSID() (string, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", fmt.Errorf("read current Windows user: %w", err)
	}
	return user.User.Sid.String(), nil
}

// requireCurrentUser rejects a helper that runs as another account, which
// happens when a standard user types an administrator's password. Its folders
// and registry would belong to that administrator instead of the requesting user.
func requireCurrentUser(requestingUserSID string) error {
	currentSID, err := currentUserSID()
	if err != nil {
		return err
	}
	if currentSID != requestingUserSID {
		return ErrOtherUser
	}
	return nil
}

func runElevated(ctx context.Context, arguments []string) (uint32, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	executable, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("find the application executable: %w", err)
	}
	process, err := startElevated(executable, windows.ComposeCommandLine(arguments))
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(process)
	for {
		waitStatus, err := windows.WaitForSingleObject(process, 100)
		if err != nil {
			return 0, fmt.Errorf("wait for the elevated helper: %w", err)
		}
		if waitStatus == windows.WAIT_OBJECT_0 {
			break
		}
		if err := ctx.Err(); err != nil {
			return 0, err
		}
	}
	var exitCode uint32
	if err := windows.GetExitCodeProcess(process, &exitCode); err != nil {
		return 0, fmt.Errorf("read the elevated helper exit code: %w", err)
	}
	return exitCode, nil
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
		return 0, fmt.Errorf("encode application executable: %w", err)
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
