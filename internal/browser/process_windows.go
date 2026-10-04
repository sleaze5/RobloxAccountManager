package browser

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	procThreadAttributeJobList = 0x0002000d
	waitObjectZero             = 0
	windowCloseGrace           = time.Second
	windowPollInterval         = 200 * time.Millisecond
)

var (
	kernel32            = windows.NewLazySystemDLL("kernel32.dll")
	user32              = windows.NewLazySystemDLL("user32.dll")
	isProcessInJob      = kernel32.NewProc("IsProcessInJob")
	setForegroundWindow = user32.NewProc("SetForegroundWindow")
	isWindowVisible     = user32.NewProc("IsWindowVisible")
)

type windowsBrowserProcess struct {
	process           windows.Handle
	job               windows.Handle
	input             *os.File
	output            *os.File
	exited            chan error
	windowsClosed     chan struct{}
	windowMonitorStop chan struct{}
	windowMonitorDone chan struct{}
	closeOnce         sync.Once
}

func startWindowsBrowserProcess(options ProcessOptions) (BrowserProcess, error) {
	if options.Executable == "" || options.UserDataPath == "" {
		return nil, errors.New("invalid browser process options")
	}
	launchToken, err := browserLaunchToken()
	if err != nil {
		return nil, err
	}
	if launchToken != 0 {
		defer launchToken.Close()
	}
	var commandRead, parentWrite, parentRead, commandWrite windows.Handle
	security := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), InheritHandle: 1}
	if err := windows.CreatePipe(&commandRead, &parentWrite, &security, 0); err != nil {
		return nil, err
	}
	defer func() {
		if commandRead != 0 {
			windows.CloseHandle(commandRead)
		}
		if parentWrite != 0 {
			windows.CloseHandle(parentWrite)
		}
	}()
	if err := windows.SetHandleInformation(parentWrite, windows.HANDLE_FLAG_INHERIT, 0); err != nil {
		return nil, err
	}
	if err := windows.CreatePipe(&parentRead, &commandWrite, &security, 0); err != nil {
		return nil, err
	}
	defer func() {
		if parentRead != 0 {
			windows.CloseHandle(parentRead)
		}
		if commandWrite != 0 {
			windows.CloseHandle(commandWrite)
		}
	}()
	if err := windows.SetHandleInformation(parentRead, windows.HANDLE_FLAG_INHERIT, 0); err != nil {
		return nil, err
	}
	standardInput, err := openInheritableNull(windows.GENERIC_READ, &security)
	if err != nil {
		return nil, fmt.Errorf("open browser standard input: %w", err)
	}
	defer windows.CloseHandle(standardInput)
	standardOutput, err := openInheritableNull(windows.GENERIC_WRITE|windows.FILE_READ_ATTRIBUTES, &security)
	if err != nil {
		return nil, fmt.Errorf("open browser standard output: %w", err)
	}
	defer windows.CloseHandle(standardOutput)
	standardError, err := openInheritableNull(windows.GENERIC_WRITE|windows.FILE_READ_ATTRIBUTES, &security)
	if err != nil {
		return nil, fmt.Errorf("open browser standard error: %w", err)
	}
	defer windows.CloseHandle(standardError)

	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if job != 0 {
			windows.CloseHandle(job)
		}
	}()
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		return nil, err
	}

	arguments := []string{options.Executable,
		"--user-data-dir=" + options.UserDataPath, "--remote-debugging-pipe",
		"--remote-debugging-io-pipes=" + inheritedPipeHandles(commandRead, commandWrite),
		"--no-first-run", "--no-default-browser-check", "--disable-background-mode",
		"--disable-infobars", "--incognito", "about:blank"}
	if launchToken != 0 {
		// The restricted, medium-integrity token retains its original elevation
		// type. Prevent Chrome from relaunching and losing the inherited CDP pipes;
		// privileges have already been reduced before process creation.
		arguments = append(arguments, "--do-not-de-elevate")
	}
	commandLine, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(arguments))
	if err != nil {
		return nil, err
	}
	executable, err := windows.UTF16PtrFromString(options.Executable)
	if err != nil {
		return nil, err
	}
	directory, err := windows.UTF16PtrFromString(options.UserDataPath)
	if err != nil {
		return nil, err
	}

	attributes, err := windows.NewProcThreadAttributeList(2)
	if err != nil {
		return nil, err
	}
	defer attributes.Delete()
	handles := []windows.Handle{standardInput, standardOutput, standardError, commandRead, commandWrite}
	if err := attributes.Update(windows.PROC_THREAD_ATTRIBUTE_HANDLE_LIST, unsafe.Pointer(&handles[0]), uintptr(len(handles))*unsafe.Sizeof(handles[0])); err != nil {
		return nil, err
	}
	jobs := []windows.Handle{job}
	if err := attributes.Update(procThreadAttributeJobList, unsafe.Pointer(&jobs[0]), unsafe.Sizeof(jobs[0])); err != nil {
		return nil, fmt.Errorf("attach browser process containment at creation: %w", err)
	}
	startup := windows.StartupInfoEx{}
	startup.Cb = uint32(unsafe.Sizeof(startup))
	startup.Flags = windows.STARTF_USESTDHANDLES
	startup.StdInput = standardInput
	startup.StdOutput = standardOutput
	startup.StdErr = standardError
	startup.ProcThreadAttributeList = attributes.List()
	information := windows.ProcessInformation{}
	flags := browserCreationFlags()
	var createErr error
	if launchToken != 0 {
		createErr = windows.CreateProcessAsUser(launchToken, executable, commandLine, nil, nil, true, flags, nil, directory, (*windows.StartupInfo)(unsafe.Pointer(&startup)), &information)
	} else {
		createErr = windows.CreateProcess(executable, commandLine, nil, nil, true, flags, nil, directory, (*windows.StartupInfo)(unsafe.Pointer(&startup)), &information)
	}
	// UpdateProcThreadAttribute retains pointers to these arrays until process
	// creation completes. Keep that ownership explicit across the syscall.
	runtime.KeepAlive(handles)
	runtime.KeepAlive(jobs)
	if createErr != nil {
		return nil, fmt.Errorf("start Chrome for Testing: %w", createErr)
	}
	windows.CloseHandle(information.Thread)
	if information.Process == 0 {
		return nil, errors.New("start Chrome for Testing: process handle is unavailable")
	}
	if err := verifyProcessInJob(information.Process, job); err != nil {
		windows.TerminateProcess(information.Process, 1)
		windows.CloseHandle(information.Process)
		return nil, fmt.Errorf("contain browser process: %w", err)
	}
	windows.CloseHandle(commandRead)
	commandRead = 0
	windows.CloseHandle(commandWrite)
	commandWrite = 0
	process := &windowsBrowserProcess{
		process:           information.Process,
		job:               job,
		input:             os.NewFile(uintptr(parentWrite), "cft-cdp-input"),
		output:            os.NewFile(uintptr(parentRead), "cft-cdp-output"),
		exited:            make(chan error, 1),
		windowsClosed:     make(chan struct{}, 1),
		windowMonitorStop: make(chan struct{}),
		windowMonitorDone: make(chan struct{}),
	}
	parentWrite, parentRead, job = 0, 0, 0
	go process.waitLoop()
	go process.watchWindows()
	return process, nil
}

