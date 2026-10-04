//go:build windows

package gamelaunch

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"golang.org/x/sys/windows"
)

const (
	shellWindowDesktop      = 8
	shellWindowNeedDispatch = 1
)

var shellWindowsClass = ole.NewGUID("{9BA05972-F6A8-11CF-A442-00A0C90A8F39}")

func launchFromExplorer(ctx context.Context, target, arguments, directory string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// S_FALSE still initializes COM and must be balanced by CoUninitialize.
	if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED|windows.COINIT_DISABLE_OLE1DDE); err != nil && !errors.Is(err, syscall.Errno(1)) {
		return fmt.Errorf("%w: initialize desktop automation: %v", ErrDesktopUnavailable, err)
	}
	defer windows.CoUninitialize()

	shell, window, err := desktopShell()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDesktopUnavailable, err)
	}
	defer shell.Release()
	if err := verifyDesktopShell(window); err != nil {
		return fmt.Errorf("%w: %v", ErrDesktopUnavailable, err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// This object belongs to the existing desktop Explorer, not this process.
	// Never fall back to a local launch: it would inherit our token and process tree.
	result, err := shell.CallMethod("ShellExecute", target, arguments, directory, "open", int32(windows.SW_SHOWNORMAL))
	if result != nil {
		defer result.Clear()
	}
	if err != nil {
		return fmt.Errorf("start Roblox Player through desktop Explorer: %w", err)
	}
	return nil
}

func desktopShell() (*ole.IDispatch, windows.HWND, error) {
	if windows.GetShellWindow() == 0 {
		return nil, 0, fmt.Errorf("desktop Explorer is not running")
	}
	unknown, err := ole.CreateInstance(shellWindowsClass, ole.IID_IUnknown)
	if err != nil {
		return nil, 0, fmt.Errorf("connect to shell windows: %w", err)
	}
	defer unknown.Release()
	shellWindows, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, 0, fmt.Errorf("query shell windows: %w", err)
	}
	defer shellWindows.Release()
	location := ole.NewVariant(ole.VT_I4, 0)
	root := ole.NewVariant(ole.VT_EMPTY, 0)
	var handle int32
	desktop, err := shellWindows.CallMethod("FindWindowSW", &location, &root, shellWindowDesktop, &handle, shellWindowNeedDispatch)
	if desktop != nil {
		defer desktop.Clear()
	}
	if err != nil {
		return nil, 0, fmt.Errorf("find desktop window: %w", err)
	}
	if desktop == nil || desktop.ToIDispatch() == nil || handle == 0 {
		return nil, 0, fmt.Errorf("desktop window automation is unavailable")
	}
	view, err := shellDispatchProperty(desktop.ToIDispatch(), "Document")
	if err != nil {
		return nil, 0, err
	}
	defer view.Release()
	shell, err := shellDispatchProperty(view, "Application")
	return shell, windows.HWND(uint32(handle)), err
}

func shellDispatchProperty(object *ole.IDispatch, name string) (*ole.IDispatch, error) {
	value, err := object.GetProperty(name)
	if value != nil {
		defer value.Clear()
	}
	if err != nil {
		return nil, fmt.Errorf("read desktop %s: %w", name, err)
	}
	if value == nil || value.ToIDispatch() == nil {
		return nil, fmt.Errorf("desktop %s is unavailable", name)
	}
	dispatch := value.ToIDispatch()
	dispatch.AddRef()
	return dispatch, nil
}

func verifyDesktopShell(window windows.HWND) error {
	if window == 0 || window != windows.GetShellWindow() {
		return fmt.Errorf("the Windows desktop changed during launch")
	}
	var processID uint32
	if _, err := windows.GetWindowThreadProcessId(window, &processID); err != nil {
		return fmt.Errorf("identify desktop process: %w", err)
	}
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, processID)
	if err != nil {
		return fmt.Errorf("open desktop process: %w", err)
	}
	defer windows.CloseHandle(process)
	var token windows.Token
	if err := windows.OpenProcessToken(process, windows.TOKEN_QUERY, &token); err != nil {
		return fmt.Errorf("open desktop token: %w", err)
	}
	defer token.Close()
	return verifyShellToken(token)
}

func verifyShellToken(token windows.Token) error {
	var elevated, returned uint32
	if err := windows.GetTokenInformation(token, windows.TokenElevation, (*byte)(unsafe.Pointer(&elevated)), uint32(unsafe.Sizeof(elevated)), &returned); err != nil {
		return fmt.Errorf("read desktop elevation: %w", err)
	}
	if elevated != 0 {
		return fmt.Errorf("desktop Explorer is running as administrator")
	}
	if err := windows.GetTokenInformation(token, windows.TokenIntegrityLevel, nil, 0, &returned); !errors.Is(err, windows.ERROR_INSUFFICIENT_BUFFER) {
		return fmt.Errorf("measure desktop integrity: %v", err)
	}
	if returned < uint32(unsafe.Sizeof(windows.Tokenmandatorylabel{})) {
		return fmt.Errorf("desktop integrity information is incomplete")
	}
	integrity := make([]byte, returned)
	if err := windows.GetTokenInformation(token, windows.TokenIntegrityLevel, &integrity[0], uint32(len(integrity)), &returned); err != nil {
		return fmt.Errorf("read desktop integrity: %w", err)
	}
	label := (*windows.Tokenmandatorylabel)(unsafe.Pointer(&integrity[0]))
	if label.Label.Sid == nil || !label.Label.Sid.IsWellKnown(windows.WinMediumLabelSid) {
		return fmt.Errorf("desktop Explorer is not running at normal-user integrity")
	}
	return nil
}
