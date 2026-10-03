package robloxmulti

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"time"

	"golang.org/x/sys/windows"
)

var managedProcessNames = []string{
	playerName,
	"RobloxPlayerLauncher.exe",
	"RobloxPlayerInstaller.exe",
	"RobloxCrashHandler.exe",
	"RobloxStudioBeta.exe",
	"RobloxStudio.exe",
}

func ListProcesses() (ProcessSnapshot, error) {
	state := ProcessSnapshot{Supported: true, Processes: []Process{}}
	processes, err := runningProcesses(managedProcessNames)
	if err != nil {
		return state, err
	}
	for _, process := range processes {
		handle, err := openProcess(process.PID, 0, process.Name)
		if err != nil {
			return state, fmt.Errorf("inspect Roblox process %d: %w", process.PID, err)
		}
		if handle == 0 {
			continue
		}
		startTime, err := processStartTime(handle)
		windows.CloseHandle(handle)
		if err != nil {
			return state, fmt.Errorf("read Roblox process %d creation time: %w", process.PID, err)
		}
		process.StartTime = startTime
		state.Processes = append(state.Processes, process)
	}
	slices.SortFunc(state.Processes, func(left, right Process) int {
		return cmp.Compare(left.PID, right.PID)
	})
	return state, nil
}

func KillProcesses(ctx context.Context, processes []Process, logger *slog.Logger) error {
	for _, process := range processes {
		if process.PID == 0 || process.StartTime == "" || !slices.Contains(managedProcessNames, process.Name) {
			return errors.New("invalid Roblox process selection")
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var killErr error
	for _, process := range processes {
		if err := ctx.Err(); err != nil {
			return errors.Join(killErr, err)
		}
		if err := killProcess(ctx, process, logger); err != nil {
			killErr = errors.Join(killErr, fmt.Errorf("kill Roblox process %d: %w", process.PID, err))
		}
	}
	return killErr
}

func killProcess(ctx context.Context, process Process, logger *slog.Logger) error {
	handle, err := openProcess(process.PID, windows.PROCESS_TERMINATE, process.Name)
	if err != nil || handle == 0 {
		return err
	}
	defer windows.CloseHandle(handle)
	startTime, err := processStartTime(handle)
	if err != nil {
		return err
	}
	// Selection applies to the listed process object, not a recycled PID.
	if startTime != process.StartTime {
		return nil
	}
	players := playerProcesses{handles: []windows.Handle{handle}}
	return players.terminate(ctx, logger)
}

func processStartTime(handle windows.Handle) (string, error) {
	var created, exited, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(handle, &created, &exited, &kernel, &user); err != nil {
		return "", err
	}
	return strconv.FormatUint(uint64(created.HighDateTime)<<32|uint64(created.LowDateTime), 10), nil
}
