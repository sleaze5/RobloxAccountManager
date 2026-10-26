//go:build windows

package shellidentity

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"syscall"
	"unsafe"

	"github.com/sleaze5/RobloxAccountManager/internal/appmeta"
	"golang.org/x/sys/windows"
)

const (
	wmShowWindow = 0x0018
	wmDestroy    = 0x0002
	vtLPWStr     = 31
)

var (
	getPropertyStore = windows.NewLazySystemDLL("shell32.dll").NewProc("SHGetPropertyStoreForWindow")
	propertyStoreID  = windows.GUID{Data1: 0x886D8EEB, Data2: 0x8CF2, Data3: 0x4446, Data4: [8]byte{0x8D, 0x02, 0xCD, 0xBA, 0x1D, 0xBD, 0xCF, 0x99}}
	appUserModelID   = windows.GUID{Data1: 0x9F4C2855, Data2: 0x9F79, Data3: 0x4B39, Data4: [8]byte{0xA8, 0xD0, 0xE1, 0xD4, 0x2D, 0xE1, 0xD5, 0xF3}}
)

type propertyKey struct {
	format windows.GUID
	id     uint32
}

// PROPVARIANT reserves two pointer-sized words for its value union.
type propertyValue struct {
	kind uint16
	_    [3]uint16
	text *uint16
	_    uintptr
}

type propertyStore struct {
	vtable *[8]uintptr
}

func New(logger *slog.Logger) (func(uintptr, uint32, uintptr, uintptr) (uintptr, bool), error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve shell relaunch executable: %w", err)
	}
	// Wails Options.Description only supplies its About dialog. Explorer needs
	// explicit relaunch properties to name shortcuts pinned from a running window.
	values := [4]string{syscall.EscapeArg(executable), executable + ",-3", appmeta.DisplayName, appmeta.Identifier}
	configured := make(map[uintptr]bool)
	return func(hwnd uintptr, message uint32, wParam, _ uintptr) (uintptr, bool) {
		switch message {
		case wmShowWindow:
			if wParam != 0 && !configured[hwnd] {
				configured[hwnd] = true
				if err := setProperties(hwnd, &values); err != nil {
					logger.Warn("could not set taskbar identity", "operation", "shell-identity", "error", err)
				}
			}
		case wmDestroy:
			if configured[hwnd] {
				// Window property stores require VT_EMPTY cleanup before destruction.
				if err := setProperties(hwnd, nil); err != nil {
					logger.Warn("could not clear taskbar identity", "operation", "shell-identity-cleanup", "error", err)
				}
				delete(configured, hwnd)
			}
		}
		return 0, false
	}, nil
}

func setProperties(hwnd uintptr, values *[4]string) error {
	var store *propertyStore
	result, _, _ := getPropertyStore.Call(hwnd, uintptr(unsafe.Pointer(&propertyStoreID)), uintptr(unsafe.Pointer(&store)))
	if int32(result) < 0 {
		return fmt.Errorf("open window property store: HRESULT %#x", uint32(result))
	}
	defer func() {
		// IUnknown::Release returns a reference count, not an HRESULT or last error.
		_, _, _ = syscall.SyscallN(store.vtable[2], uintptr(unsafe.Pointer(store)))
	}()

	// RelaunchCommand (2), RelaunchIconResource (3), RelaunchDisplayNameResource
	// (4), then ID (5). Setting ID last tells Explorer to refresh the complete set.
	var clearErr error
	for index := range 4 {
		key := propertyKey{format: appUserModelID, id: uint32(index + 2)}
		value := propertyValue{}
		if values != nil {
			text, err := windows.UTF16PtrFromString(values[index])
			if err != nil {
				return fmt.Errorf("encode shell property %d: %w", key.id, err)
			}
			value.kind = vtLPWStr
			value.text = text
		}
		// IPropertyStore::SetValue copies the string; this variant borrows Go memory.
		result, _, _ = syscall.SyscallN(store.vtable[6], uintptr(unsafe.Pointer(store)), uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&value)))
		if int32(result) < 0 {
			err := fmt.Errorf("set shell property %d: HRESULT %#x", key.id, uint32(result))
			if values != nil {
				return err
			}
			clearErr = errors.Join(clearErr, err)
		}
	}
	return clearErr
}
