//go:build windows

package singleinstance

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32          = windows.NewLazySystemDLL("user32.dll")
	messageBox      = user32.NewProc("MessageBoxW")
	alreadyRunning  = windows.StringToUTF16Ptr("Roblox Account Manager is already running.")
	applicationName = windows.StringToUTF16Ptr("Roblox Account Manager")
)

type Instance struct{ mutex windows.Handle }

func Acquire(applicationID string) (*Instance, bool, error) {
	name, err := windows.UTF16PtrFromString(`Local\` + applicationID)
	if err != nil {
		return nil, false, fmt.Errorf("encode single-instance mutex name: %w", err)
	}
	mutex, err := windows.CreateMutex(nil, false, name)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		if mutex != 0 {
			windows.CloseHandle(mutex)
		}
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("create single-instance mutex: %w", err)
	}
	if windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		windows.CloseHandle(mutex)
		return nil, false, nil
	}
	return &Instance{mutex: mutex}, true, nil
}

func ShowAlreadyRunning() {
	_, _, _ = messageBox.Call(0, uintptr(unsafe.Pointer(alreadyRunning)), uintptr(unsafe.Pointer(applicationName)), 0)
}

func (instance *Instance) Close() error {
	if instance == nil || instance.mutex == 0 {
		return nil
	}
	err := windows.CloseHandle(instance.mutex)
	instance.mutex = 0
	return err
}
