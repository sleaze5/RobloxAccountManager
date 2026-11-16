//go:build windows

package appdata

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// DefaultDownloadDirectory is where a save dialog for a download opens: the
// Desktop, which Windows can move, such as into OneDrive. It falls back to the
// home folder, and is empty when that is unknown too, so that the dialog opens
// in its own default folder.
func DefaultDownloadDirectory() string {
	if desktop, err := windows.KnownFolderPath(windows.FOLDERID_Desktop, 0); err == nil && filepath.IsAbs(desktop) {
		return desktop
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}
