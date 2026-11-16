//go:build windows

package appdata

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// DownloadsDirectory is where a save dialog for a download opens: the
// Desktop, which Windows can move, such as into OneDrive. It falls back to the
// home folder.
func DownloadsDirectory() string {
	if desktop, err := windows.KnownFolderPath(windows.FOLDERID_Desktop, 0); err == nil && filepath.IsAbs(desktop) {
		return desktop
	}
	home, _ := os.UserHomeDir()
	return home
}
