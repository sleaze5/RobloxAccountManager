//go:build !windows

package robloxcleaner

import (
	"context"
	"errors"
	"log/slog"
)

const Supported = false

func clean(context.Context, Scope, bool, *slog.Logger) (bool, error) {
	return false, errors.New("cleaning Roblox is only available on Windows")
}