func browserCreationFlags() uint32 {
	flags := uint32(windows.EXTENDED_STARTUPINFO_PRESENT)
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	if err := windows.QueryInformationJobObject(0, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits)), nil); err == nil && limits.BasicLimitInformation.LimitFlags&windows.JOB_OBJECT_LIMIT_BREAKAWAY_OK != 0 {
		flags |= windows.CREATE_BREAKAWAY_FROM_JOB
	}
	return flags
}

func verifyProcessInJob(process, job windows.Handle) error {
	var result int32
	r1, _, callErr := isProcessInJob.Call(uintptr(process), uintptr(job), uintptr(unsafe.Pointer(&result)))
	if r1 == 0 {
		if callErr != windows.ERROR_SUCCESS {
			return callErr
		}
		return errors.New("could not verify session containment")
	}
	if result == 0 {
		return errors.New("process was not assigned to its session job")
	}
	return nil
}

func inheritedPipeHandles(input, output windows.Handle) string {
	// Chromium's Windows pipe builder serializes inherited HANDLE values as
	// unsigned 32-bit integers.
	inputValue := strconv.FormatUint(uint64(uint32(uintptr(input))), 10)
	outputValue := strconv.FormatUint(uint64(uint32(uintptr(output))), 10)
	return inputValue + "," + outputValue
}

