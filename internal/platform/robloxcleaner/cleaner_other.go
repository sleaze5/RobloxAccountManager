//go:build !windows

package robloxcleaner

import (
	"context"
	"errors"
	"log/slog"
)

const Supported = false

func clean(context.Context, Scope, bool, func(Progress), *slog.Logger) (Outcome, error) {
	return Outcome{}, errors.New("cleaning Roblox is only available on Windows")
}
