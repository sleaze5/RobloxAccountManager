//go:build windows

package appdata

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/sleaze5/RobloxAccountManager/internal/appmeta"
	"golang.org/x/sys/windows"
)

const portableSupported = true

func StandardRoot() (string, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" || !filepath.IsAbs(localAppData) {
		folder, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppData, 0)
		if err != nil {
			return "", err
		}
		localAppData = folder
	}
	if !filepath.IsAbs(localAppData) {
		return "", errors.New("local application data folder is unavailable")
	}
	return filepath.Join(localAppData, appmeta.Name), nil
}
