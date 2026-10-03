package robloxlogs

import (
	"errors"
	"os"
	"path/filepath"
)

func Directory() (string, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" || !filepath.IsAbs(localAppData) {
		return "", errors.New("local application data folder is unavailable")
	}
	return filepath.Join(localAppData, "Roblox", "logs"), nil
}
