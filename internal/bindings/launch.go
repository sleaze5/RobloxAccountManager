package bindings

import (
	"context"
	"errors"

	"github.com/sleaze5/RobloxAccountManager/internal/appservice"
	"github.com/sleaze5/RobloxAccountManager/internal/gamelaunch"
	"github.com/sleaze5/RobloxAccountManager/internal/platform/robloxmulti"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func (service *Service) GetMultiInstanceState() robloxmulti.Snapshot {
	return service.core.GetMultiInstanceState()
}

func (service *Service) GetRobloxProcesses() (robloxmulti.ProcessSnapshot, error) {
	return service.core.GetRobloxProcesses()
}

func (service *Service) KillRobloxProcesses(ctx context.Context, processes []robloxmulti.Process) error {
	for _, process := range processes {
		if process.PID == 0 || process.StartTime == "" || process.Name == "" {
			return errors.New("refresh the Roblox process list and try again")
		}
	}
	return service.core.KillRobloxProcesses(ctx, processes)
}

func (service *Service) GetLaunchConfirmation() appservice.LaunchConfirmation {
	return service.core.GetLaunchConfirmation()
}

func (service *Service) ResolveLaunchConfirmation(confirmationID string, approved bool) {
	service.core.ResolveLaunchConfirmation(confirmationID, approved)
}

func (service *Service) SetMultiInstanceEnabled(enabled bool) (robloxmulti.Snapshot, error) {
	return service.core.SetMultiInstanceEnabled(enabled)
}

func (service *Service) GetRobloxPlayers() appservice.RobloxPlayers {
	return service.core.GetRobloxPlayers()
}

func (service *Service) CheckRobloxPlayerPath(path string) appservice.RobloxPlayerPathCheck {
	return service.core.CheckRobloxPlayerPath(path)
}

func (service *Service) SetRobloxPlayerPath(path string) error {
	return service.core.SetRobloxPlayerPath(path)
}

func (service *Service) GetRobloxClients() gamelaunch.ClientState {
	return service.core.GetRobloxClients()
}

func (service *Service) SetLinuxClient(client string) (gamelaunch.ClientState, error) {
	return service.core.SetLinuxClient(client)
}

func (service *Service) LaunchAccounts(ctx context.Context, accountIDs []int64, input gamelaunch.Input) (appservice.LaunchResult, error) {
	if err := accountSelection(accountIDs, "game-launch"); err != nil {
		return appservice.LaunchResult{}, err
	}
	return service.core.LaunchAccounts(ctx, accountIDs, input)
}

func (service *Service) CopyLaunchOptions(ctx context.Context, accountIDs []int64, input gamelaunch.Input) error {
	if err := accountSelection(accountIDs, "copy-launch-options"); err != nil {
		return err
	}
	app := application.Get()
	if app == nil {
		return clipboardUnavailable()
	}
	return service.core.CopyLaunchOptions(ctx, accountIDs, input, app.Clipboard.SetText)
}
