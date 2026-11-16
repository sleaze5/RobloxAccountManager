//go:build linux || darwin

package appdata

import "os"

// DefaultDownloadDirectory is where a save dialog for a download opens: the
// first folder of downloadDirectoryCandidates that exists, or the home folder.
// It is empty when the home folder is unknown, and the dialog then opens in
// its own default folder.
func DefaultDownloadDirectory() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	for _, candidate := range downloadDirectoryCandidates(home) {
		if status, err := os.Stat(candidate); err == nil && status.IsDir() {
			return candidate
		}
	}
	return home
}
