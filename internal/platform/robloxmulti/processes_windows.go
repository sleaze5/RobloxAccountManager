package robloxmulti

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

const playerName = "RobloxPlayerBeta.exe"

var compareObjectHandles = windows.NewLazySystemDLL("kernelbase.dll").NewProc("CompareObjectHandles")

// SYSTEM_HANDLE_TABLE_ENTRY_INFO_EX, used only on the supported Windows AMD64 target.
type systemHandle struct {
	Object           uintptr
	ProcessID        uintptr
	Handle           windows.Handle
	GrantedAccess    uint32
	CreatorBackTrace uint16
	ObjectType       uint16
	Attributes       uint32
	Reserved         uint32
}

func clearSingletonEvents(logger *slog.Logger) (bool, error) {
	event, err := openSingletonEvent()
	if err != nil {
		return true, err
	}
	if event == 0 {
		return false, nil
	}
	players, err := runningPlayerIDs()
	if err == nil {
		err = clearSingletonHandles(players, event, logger)
	}
	windows.CloseHandle(event)
	// Our own reference must be closed before checking whether cleanup removed
	// the event. Keeping it open would preserve the singleton state ourselves.
	event, probeErr := openSingletonEvent()
	present := event != 0 || probeErr != nil
	if event != 0 {
		windows.CloseHandle(event)
	}
	return present, errors.Join(err, probeErr)
}

func runningPlayerIDs() ([]uint32, error) {
	processes, err := runningProcesses([]string{playerName})
	players := make([]uint32, 0, len(processes))
	for _, process := range processes {
		players = append(players, process.PID)
	}
	return players, err
}

func runningProcesses(names []string) ([]Process, error) {
	processes := []Process{}
	var sessionID uint32
	if err := windows.ProcessIdToSessionId(windows.GetCurrentProcessId(), &sessionID); err != nil {
		return processes, fmt.Errorf("get current Windows session: %w", err)
	}
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return processes, fmt.Errorf("list running processes: %w", err)
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		name := windows.UTF16ToString(entry.ExeFile[:])
		nameIndex := slices.IndexFunc(names, func(candidate string) bool { return strings.EqualFold(name, candidate) })
		if nameIndex == -1 {
			continue
		}
		var playerSession uint32
		if sessionErr := windows.ProcessIdToSessionId(entry.ProcessID, &playerSession); sessionErr != nil {
			if errors.Is(sessionErr, windows.ERROR_INVALID_PARAMETER) {
				continue // The process exited after the snapshot.
			}
			return processes, fmt.Errorf("get Roblox session for process %d: %w", entry.ProcessID, sessionErr)
		}
		if playerSession == sessionID {
			processes = append(processes, Process{Name: names[nameIndex], PID: entry.ProcessID})
		}
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return processes, fmt.Errorf("read process list: %w", err)
	}
	return processes, nil
}

func openSingletonEvent() (windows.Handle, error) {
	event, err := windows.OpenEvent(windows.SYNCHRONIZE, false, windows.StringToUTF16Ptr("ROBLOX_singletonEvent"))
	if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("open Roblox singleton event: %w", err)
	}
	return event, nil
}

func clearSingletonHandles(players []uint32, event windows.Handle, logger *slog.Logger) error {
	if err := compareObjectHandles.Find(); err != nil {
		return fmt.Errorf("load handle comparison: %w", err)
	}
	entries, err := systemHandles()
	if err != nil {
		return err
	}
	var eventType uint16
	for _, entry := range entries {
		if entry.ProcessID == uintptr(windows.GetCurrentProcessId()) && entry.Handle == event {
			eventType = entry.ObjectType
			break
		}
	}
	if eventType == 0 {
		return errors.New("singleton event is missing from the handle snapshot")
	}
	byPID := make(map[uint32][]windows.Handle, len(players))
	for _, player := range players {
		byPID[player] = nil
	}
	for _, entry := range entries {
		if _, exists := byPID[uint32(entry.ProcessID)]; exists && entry.ObjectType == eventType {
			byPID[uint32(entry.ProcessID)] = append(byPID[uint32(entry.ProcessID)], entry.Handle)
		}
	}
	var scanErr error
	for _, player := range players {
		if err := clearPlayerSingleton(player, byPID[player], event, logger); err != nil {
			scanErr = errors.Join(scanErr, fmt.Errorf("clear singleton for Roblox process %d: %w", player, err))
		}
	}
	return scanErr
}

