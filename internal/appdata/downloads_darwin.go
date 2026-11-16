//go:build darwin

package appdata

import "path/filepath"

func downloadDirectoryCandidates(home string) []string {
	return []string{filepath.Join(home, "Downloads")}
}
