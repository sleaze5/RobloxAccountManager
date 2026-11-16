//go:build darwin

package appdata

import (
	"os"
	"path/filepath"
)

// DownloadsDirectory is where a save dialog for a download opens: the
// Downloads folder, or the home folder when it is missing.
func DownloadsDirectory() string {
	home, _ := os.UserHomeDir()
	downloads := filepath.Join(home, "Downloads")
	if info, err := os.Stat(downloads); err == nil && info.IsDir() {
		return downloads
	}
	return home
}
