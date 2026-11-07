//go:build unix

package appdata

import (
	"fmt"
	"os"
	"path/filepath"
)

func ReplaceFile(source, destination string) error {
	if err := os.Rename(source, destination); err != nil {
		return fmt.Errorf("replace %q: %w", filepath.Base(destination), err)
	}
	return nil
}

// The owner-only permission bits are the access list on Unix.
func restrictACL(string, bool) error { return nil }
