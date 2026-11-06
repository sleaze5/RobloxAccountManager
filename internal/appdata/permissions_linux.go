//go:build linux

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
	switch stat.Type {
	case unix.NFS_SUPER_MAGIC, unix.CIFS_SUPER_MAGIC, unix.SMB_SUPER_MAGIC, unix.SMB2_SUPER_MAGIC, unix.AFS_SUPER_MAGIC, unix.CODA_SUPER_MAGIC, unix.CEPH_SUPER_MAGIC:
		return errors.New("network application paths are unsupported")
	default:
		return nil
	}
}
