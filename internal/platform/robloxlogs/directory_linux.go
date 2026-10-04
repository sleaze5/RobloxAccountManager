//go:build linux

package robloxlogs

import (
	"errors"
	"os"
	"path/filepath"
)

func Directories() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil || !filepath.IsAbs(home) {
		return nil, errors.New("home folder is unavailable")
	}
	apps := filepath.Join(home, ".var", "app")
	return []string{
		filepath.Join(apps, "org.vinegarhq.Sober", "data", "sober", "appData", "logs"),
		filepath.Join(apps, "space.bigrat.mocktail", "data", "mocktail", "android", "data", "files", "appData", "logs"),
	}, nil
}
