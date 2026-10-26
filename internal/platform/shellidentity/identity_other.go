//go:build !windows

package shellidentity

import "log/slog"

func New(_ *slog.Logger) (func(uintptr, uint32, uintptr, uintptr) (uintptr, bool), error) {
	return nil, nil
}
