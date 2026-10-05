//go:build darwin

package robloxmulti

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const (
	// The kernel keeps at most 16 bytes of a process name, so longer names
	// are matched by that prefix.
	processNameLimit = 16
	// zombieState is SZOMB from sys/proc.h: the process has exited.
	zombieState = 5
)

var managedProcessNames = []string{
	"RobloxPlayer",
	"RobloxCrashHandler",
	"RobloxStudio",
}

func ListProcesses() (ProcessSnapshot, error) {
	state := ProcessSnapshot{Supported: true, Processes: []Process{}}
	entries, err := unix.SysctlKinfoProcSlice("kern.proc.uid", os.Getuid())
	if err != nil {
		return state, fmt.Errorf("list running processes: %w", err)
	}
	for _, entry := range entries {
		if process, ok := managedProcess(&entry); ok {
			state.Processes = append(state.Processes, process)
		}
	}
	slices.SortFunc(state.Processes, func(left, right Process) int {
		return cmp.Compare(left.PID, right.PID)
	})
	return state, nil
}

func managedProcess(entry *unix.KinfoProc) (Process, bool) {
	name := unix.ByteSliceToString(entry.Proc.P_comm[:])
	if entry.Proc.P_pid <= 0 || name == "" {
		return Process{}, false
	}
	for _, managed := range managedProcessNames {
		if name == managed || len(managed) > processNameLimit && name == managed[:processNameLimit] {
			start := entry.Proc.P_starttime
			return Process{
				Name:      managed,
				PID:       uint32(entry.Proc.P_pid),
				StartTime: strconv.FormatInt(start.Sec*1_000_000+int64(start.Usec), 10),
			}, true
		}
	}
	return Process{}, false
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
	signal := syscall.SIGTERM
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	forceAt := time.Now().Add(2 * time.Second)
	for {
		if !stillRunning(process) {
			return nil
		}
		if err := syscall.Kill(int(process.PID), signal); err != nil {
			if errors.Is(err, syscall.ESRCH) {
				return nil
			}
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		if signal == syscall.SIGTERM && time.Now().After(forceAt) {
			logger.Warn("Roblox process did not quit; forcing it", "operation", "roblox-process-kill", "pid", process.PID, "name", process.Name)
			signal = syscall.SIGKILL
		}
	}
}

func stillRunning(process Process) bool {
	entry, err := unix.SysctlKinfoProc("kern.proc.pid", int(process.PID))
	if err != nil || entry == nil || entry.Proc.P_pid != int32(process.PID) || entry.Proc.P_stat == zombieState {
		return false
	}
	current, ok := managedProcess(entry)
	return ok && current.Name == process.Name && current.StartTime == process.StartTime
}
