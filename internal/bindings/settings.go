package bindings

import (
	"github.com/sleaze5/RobloxAccountManager/internal/appservice"
	"github.com/sleaze5/RobloxAccountManager/internal/appsettings"
	"github.com/sleaze5/RobloxAccountManager/internal/timestampformat"
)

func (service *Service) GetAppSettings() (appservice.AppSettingsState, error) {
	return service.core.GetAppSettings()
}

func (service *Service) SetLoggingLevelEnabled(level string, enabled bool) error {
	return service.core.SetLoggingLevelEnabled(level, enabled)
}

func (service *Service) SetAllLoggingLevelsEnabled(enabled bool) error {
	return service.core.SetAllLoggingLevelsEnabled(enabled)
}

func (service *Service) SetRoValraEnabled(enabled bool) error {
	return service.core.SetRoValraEnabled(enabled)
}

func (service *Service) SetRBXServersEnabled(enabled bool) error {
	return service.core.SetRBXServersEnabled(enabled)
}

func (service *Service) ParseTimestampFormat(source string) (timestampformat.Format, error) {
	return timestampformat.Parse(source)
}

func (service *Service) SetTimestampFormats(timestampFormat, timestampHoverFormat string) (appservice.TimestampFormats, error) {
	return service.core.SetTimestampFormats(timestampFormat, timestampHoverFormat)
}

func (service *Service) SetMotion(motion appsettings.MotionPreference) error {
	return service.core.SetMotion(motion)
}

func (service *Service) SetPresenceUpdates(scope appsettings.PresenceScope, enabled bool, intervalSeconds int) error {
	return service.core.SetPresenceUpdates(scope, enabled, intervalSeconds)
}