func clearPlayerSingleton(playerID uint32, handles []windows.Handle, event windows.Handle, logger *slog.Logger) error {
	process, err := windows.OpenProcess(windows.PROCESS_DUP_HANDLE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, playerID)
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		return nil
	}
	if err != nil {
		return err
	}
	defer windows.CloseHandle(process)
	if err := verifyProcess(process, playerName); err != nil {
		return err
	}
	for _, handle := range handles {
		matched, err := clearEventHandle(process, handle, event)
		if err != nil {
			return err
		}
		if !matched {
			continue
		}
		logger.Info("cleared Roblox singleton event", "process_id", playerID)
	}
	return nil
}

func verifyProcess(process windows.Handle, expectedName string) error {
	var path [32768]uint16
	size := uint32(len(path))
	if err := windows.QueryFullProcessImageName(process, 0, &path[0], &size); err != nil {
		return err
	}
	if !strings.EqualFold(filepath.Base(windows.UTF16ToString(path[:size])), expectedName) {
		return errors.New("process name no longer matches")
	}
	var token windows.Token
	if err := windows.OpenProcessToken(process, windows.TOKEN_QUERY, &token); err != nil {
		return err
	}
	defer token.Close()
	owner, err := token.GetTokenUser()
	if err != nil {
		return err
	}
	current, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	if !owner.User.Sid.Equals(current.User.Sid) {
		return errors.New("roblox process belongs to another Windows user")
	}
	return nil
}

func clearEventHandle(process, handle, event windows.Handle) (bool, error) {
	var duplicate windows.Handle
	if err := windows.DuplicateHandle(process, handle, windows.CurrentProcess(), &duplicate, 0, false, windows.DUPLICATE_SAME_ACCESS); err != nil {
		return false, nil // Handles can disappear while Roblox is starting or exiting.
	}
	defer windows.CloseHandle(duplicate)
	// Compare against the exact named Event in our session. Unlike querying the
	// names of arbitrary remote handles, this cannot block on a synchronous pipe.
	same, _, _ := compareObjectHandles.Call(uintptr(duplicate), uintptr(event))
	if same == 0 {
		return false, nil
	}
	var closer windows.Handle
	err := windows.DuplicateHandle(process, handle, windows.CurrentProcess(), &closer, 0, false, windows.DUPLICATE_CLOSE_SOURCE|windows.DUPLICATE_SAME_ACCESS)
	if closer != 0 {
		windows.CloseHandle(closer)
	}
	if err != nil {
		return true, fmt.Errorf("close Roblox singleton event: %w", err)
	}
	return true, nil
}

func systemHandles() ([]systemHandle, error) {
	for size := uint32(64 * 1024); size <= 64*1024*1024; {
		buffer := make([]byte, size)
		var required uint32
		err := windows.NtQuerySystemInformation(windows.SystemExtendedHandleInformation, unsafe.Pointer(&buffer[0]), size, &required)
		if errors.Is(err, windows.STATUS_INFO_LENGTH_MISMATCH) {
			size = max(size*2, required)
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read system handle snapshot: %w", err)
		}
		headerSize := 2 * unsafe.Sizeof(uintptr(0))
		count := *(*uintptr)(unsafe.Pointer(&buffer[0]))
		if required < uint32(headerSize) || required > size || count > (uintptr(required)-headerSize)/unsafe.Sizeof(systemHandle{}) {
			return nil, errors.New("invalid system handle snapshot size")
		}
		return unsafe.Slice((*systemHandle)(unsafe.Pointer(&buffer[headerSize])), int(count)), nil
	}
	return nil, errors.New("system handle snapshot exceeded the size limit")
}
