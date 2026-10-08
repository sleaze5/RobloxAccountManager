//go:build !windows

package robloxmulti

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
)

const supported = false

type mutexHolder struct {
	owned  atomic.Bool
	failed atomic.Bool
}

func newMutexHolder(*slog.Logger) (*mutexHolder, error) {
	return nil, errors.New("multi-instance is only available on Windows")
}

func (*mutexHolder) close() {}

func clearSingletonEvents(*slog.Logger) (bool, error) {
	return false, nil
}

type playerProcesses struct{}

func capturePlayers() (*playerProcesses, error)                        { return &playerProcesses{}, nil }
func (*playerProcesses) count() int                                    { return 0 }
func (*playerProcesses) close()                                        {}
func (*playerProcesses) terminate(context.Context, *slog.Logger) error { return nil }
