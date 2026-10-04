package robloxlogs

import (
	"errors"
	"os"
	"path/filepath"
)

func Directories() ([]string, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" || !filepath.IsAbs(localAppData) {
		return nil, errors.New("local application data folder is unavailable")
	}
	return []string{filepath.Join(localAppData, "Roblox", "logs")}, nil
}
