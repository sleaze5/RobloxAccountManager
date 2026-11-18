//go:build darwin

package appdata

import (
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

func rejectNetworkPath(path string) error {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return fmt.Errorf("inspect the app file system: %w", err)
	}
	if stat.Flags&unix.MNT_LOCAL == 0 {
		return errors.New("the data folder cannot be on a network drive")
	}
	return nil
}
