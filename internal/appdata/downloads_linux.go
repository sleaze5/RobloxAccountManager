//go:build linux

package appdata

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// DownloadsDirectory is where a save dialog for a download opens: the XDG
// download folder, then ~/Downloads, then the home folder.
func DownloadsDirectory() string {
	home, _ := os.UserHomeDir()
	for _, downloads := range []string{xdgDownloadDirectory(home), filepath.Join(home, "Downloads")} {
		if info, err := os.Stat(downloads); downloads != "" && err == nil && info.IsDir() {
			return downloads
		}
	}
	return home
}

// xdgDownloadDirectory reads XDG_DOWNLOAD_DIR from user-dirs.dirs, which
// xdg-user-dirs writes as a shell assignment such as
// XDG_DOWNLOAD_DIR="$HOME/Downloads". It is empty when the file names none.
func xdgDownloadDirectory(home string) string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" || !filepath.IsAbs(configHome) {
		configHome = filepath.Join(home, ".config")
	}
	userDirs, err := os.Open(filepath.Join(configHome, "user-dirs.dirs"))
	if err != nil {
		return ""
	}
	defer func() { _ = userDirs.Close() }()
	scanner := bufio.NewScanner(userDirs)
	for scanner.Scan() {
		assignment, isDownloadDirectory := strings.CutPrefix(strings.TrimSpace(scanner.Text()), "XDG_DOWNLOAD_DIR=")
		if !isDownloadDirectory {
			continue
		}
		directory := strings.Trim(assignment, `"`)
		if rest, isHomeRelative := strings.CutPrefix(directory, "$HOME"); isHomeRelative {
			directory = filepath.Join(home, rest)
		}
		if filepath.IsAbs(directory) {
			return directory
		}
	}
	return ""
}
