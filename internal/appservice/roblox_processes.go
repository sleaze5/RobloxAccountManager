package appservice

import (
	"context"
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
	if err := robloxmulti.KillProcesses(ctx, processes, service.logger); err != nil {
		service.logger.Warn("could not kill selected Roblox processes", "operation", "roblox-processes-kill", "error", err)
		return errors.New("some Roblox processes could not be killed; try again")
	}
	return nil
}
