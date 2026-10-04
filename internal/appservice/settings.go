package appservice

import (
	"fmt"

	"github.com/sleaze5/RobloxAccountManager/internal/appsettings"
	"github.com/sleaze5/RobloxAccountManager/internal/logging"
	"github.com/sleaze5/RobloxAccountManager/internal/timestampformat"
)

type AppSettingsState struct {
	Presence             appsettings.PresenceSettings `json:"presence"`
	EnabledLoggingLevels map[string]bool              `json:"enabledLoggingLevels"`
	TimestampFormats     TimestampFormats             `json:"timestampFormats"`
	Motion               appsettings.MotionPreference `json:"motion"`
	RoValraEnabled       bool                         `json:"roValraEnabled"`
	RoValraRegion        string                       `json:"roValraRegion"`
	SettingsRecovered    bool                         `json:"settingsRecovered"`
}

type TimestampFormats struct {
	Shown timestampformat.Format `json:"shown"`
	Hover timestampformat.Format `json:"hover"`
}

func (service *Service) GetAppSettings() (AppSettingsState, error) {
	loggingSettings := service.settings.Logging()
	userInterfaceSettings := service.settings.UserInterface()
	presenceSettings := service.settings.Presence()
	timestampFormats, err := parseTimestampFormats(userInterfaceSettings.TimestampFormat, userInterfaceSettings.TimestampHoverFormat)
	if err != nil {
		return AppSettingsState{}, err
	}
	return AppSettingsState{
		Presence:             presenceSettings,
		EnabledLoggingLevels: loggingSettings.EnabledLevels,
		TimestampFormats:     timestampFormats,
		Motion:               userInterfaceSettings.Motion,
		RoValraEnabled:       service.settings.Integrations().RoValra,
		RoValraRegion:        service.settings.Integrations().RoValraRegion,
		SettingsRecovered:    service.settings.RecoveryError() != nil,
	}, nil
}

func (service *Service) SetPresenceUpdates(scope appsettings.PresenceScope, enabled bool, intervalSeconds int) error {
	if err := service.settings.SetPresenceUpdates(scope, enabled, intervalSeconds); err != nil {
		service.logger.Warn("could not save presence settings", "operation", "presence-settings", "error", err)
		return err
	}
	service.logger.Debug("presence settings saved", "operation", "presence-settings", "scope", scope, "enabled", enabled, "interval_seconds", intervalSeconds)
	return nil
}

func (service *Service) SetRoValraEnabled(enabled bool) error {
	if err := service.settings.SetRoValraEnabled(enabled); err != nil {
		service.logger.Warn("could not save integration settings", "operation", "integration-settings", "integration", "rovalra", "error", err)
		return err
	}
	service.logger.Info("integration settings saved", "operation", "integration-settings", "integration", "rovalra", "enabled", enabled)
	return nil
}

func (service *Service) SetLoggingLevelEnabled(level string, enabled bool) error {
	if err := service.settings.SetLoggingLevelEnabled(level, enabled); err != nil {
		return err
	}
	return service.applyLoggingSettings()
}

func (service *Service) SetAllLoggingLevelsEnabled(enabled bool) error {
	if err := service.settings.SetAllLoggingLevelsEnabled(enabled); err != nil {
		return err
	}
	return service.applyLoggingSettings()
}

func (service *Service) applyLoggingSettings() error {
	levels, err := logging.ParseLevels(service.settings.Logging().EnabledLevelNames())
	if err != nil {
		return err
	}
	return service.logs.SetEnabledLevels(levels)
}

func (service *Service) ParseTimestampFormat(source string) (timestampformat.Format, error) {
	return timestampformat.Parse(source)
}

func (service *Service) SetTimestampFormats(timestampFormat, timestampHoverFormat string) (TimestampFormats, error) {
	if err := service.settings.SetTimestampFormats(timestampFormat, timestampHoverFormat); err != nil {
		return TimestampFormats{}, err
	}
	return parseTimestampFormats(timestampFormat, timestampHoverFormat)
}

func parseTimestampFormats(timestampFormat, timestampHoverFormat string) (TimestampFormats, error) {
	shown, err := timestampformat.Parse(timestampFormat)
	if err != nil {
		return TimestampFormats{}, fmt.Errorf("parse timestamp format: %w", err)
	}
	hover, err := timestampformat.Parse(timestampHoverFormat)
	if err != nil {
		return TimestampFormats{}, fmt.Errorf("parse timestamp hover format: %w", err)
	}
	return TimestampFormats{Shown: shown, Hover: hover}, nil
}

func (service *Service) SetMotion(motion appsettings.MotionPreference) error {
	if err := service.settings.SetMotion(motion); err != nil {
		service.logger.Warn("could not save motion setting", "operation", "user-interface-settings", "error", err)
		return err
	}
	service.logger.Debug("motion setting saved", "operation", "user-interface-settings", "motion", motion)
	return nil
}
