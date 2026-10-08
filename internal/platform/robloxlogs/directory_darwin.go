//go:build darwin

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
	return []string{filepath.Join(home, "Library", "Logs", "Roblox")}, nil
}
