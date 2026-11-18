package appservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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
	DeletedCount int `json:"deletedCount"`
	SkippedCount int `json:"skippedCount"`
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
		return state, robloxClientError("clean-before-launch", "Turn off multi-instance first.", nil)
	}
	if err := service.settings.SetCleanBeforeLaunch(enabled); err != nil {
		service.logger.Warn("could not save Roblox settings", "operation", "clean-before-launch", "error", err)
		return state, robloxClientError("clean-before-launch", "Could not save the setting. Try again.", err)
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
		return service.GetRobloxCleanerState(), robloxClientError("delete-logs-when-cleaning", "Could not save the setting. Try again.", err)
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
		return service.GetRobloxCleanerState(), robloxClientError("administrator-rights", "Could not save the setting. Try again.", err)
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
	outcome, err := service.runRobloxCleaning(ctx, robloxcleaner.CleanRequest{
		Scope:                scope,
		ShouldCloseProcesses: true,
		ShouldDeleteLogs:     service.settings.Roblox().DeleteLogsWhenCleaning,
	}, service.events.robloxCleanProgressed)
	if err != nil {
		logger.Warn("could not clean Roblox", "error", err)
		return RobloxCleanReport{}, robloxClientError("roblox-clean", elevationMessage(err, "Could not clean Roblox. Try again."), err)
	}
	logger.Info("Roblox cleaned", "deleted_count", outcome.DeletedCount, "skipped_count", outcome.SkippedCount)
	return RobloxCleanReport{DeletedCount: outcome.DeletedCount, SkippedCount: outcome.SkippedCount}, nil
}

func (service *Service) shouldCleanBeforeLaunch() bool {
	return robloxcleaner.Supported && service.settings.Roblox().CleanBeforeLaunch && !service.multiInstance.Snapshot().Enabled
}

func (service *Service) cleanBeforeLaunch(ctx context.Context) error {
	// Running games were closed after confirmation. Other Roblox processes,
	// such as Roblox Studio, keep running, and the files they hold are skipped.
	outcome, err := service.runRobloxCleaning(ctx, robloxcleaner.CleanRequest{
		Scope:            robloxcleaner.ScopeData,
		ShouldDeleteLogs: service.settings.Roblox().DeleteLogsWhenCleaning,
	}, func(robloxcleaner.Progress) {})
	if err != nil {
		service.logger.Warn("could not clean Roblox before launch", "operation", "clean-before-launch", "error", err)
		return robloxClientError("game-launch", elevationMessage(err, "Could not clean Roblox before launch. Try again."), err)
	}
	if outcome.SkippedCount > 0 {
		service.logger.Warn("some Roblox items were in use or need administrator rights and were not cleaned", "operation", "clean-before-launch", "skipped_count", outcome.SkippedCount)
	}
	return nil
}

func (service *Service) runRobloxCleaning(ctx context.Context, request robloxcleaner.CleanRequest, reportProgress func(robloxcleaner.Progress)) (robloxcleaner.Outcome, error) {
	if !service.shouldUseElevatedHelper() {
		return robloxcleaner.Clean(ctx, request, reportProgress, service.logger)
	}
	encodedOutcome, err := service.elevatedHelper.Run(ctx, robloxcleaner.CleanTaskName, request.Arguments(), func(encodedProgress json.RawMessage) {
		var progress robloxcleaner.Progress
		if json.Unmarshal(encodedProgress, &progress) == nil {
			reportProgress(progress)
		}
	})
	if err != nil {
		return robloxcleaner.Outcome{}, err
	}
	var outcome robloxcleaner.Outcome
	if err := json.Unmarshal(encodedOutcome, &outcome); err != nil {
		return robloxcleaner.Outcome{}, fmt.Errorf("decode the elevated cleaning outcome: %w", err)
	}
	return outcome, nil
}

// shouldUseElevatedHelper reports whether Roblox processes are closed and
// Roblox is cleaned in the elevated helper. An elevated application does it itself.
func (service *Service) shouldUseElevatedHelper() bool {
	return elevation.Supported && service.settings.Roblox().UseAdministratorRights && !elevation.IsElevated()
}

func elevationMessage(err error, fallback string) string {
	switch {
	case errors.Is(err, elevation.ErrDeclined):
		return "Administrator rights were declined. Try again or turn them off."
	case errors.Is(err, elevation.ErrOtherUser):
		return "Use your own Windows account for administrator rights."
	default:
		return fallback
	}
}

func robloxClientError(endpoint, message string, cause error) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: message, Cause: cause}
}
