package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/appservice"
	"github.com/sleaze5/RobloxAccountManager/internal/platform/robloxcleaner"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

func (service *Service) GetRobloxCleanerState() appservice.RobloxCleanerState {
	return service.core.GetRobloxCleanerState()
}

func (service *Service) SetCleanBeforeLaunch(enabled bool) (appservice.RobloxCleanerState, error) {
	return service.core.SetCleanBeforeLaunch(enabled)
}

func (service *Service) SetDeleteLogsWhenCleaning(enabled bool) (appservice.RobloxCleanerState, error) {
	return service.core.SetDeleteLogsWhenCleaning(enabled)
}

func (service *Service) SetUseAdministratorRights(enabled bool) (appservice.RobloxCleanerState, error) {
	return service.core.SetUseAdministratorRights(enabled)
}

func (service *Service) CleanRoblox(ctx context.Context, scope robloxcleaner.Scope) (appservice.RobloxCleanReport, error) {
	if !robloxcleaner.ValidScope(scope) {
		return appservice.RobloxCleanReport{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "roblox-clean", Message: "Select what to clean and try again."}
	}
	return service.core.CleanRoblox(ctx, scope)
}
