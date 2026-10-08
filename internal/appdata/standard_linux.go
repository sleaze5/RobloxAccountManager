//go:build linux

package appdata

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/sleaze5/RobloxAccountManager/internal/appmeta"
)

const portableSupported = true

func StandardRoot() (string, error) {
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" && filepath.IsAbs(dataHome) {
		return filepath.Join(dataHome, appmeta.Name), nil
	}
	home, err := os.UserHomeDir()
	if err != nil || !filepath.IsAbs(home) {
		return "", errors.New("home folder is unavailable")
	}
	return filepath.Join(home, ".local", "share", appmeta.Name), nil
}
