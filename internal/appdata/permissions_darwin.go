//go:build darwin

package appdata

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func rejectNetworkPath(path string) error {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return fmt.Errorf("inspect application filesystem: %w", err)
	}
	if stat.Flags&unix.MNT_LOCAL == 0 {
		return fmt.Errorf("network application paths are unsupported")
	}
	return nil
}

func restrictDirectory(path string) error {
	// Only the owner can read, write, and enter the directory.
	if err := os.Chmod(path, 0o700); err != nil {
		return fmt.Errorf("restrict directory permissions: %w", err)
	}
	return restrictACL(path, true)
}

func ReplaceFile(source, destination string) error {
	if err := os.Rename(source, destination); err != nil {
		return fmt.Errorf("replace %q: %w", filepath.Base(destination), err)
	}
	return nil
}

func restrictACL(path string, directory bool) error {
	// Only the owner has access. Directories also need the execute bit to be entered.
	mode := os.FileMode(0o600)
	if directory {
		mode = 0o700
	}
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("restrict permissions: %w", err)
	}
	return nil
}
