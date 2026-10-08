package robloxmulti

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/sys/windows"
)

type playerProcesses struct {
	handles []windows.Handle
}

func capturePlayers() (*playerProcesses, error) {
	pids, err := runningPlayerIDs()
	if err != nil {
		return nil, err
	}
	players := &playerProcesses{}
	for _, pid := range pids {
		handle, err := openProcess(pid, windows.PROCESS_TERMINATE, playerName)
		if err != nil {
			players.close()
			return nil, fmt.Errorf("open Roblox process %d for replacement: %w", pid, err)
		}
		if handle != 0 {
			players.handles = append(players.handles, handle)
		}
	}
	return players, nil
}

func openProcess(pid uint32, access uint32, expectedName string) (windows.Handle, error) {
	handle, err := windows.OpenProcess(access|windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, pid)
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if result, err := windows.WaitForSingleObject(handle, 0); err != nil || result == windows.WAIT_OBJECT_0 {
		windows.CloseHandle(handle)
		return 0, err
	}
	if err := verifyProcess(handle, expectedName); err != nil {
		if result, _ := windows.WaitForSingleObject(handle, 0); result == windows.WAIT_OBJECT_0 {
			windows.CloseHandle(handle)
			return 0, nil
		}
		windows.CloseHandle(handle)
		return 0, err
	}
	var session, currentSession uint32
	err = windows.ProcessIdToSessionId(pid, &session)
	if err == nil {
		err = windows.ProcessIdToSessionId(windows.GetCurrentProcessId(), &currentSession)
	}
	if err != nil || session != currentSession {
		windows.CloseHandle(handle)
		if err != nil {
			return 0, err
		}
		return 0, errors.New("roblox process belongs to another Windows session")
	}
	// Retain the verified process object across confirmation. A recycled PID
	// must never authorize terminating an unrelated or newly started process.
	return handle, nil
}

func (players *playerProcesses) count() int { return len(players.handles) }

func (players *playerProcesses) close() {
	for _, handle := range players.handles {
		windows.CloseHandle(handle)
	}
}

func (players *playerProcesses) terminate(ctx context.Context, logger *slog.Logger) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for _, handle := range players.handles {
		if err := ctx.Err(); err != nil {
			return err
		}
		result, err := windows.WaitForSingleObject(handle, 0)
		if err != nil {
			return err
		}
		if result == windows.WAIT_OBJECT_0 {
			continue
		}
		if err := windows.TerminateProcess(handle, 0); err != nil {
			if result, _ := windows.WaitForSingleObject(handle, 0); result != windows.WAIT_OBJECT_0 {
				return err
			}
		}
		pid, err := windows.GetProcessId(handle)
		if err != nil {
			return err
		}
		logger.Info("closed Roblox process after confirmation", "process_id", pid)
	}
	for _, handle := range players.handles {
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			// Wait 50 ms.
			result, err := windows.WaitForSingleObject(handle, 50)
			if err != nil {
				return err
			}
			if result == windows.WAIT_OBJECT_0 {
				break
			}
		}
	}
	return nil
}
