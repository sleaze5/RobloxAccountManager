//go:build linux

package robloxmulti

import (
	"context"
	"errors"
	"log/slog"
)

func ListProcesses() (ProcessSnapshot, error) {
	return ProcessSnapshot{Processes: []Process{}}, nil
}

func KillProcesses(context.Context, []Process, *slog.Logger) error {
	return errors.New("closing Roblox processes is unavailable on Linux")
}
