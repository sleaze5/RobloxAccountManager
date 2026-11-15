package robloxmulti

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
)

type Process struct {
	Name      string `json:"name"`
	PID       uint32 `json:"pid"`
	StartTime string `json:"startTime"`
}

type ProcessSnapshot struct {
	Supported bool      `json:"supported"`
	Processes []Process `json:"processes"`
}

const KillProcessesTaskName = "close-roblox-processes"

// ProcessArguments encodes processes for KillProcessesTask in the elevated
// helper, one "<PID>:<start time>:<name>" argument per process.
func ProcessArguments(processes []Process) []string {
	arguments := make([]string, 0, len(processes))
	for _, process := range processes {
		arguments = append(arguments, strconv.FormatUint(uint64(process.PID), 10)+":"+process.StartTime+":"+process.Name)
	}
	return arguments
}

func parseProcessArguments(arguments []string) ([]Process, error) {
	processes := make([]Process, 0, len(arguments))
	for _, argument := range arguments {
		parts := strings.SplitN(argument, ":", 3)
		if len(parts) != 3 {
			return nil, errors.New("invalid Roblox process argument")
		}
		pid, err := strconv.ParseUint(parts[0], 10, 32)
		if err != nil {
			return nil, errors.New("invalid Roblox process ID")
		}
		processes = append(processes, Process{PID: uint32(pid), StartTime: parts[1], Name: parts[2]})
	}
	return processes, nil
}

// KillProcessesTask closes processes that ProcessArguments encoded. The
// elevated helper runs it, and KillProcesses verifies each process again.
func KillProcessesTask(ctx context.Context, arguments []string, logger *slog.Logger) (bool, error) {
	processes, err := parseProcessArguments(arguments)
	if err != nil {
		return false, err
	}
	return false, KillProcesses(ctx, processes, logger)
}
