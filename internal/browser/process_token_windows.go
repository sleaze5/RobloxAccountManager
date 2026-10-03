package browser

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	saferScopeUser       = 2
	saferLevelNormalUser = 0x20000
	saferLevelOpen       = 1
)

var (
	saferAPI                   = windows.NewLazySystemDLL("advapi32.dll")
	saferCreateLevel           = saferAPI.NewProc("SaferCreateLevel")
	saferComputeTokenFromLevel = saferAPI.NewProc("SaferComputeTokenFromLevel")
	saferCloseLevel            = saferAPI.NewProc("SaferCloseLevel")
)

func browserLaunchToken() (windows.Token, error) {
	var elevated, returned uint32
	if err := windows.GetTokenInformation(windows.GetCurrentProcessToken(), windows.TokenElevation, (*byte)(unsafe.Pointer(&elevated)), uint32(unsafe.Sizeof(elevated)), &returned); err != nil {
		return 0, fmt.Errorf("read browser launcher elevation: %w", err)
	}
	if elevated == 0 {
		return 0, nil
	}
	var current windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY|windows.TOKEN_DUPLICATE, &current); err != nil {
		return 0, fmt.Errorf("open browser launcher token: %w", err)
	}
	defer current.Close()
	return restrictedBrowserToken(current)
}

func restrictedBrowserToken(current windows.Token) (windows.Token, error) {
	// Restrict our own primary token: CreateProcessAsUser does not require
	// SeAssignPrimaryTokenPrivilege for a restricted version of the caller's token.
	var level windows.Handle
	result, _, err := saferCreateLevel.Call(saferScopeUser, saferLevelNormalUser, saferLevelOpen, uintptr(unsafe.Pointer(&level)), 0)
	if result == 0 {
		return 0, fmt.Errorf("prepare normal-user browser restrictions: %w", err)
	}
	defer saferCloseLevel.Call(uintptr(level))
	var restricted windows.Token
	result, _, err = saferComputeTokenFromLevel.Call(uintptr(level), uintptr(current), uintptr(unsafe.Pointer(&restricted)), 0, 0)
	if result == 0 {
		return 0, fmt.Errorf("restrict browser launch token: %w", err)
	}
	if err := setBrowserTokenIntegrity(restricted); err != nil {
		restricted.Close()
		return 0, err
	}
	return restricted, nil
}

func setBrowserTokenIntegrity(token windows.Token) error {
	medium, err := windows.CreateWellKnownSid(windows.WinMediumLabelSid)
	if err != nil {
		return fmt.Errorf("prepare browser integrity label: %w", err)
	}
	label := windows.Tokenmandatorylabel{Label: windows.SIDAndAttributes{Sid: medium, Attributes: windows.SE_GROUP_INTEGRITY}}
	if err := windows.SetTokenInformation(token, windows.TokenIntegrityLevel, (*byte)(unsafe.Pointer(&label)), label.Size()); err != nil {
		return fmt.Errorf("set browser medium integrity: %w", err)
	}
	return nil
}
