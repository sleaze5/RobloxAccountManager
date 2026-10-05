//go:build darwin

package appdata

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/sleaze5/RobloxAccountManager/internal/appmeta"
)

const portableSupported = false

func StandardRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || !filepath.IsAbs(home) {
		return "", errors.New("home folder is unavailable")
	}
	return filepath.Join(home, "Library", "Application Support", appmeta.Name), nil
}
