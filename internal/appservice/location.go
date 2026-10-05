package appservice

import (
	"errors"
	"log/slog"
	"sync"

	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
	"github.com/sleaze5/RobloxAccountManager/internal/logging"
)

type Relaunch func(appdata.Mode) error

type Location struct {
	mu       sync.Mutex
	state    appdata.Location
	paths    appdata.Paths
	launch   *logging.Launch
	relaunch Relaunch
	logger   *slog.Logger
}

func NewLocation(state appdata.Location, paths appdata.Paths, launch *logging.Launch, relaunch Relaunch, logger *slog.Logger) *Location {
	return &Location{state: state, paths: paths, launch: launch, relaunch: relaunch, logger: logger}
}

func (location *Location) State() appdata.Location {
	location.mu.Lock()
	defer location.mu.Unlock()
	return location.state
}

// Choosing the other root restarts the application, because settings, the
// vault, and the browser runtime were opened for the provisional root.
func (location *Location) Confirm(mode appdata.Mode) (bool, error) {
	location.mu.Lock()
	defer location.mu.Unlock()
	if location.state.Initialized {
		return false, nil
	}
	candidate, ok := location.state.Candidate(mode)
	if !ok {
		return false, errors.New("this storage option is unavailable")
	}
	restart := mode != location.state.Mode
	paths := location.paths
	if restart {
		var err error
		if paths, err = appdata.At(candidate.Directory); err != nil {
			location.logger.Error("data folder rejected", "operation", "location-setup", "mode", mode, "error", err)
			return false, errors.New("the storage folder cannot be on a network drive")
		}
	}
	if err := paths.Prepare(); err != nil {
		location.logger.Error("data folder setup failed", "operation", "location-setup", "mode", mode, "error", err)
		return false, errors.New("the storage folder could not be created")
	}
	if err := location.launch.Release(candidate.Directory); err != nil {
		location.logger.Error("application log setup failed", "operation", "location-setup", "mode", mode, "error", err)
		return false, errors.New("the logs folder could not be created")
	}
	location.logger.Info("data folder set up", "operation", "location-setup", "mode", mode,
		"choice", location.state.Choice, "state", candidate.State, "other_items", candidate.OtherItemCount, "restart", restart)
	if restart {
		if err := location.relaunch(mode); err != nil {
			location.logger.Error("application restart failed", "operation", "location-setup", "mode", mode, "error", err)
			return false, errors.New("the app could not restart. Open it again")
		}
		return true, nil
	}
	location.state.Initialized, location.state.Choice = true, appdata.ChoiceNone
	return false, nil
}
