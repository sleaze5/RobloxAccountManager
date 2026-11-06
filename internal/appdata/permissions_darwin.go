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
		return fmt.Errorf("inspect application filesystem: %w", err)
	}
	if stat.Flags&unix.MNT_LOCAL == 0 {
		return errors.New("network application paths are unsupported")
	}
	return nil
}
