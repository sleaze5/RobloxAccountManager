package appservice

import (
	"context"
	"errors"

	"github.com/sleaze5/RobloxAccountManager/internal/platform/elevation"
	"github.com/sleaze5/RobloxAccountManager/internal/platform/robloxcleaner"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

type RobloxCleanerState struct {
	Supported         bool `json:"supported"`
	CleanBeforeLaunch bool `json:"cleanBeforeLaunch"`
	// DeleteLogsWhenCleaning deletes Roblox logs, which Logs Explorer reads,
	// together with the other Roblox data.
	DeleteLogsWhenCleaning bool `json:"deleteLogsWhenCleaning"`
	// IsBlockedByMultiInstance is true while multi-instance is on. The saved
	// choice stays, but Roblox is not cleaned before a launch.
	IsBlockedByMultiInstance bool `json:"isBlockedByMultiInstance"`
	ElevationSupported       bool `json:"elevationSupported"`
	UseAdministratorRights   bool `json:"useAdministratorRights"`
	IsElevated               bool `json:"isElevated"`
}

type RobloxCleanReport struct {
	HasSkippedItems bool `json:"hasSkippedItems"`
}

func (service *Service) GetRobloxCleanerState() RobloxCleanerState {
	settings := service.settings.Roblox()
	return RobloxCleanerState{
		Supported:                robloxcleaner.Supported,
		CleanBeforeLaunch:        settings.CleanBeforeLaunch,
		DeleteLogsWhenCleaning:   settings.DeleteLogsWhenCleaning,
		IsBlockedByMultiInstance: service.multiInstance.Snapshot().Enabled,
		ElevationSupported:       elevation.Supported,
		UseAdministratorRights:   settings.UseAdministratorRights,
		IsElevated:               elevation.IsElevated(),
	}
}

func (service *Service) SetCleanBeforeLaunch(enabled bool) (RobloxCleanerState, error) {
	service.launchMutex.Lock()
	defer service.launchMutex.Unlock()
	state := service.GetRobloxCleanerState()
	if !state.Supported {
		return state, robloxClientError("clean-before-launch", "Cleaning Roblox is only available on Windows.", nil)
	}
	if state.IsBlockedByMultiInstance {
		return state, robloxClientError("clean-before-launch", "Turn off multi-instance before you change this setting.", nil)
	}
	if err := service.settings.SetCleanBeforeLaunch(enabled); err != nil {
		service.logger.Warn("could not save Roblox settings", "operation", "clean-before-launch", "error", err)
		return state, robloxClientError("clean-before-launch", "The cleaning setting could not be saved. Try again.", err)
	}
	service.logger.Info("clean before launch saved", "operation", "clean-before-launch", "enabled", enabled)
	return service.GetRobloxCleanerState(), nil
}

func (service *Service) SetDeleteLogsWhenCleaning(enabled bool) (RobloxCleanerState, error) {
	service.launchMutex.Lock()
	defer service.launchMutex.Unlock()
	if !robloxcleaner.Supported {
		return service.GetRobloxCleanerState(), robloxClientError("delete-logs-when-cleaning", "Cleaning Roblox is only available on Windows.", nil)
	}
	if err := service.settings.SetDeleteLogsWhenCleaning(enabled); err != nil {
		service.logger.Warn("could not save Roblox settings", "operation", "delete-logs-when-cleaning", "error", err)
		return service.GetRobloxCleanerState(), robloxClientError("delete-logs-when-cleaning", "The log cleaning setting could not be saved. Try again.", err)
	}
	service.logger.Info("log cleaning setting saved", "operation", "delete-logs-when-cleaning", "enabled", enabled)
	return service.GetRobloxCleanerState(), nil
}

func (service *Service) SetUseAdministratorRights(enabled bool) (RobloxCleanerState, error) {
	service.launchMutex.Lock()
	defer service.launchMutex.Unlock()
	if !elevation.Supported {
		return service.GetRobloxCleanerState(), robloxClientError("administrator-rights", "Administrator rights are only available on Windows.", nil)
	}
	if err := service.settings.SetUseAdministratorRights(enabled); err != nil {
		service.logger.Warn("could not save Roblox settings", "operation", "administrator-rights", "error", err)
		return service.GetRobloxCleanerState(), robloxClientError("administrator-rights", "The administrator rights setting could not be saved. Try again.", err)
	}
	if !enabled {
		service.elevatedHelper.Close()
	}
	service.logger.Info("administrator rights setting saved", "operation", "administrator-rights", "enabled", enabled)
	return service.GetRobloxCleanerState(), nil
}

// CleanRoblox closes every Roblox process, then cleans Roblox. Launches wait
// until it finishes, so a game never starts while its files are deleted.
func (service *Service) CleanRoblox(ctx context.Context, scope robloxcleaner.Scope) (RobloxCleanReport, error) {
	service.launchMutex.Lock()
	defer service.launchMutex.Unlock()
	logger := service.logger.With("operation", "roblox-clean", "scope", scope)
	hasSkippedItems, err := service.runRobloxCleaning(ctx, robloxcleaner.CleanRequest{
		Scope:                scope,
		ShouldCloseProcesses: true,
		ShouldDeleteLogs:     service.settings.Roblox().DeleteLogsWhenCleaning,
	})
	if err != nil {
		logger.Warn("could not clean Roblox", "error", err)
		return RobloxCleanReport{}, robloxClientError("roblox-clean", elevationMessage(err, "Roblox could not be cleaned. Close Roblox and try again."), err)
	}
	logger.Info("Roblox cleaned", "has_skipped_items", hasSkippedItems)
	return RobloxCleanReport{HasSkippedItems: hasSkippedItems}, nil
}

func (service *Service) shouldCleanBeforeLaunch() bool {
	return robloxcleaner.Supported && service.settings.Roblox().CleanBeforeLaunch && !service.multiInstance.Snapshot().Enabled
}

func (service *Service) cleanBeforeLaunch(ctx context.Context) error {
	// Running games were closed after confirmation. Other Roblox processes,
	// such as Roblox Studio, keep running, and the files they hold are skipped.
	hasSkippedItems, err := service.runRobloxCleaning(ctx, robloxcleaner.CleanRequest{
		Scope:            robloxcleaner.ScopeData,
		ShouldDeleteLogs: service.settings.Roblox().DeleteLogsWhenCleaning,
	})
	if err != nil {
		service.logger.Warn("could not clean Roblox before launch", "operation", "clean-before-launch", "error", err)
		return robloxClientError("game-launch", elevationMessage(err, "Roblox could not be cleaned before the launch. Try again, or turn off Clean before launch."), err)
	}
	if hasSkippedItems {
		service.logger.Warn("some Roblox items were in use or need administrator rights and were not cleaned", "operation", "clean-before-launch")
	}
	return nil
}

func (service *Service) runRobloxCleaning(ctx context.Context, request robloxcleaner.CleanRequest) (bool, error) {
	if service.shouldUseElevatedHelper() {
		return service.elevatedHelper.Run(ctx, robloxcleaner.CleanTaskName, request.Arguments())
	}
	return robloxcleaner.Clean(ctx, request, service.logger)
}

// shouldUseElevatedHelper reports whether Roblox processes are closed and
// Roblox is cleaned in the elevated helper. An elevated application does it itself.
func (service *Service) shouldUseElevatedHelper() bool {
	return elevation.Supported && service.settings.Roblox().UseAdministratorRights && !elevation.IsElevated()
}

func elevationMessage(err error, fallback string) string {
	switch {
	case errors.Is(err, elevation.ErrDeclined):
		return "Administrator rights were declined. Allow them, or turn off Use administrator rights."
	case errors.Is(err, elevation.ErrOtherUser):
		return "Administrator rights must come from your own Windows account. Turn off Use administrator rights to continue without them."
	default:
		return fallback
	}
}

func robloxClientError(endpoint, message string, cause error) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: message, Cause: cause}
}