func openInheritableNull(access uint32, security *windows.SecurityAttributes) (windows.Handle, error) {
	name, err := windows.UTF16PtrFromString("NUL")
	if err != nil {
		return 0, err
	}
	return windows.CreateFile(
		name,
		access,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		security,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
}

func (process *windowsBrowserProcess) CDPPipes() (io.WriteCloser, io.ReadCloser) {
	return process.input, process.output
}
func (process *windowsBrowserProcess) AllWindowsClosed() <-chan struct{} {
	return process.windowsClosed
}
func (process *windowsBrowserProcess) Exited() <-chan error { return process.exited }

func (process *windowsBrowserProcess) waitLoop() {
	_, err := windows.WaitForSingleObject(process.process, windows.INFINITE)
	<-process.windowMonitorDone
	process.exited <- err
	close(process.exited)
}

func (process *windowsBrowserProcess) watchWindows() {
	defer close(process.windowMonitorDone)
	defer close(process.windowsClosed)
	ticker := time.NewTicker(windowPollInterval)
	defer ticker.Stop()
	seenWindow := false
	missingSince := time.Time{}
	for {
		select {
		case <-process.windowMonitorStop:
			return
		case <-ticker.C:
			if process.Wait(0) {
				if seenWindow && process.exitedNormally() {
					process.windowsClosed <- struct{}{}
				}
				return
			}
			window, err := visibleWindowInJob(process.job)
			if err != nil {
				continue
			}
			if window != 0 {
				seenWindow = true
				missingSince = time.Time{}
				continue
			}
			if !seenWindow {
				continue
			}
			if missingSince.IsZero() {
				missingSince = time.Now()
				continue
			}
			if time.Since(missingSince) >= windowCloseGrace {
				process.windowsClosed <- struct{}{}
				return
			}
		}
	}
}

func (process *windowsBrowserProcess) exitedNormally() bool {
	var exitCode uint32
	return windows.GetExitCodeProcess(process.process, &exitCode) == nil && exitCode == 0
}

func (process *windowsBrowserProcess) ExitError() error {
	if !process.Wait(200 * time.Millisecond) {
		return nil
	}
	var exitCode uint32
	if err := windows.GetExitCodeProcess(process.process, &exitCode); err != nil {
		return fmt.Errorf("read browser exit code: %w", err)
	}
	return fmt.Errorf("browser process exited with code 0x%08X", exitCode)
}

func (process *windowsBrowserProcess) Wait(timeout time.Duration) bool {
	milliseconds := uint32(max(timeout.Milliseconds(), 0))
	if timeout <= 0 {
		milliseconds = 0
	}
	result, _ := windows.WaitForSingleObject(process.process, milliseconds)
	return result == waitObjectZero
}

func (process *windowsBrowserProcess) Terminate() error {
	if process.job == 0 {
		return nil
	}
	return windows.TerminateJobObject(process.job, 1)
}

func (process *windowsBrowserProcess) Close() error {
	process.closeOnce.Do(func() {
		close(process.windowMonitorStop)
		<-process.windowMonitorDone
		if process.input != nil {
			_ = process.input.Close()
		}
		if process.output != nil {
			_ = process.output.Close()
		}
		if process.process != 0 {
			windows.CloseHandle(process.process)
			process.process = 0
		}
		if process.job != 0 {
			windows.CloseHandle(process.job)
			process.job = 0
		}
	})
	return nil
}

type browserWindowSearch struct {
	processIDs map[uint32]struct{}
	window     windows.HWND
}

// Go never releases Windows callbacks and terminates the process when its
// callback table fills. Reuse one callback, with per-enumeration state in LPARAM.
var browserWindowCallback = syscall.NewCallback(func(window windows.HWND, search *browserWindowSearch) uintptr {
	if search.window != 0 {
		return 1
	}
	var processID uint32
	windows.GetWindowThreadProcessId(window, &processID)
	if _, owned := search.processIDs[processID]; !owned {
		return 1
	}
	visible, _, _ := isWindowVisible.Call(uintptr(window))
	if visible != 0 {
		search.window = window
	}
	return 1
})

func visibleWindowInJob(job windows.Handle) (windows.HWND, error) {
	processIDs, err := processIDsInJob(job)
	if err != nil {
		return 0, err
	}
	search := browserWindowSearch{processIDs: processIDs}
	if err := windows.EnumWindows(browserWindowCallback, unsafe.Pointer(&search)); err != nil {
		return 0, err
	}
	return search.window, nil
}

func (process *windowsBrowserProcess) Focus() error {
	target, err := visibleWindowInJob(process.job)
	if err != nil {
		return err
	}
	if target == 0 {
		return errors.New("the browser window is not available")
	}
	result, _, callErr := setForegroundWindow.Call(uintptr(target))
	if result == 0 {
		if callErr != windows.ERROR_SUCCESS {
			return callErr
		}
		return errors.New("windows refused to focus the browser")
	}
	return nil
}

func processIDsInJob(job windows.Handle) (map[uint32]struct{}, error) {
	const capacity = 256
	buffer := make([]byte, 8+capacity*int(unsafe.Sizeof(uintptr(0))))
	if err := windows.QueryInformationJobObject(job, windows.JobObjectBasicProcessIdList, uintptr(unsafe.Pointer(&buffer[0])), uint32(len(buffer)), nil); err != nil {
		return nil, err
	}
	assigned := binary.LittleEndian.Uint32(buffer[0:4])
	listed := binary.LittleEndian.Uint32(buffer[4:8])
	if assigned > listed {
		return nil, errors.New("too many browser processes to focus safely")
	}
	result := make(map[uint32]struct{}, listed)
	offset := 8
	size := int(unsafe.Sizeof(uintptr(0)))
	for range listed {
		var value uint64
		if size == 8 {
			value = binary.LittleEndian.Uint64(buffer[offset:])
		} else {
			value = uint64(binary.LittleEndian.Uint32(buffer[offset:]))
		}
		result[uint32(value)] = struct{}{}
		offset += size
	}
	return result, nil
}
