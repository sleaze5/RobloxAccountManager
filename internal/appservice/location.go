package appservice

import (
	"errors"
	"log/slog"
	"sync"

	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
	"github.com/sleaze5/RobloxAccountManager/internal/logging"
)

// Location holds the first-launch confirmation of the application directory. Until
// Confirm succeeds, the application writes no storage or log files there.
type Location struct {
	mu     sync.Mutex
	state  appdata.Location
	paths  appdata.Paths
	launch *logging.Launch
	logger *slog.Logger
}

func NewLocation(state appdata.Location, paths appdata.Paths, launch *logging.Launch, logger *slog.Logger) *Location {
	return &Location{state: state, paths: paths, launch: launch, logger: logger}
}

func (location *Location) State() appdata.Location {
	location.mu.Lock()
	defer location.mu.Unlock()
	return location.state
}

func (location *Location) Confirm() error {
	location.mu.Lock()
	defer location.mu.Unlock()
	if location.state.Initialized {
		return nil
	}
	if err := location.paths.Prepare(); err != nil {
		location.logger.Error("application folder setup failed", "operation", "location-setup", "error", err)
		return errors.New("the storage folder could not be created")
	}
	if err := location.launch.Release(); err != nil {
		location.logger.Error("application log setup failed", "operation", "location-setup", "error", err)
		return errors.New("the logs folder could not be created")
	}
	location.logger.Info("application folder set up", "operation", "location-setup", "other_items", location.state.OtherItemCount)
	location.state = appdata.Location{Directory: location.state.Directory, Initialized: true, OtherItems: []string{}}
	return nil
}
