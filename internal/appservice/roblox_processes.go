package appservice

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/sleaze5/RobloxAccountManager/internal/platform/robloxmulti"
)

func (service *Service) GetRobloxProcesses() (robloxmulti.ProcessSnapshot, error) {
	state, err := robloxmulti.ListProcesses()
	if err != nil {
		service.logger.Warn("could not list Roblox processes", "operation", "roblox-processes", "error", err)
		return state, errors.New("could not list Roblox processes; try again")
	}
	return state, nil
}

func (service *Service) KillRobloxProcesses(ctx context.Context, processes []robloxmulti.Process) error {
	var err error
	if service.shouldUseElevatedHelper() {
		_, err = service.elevatedHelper.Run(ctx, robloxmulti.KillProcessesTaskName, robloxmulti.ProcessArguments(processes), func(json.RawMessage) {})
	} else {
		err = robloxmulti.KillProcesses(ctx, processes, service.logger)
	}
	if err != nil {
		service.logger.Warn("could not close selected Roblox processes", "operation", "roblox-processes-close", "error", err)
		return robloxClientError("roblox-processes-close", elevationMessage(err, "Could not close some Roblox processes. Try again."), err)
	}
	return nil
}
