//go:build windows || darwin

package robloxmulti

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"
)

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
