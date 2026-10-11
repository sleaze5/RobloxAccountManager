//go:build windows

package gamelaunch

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	assocIsProtocol    = 0x1000
	assocStrExecutable = 2
	noAssociation      = windows.Errno(0x80070483)
)

var assocQueryString = windows.NewLazySystemDLL("shlwapi.dll").NewProc("AssocQueryStringW")

func requireProtocolHandler() error {
	executable, err := protocolExecutable("roblox-player")
	if errors.Is(err, noAssociation) {
		return &ClientError{Message: protocolUnavailable}
	}
	if err != nil {
		return fmt.Errorf("find the roblox-player handler: %w", err)
	}
	if _, err := os.Stat(executable); errors.Is(err, fs.ErrNotExist) {
		return &ClientError{Message: protocolUnavailable}
	}
	return nil
}

func protocolExecutable(protocol string) (string, error) {
	name, err := windows.UTF16PtrFromString(protocol)
	if err != nil {
		return "", err
	}
	verb, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return "", err
	}
	var size uint32
	if result, _, _ := assocQueryString.Call(assocIsProtocol, assocStrExecutable, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(verb)), 0, uintptr(unsafe.Pointer(&size))); int32(result) < 0 {
		return "", windows.Errno(uint32(result))
	}
	if size == 0 {
		return "", noAssociation
	}
	buffer := make([]uint16, size)
	if result, _, _ := assocQueryString.Call(assocIsProtocol, assocStrExecutable, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&size))); int32(result) < 0 {
		return "", windows.Errno(uint32(result))
	}
	return windows.UTF16ToString(buffer), nil
}
