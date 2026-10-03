// Package robloxmulti owns Roblox's session-local multi-instance support.
package robloxmulti

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

var ErrLaunchCancelled = errors.New("game launch cancelled")

type Snapshot struct {
	Supported bool   `json:"supported"`
	Enabled   bool   `json:"enabled"`
	Ready     bool   `json:"ready"`
	Message   string `json:"message"`
}

type Manager struct {
	mu               sync.Mutex
	logger           *slog.Logger
	enabled          bool
	closed           bool
	holder           *mutexHolder
	startErr         error
	sweepStop        chan struct{}
	sweepUntil       time.Time
	lastCleanupError string
}

func New(logger *slog.Logger) *Manager {
	return &Manager{logger: logger}
}

func (manager *Manager) SetEnabled(enabled bool) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if !supported || manager.closed || manager.enabled == enabled {
		return
	}
	manager.enabled = enabled
	if !enabled {
		manager.stopLocked()
		manager.logger.Info("multi-instance disabled")
		return
	}
	manager.holder, manager.startErr = newMutexHolder(manager.logger)
	if manager.startErr != nil {
		manager.logger.Warn("could not start multi-instance", "error", manager.startErr)
		return
	}
	manager.logger.Info("multi-instance enabled")
	manager.cleanupLocked()
	manager.scheduleLocked()
}

func (manager *Manager) Snapshot() Snapshot {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	return manager.snapshotLocked()
}

// PrepareForLaunch prevents a new launch from replacing a running client when
// the requested multi-instance protection could not be established.
func (manager *Manager) PrepareForLaunch(ctx context.Context, confirm func(context.Context, int, bool) (bool, error)) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	for {
		if err := manager.checkLaunchLocked(ctx); err != nil {
			return err
		}
		if manager.enabled && (manager.snapshotLocked().Ready || manager.holder == nil || manager.holder.failed.Load()) {
			return manager.prepareMultiInstanceLocked(ctx)
		}
		players, err := capturePlayers()
		if err != nil {
			manager.logger.Warn("could not inspect Roblox before launch", "error", err)
			return errors.New("RAM could not access a running Roblox game. Close Roblox, then try joining again")
		}
		if players.count() == 0 {
			players.close()
			return manager.prepareMultiInstanceLocked(ctx)
		}
		if err := manager.replacePlayersLocked(ctx, players, confirm); err != nil {
			return err
		}
		// Check again after closure so a newly started process is never killed
		// under consent for the previous set of process objects.
	}
}

func (manager *Manager) replacePlayersLocked(ctx context.Context, players *playerProcesses, confirm func(context.Context, int, bool) (bool, error)) error {
	defer players.close()
	multiInstance := manager.enabled
	// Keep runtime status reads and shutdown responsive while awaiting a decision.
	manager.mu.Unlock()
	approved, err := confirm(ctx, players.count(), multiInstance)
	manager.mu.Lock()
	if err != nil {
		return err
	}
	if !approved {
		return ErrLaunchCancelled
	}
	if err := manager.checkLaunchLocked(ctx); err != nil {
		return err
	}
	if manager.enabled && manager.snapshotLocked().Ready {
		return nil
	}
	if err := players.terminate(ctx, manager.logger); err != nil {
		manager.logger.Warn("could not close confirmed Roblox processes", "error", err)
		return errors.New("close Roblox manually, then try joining again")
	}
	return nil
}

func (manager *Manager) checkLaunchLocked(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if manager.closed {
		return errors.New("RAM is closing; reopen it before joining a game")
	}
	return nil
}

func (manager *Manager) prepareMultiInstanceLocked(ctx context.Context) error {
	deadline := time.Now().Add(3 * time.Second)
	for {
		if err := manager.checkLaunchLocked(ctx); err != nil {
			return err
		}
		if !manager.enabled {
			return nil
		}
		state := manager.snapshotLocked()
		if manager.holder == nil || manager.holder.failed.Load() {
			return errors.New(state.Message)
		}
		if state.Ready {
			if present, err := manager.cleanupLocked(); err == nil && !present {
				return nil
			}
		}
		if time.Now().After(deadline) {
			if !state.Ready {
				return errors.New("close other Roblox launchers, then try joining again")
			}
			return errors.New("could not prepare Roblox for another game. Try joining again")
		}
		manager.mu.Unlock()
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
		case <-timer.C:
		}
		timer.Stop()
		manager.mu.Lock()
	}
}

func (manager *Manager) AfterLaunch() {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if manager.enabled && !manager.closed {
		manager.scheduleLocked()
	}
}

func (manager *Manager) Close() {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	manager.closed = true
	manager.enabled = false
	manager.stopLocked()
}

func (manager *Manager) stopLocked() {
	if manager.sweepStop != nil {
		close(manager.sweepStop)
		manager.sweepStop = nil
	}
	if manager.holder != nil {
		manager.holder.close()
		manager.holder = nil
	}
	manager.startErr = nil
}

func (manager *Manager) snapshotLocked() Snapshot {
	state := Snapshot{Supported: supported, Enabled: manager.enabled}
	if !manager.enabled {
		return state
	}
	if manager.startErr != nil || manager.holder == nil || manager.holder.failed.Load() {
		state.Message = "turn multi-instance off and on to retry"
		return state
	}
	if !manager.holder.owned.Load() {
		state.Message = "you must close all Roblox processes first"
		return state
	}
	state.Ready = true
	return state
}

func (manager *Manager) cleanupLocked() (bool, error) {
	present, err := clearSingletonEvents(manager.logger)
	message := ""
	if err != nil {
		message = err.Error()
	}
	if message != manager.lastCleanupError {
		manager.lastCleanupError = message
		if err != nil {
			manager.logger.Warn("could not clear Roblox singleton state", "error", err)
		}
	}
	return present, err
}

func (manager *Manager) scheduleLocked() {
	manager.sweepUntil = time.Now().Add(45 * time.Second)
	if manager.sweepStop != nil {
		return
	}
	stop := make(chan struct{})
	manager.sweepStop = stop
	go manager.sweep(stop)
}

func (manager *Manager) sweep(stop chan struct{}) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			manager.mu.Lock()
			if manager.sweepStop != stop {
				manager.mu.Unlock()
				return
			}
			if time.Now().After(manager.sweepUntil) {
				manager.sweepStop = nil
				manager.mu.Unlock()
				return
			}
			manager.cleanupLocked()
			manager.mu.Unlock()
		}
	}
}
