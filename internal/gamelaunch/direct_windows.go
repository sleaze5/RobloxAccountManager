//go:build windows

package gamelaunch

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

func preparePlayerDirectly(ticket, browserTrackerID string, request Request) (preparedLaunch, error) {
	if err := validateTicket(ticket); err != nil {
		return preparedLaunch{}, err
	}
	joinURL, err := buildPlaceLauncherURL(browserTrackerID, request)
	if err != nil {
		return preparedLaunch{}, err
	}
	executable, err := playerExecutable(request.PlayerPath)
	if err != nil {
		return preparedLaunch{}, err
	}
	return preparedLaunch{executable: executable, arguments: []string{"--app", "-t", ticket, "-j", joinURL}}, nil
}

const playerExecutableName = "RobloxPlayerBeta.exe"

func playerExecutable(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return findRobloxPlayer()
	}
	parsed, found, err := CheckPlayerPath(path)
	if err != nil {
		return "", err
	}
	if !found {
		return "", &ClientError{Message: "the Roblox Player binary was not found at the selected path"}
	}
	return parsed, nil
}

func CheckPlayerPath(path string) (string, bool, error) {
	parsed, err := parsePlayerPath(path)
	if err != nil {
		return "", false, err
	}
	info, err := os.Stat(parsed)
	return parsed, err == nil && info.Mode().IsRegular(), nil
}

func parsePlayerPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	for _, quote := range []string{`"`, `'`} {
		if len(path) >= 2 && strings.HasPrefix(path, quote) && strings.HasSuffix(path, quote) {
			path = strings.TrimSpace(path[1 : len(path)-1])
		}
	}
	expanded, err := expandEnvironment(path)
	if err != nil {
		return "", &ClientError{Message: "the custom path contains characters that are not allowed"}
	}
	path = filepath.Clean(expanded)
	if !filepath.IsAbs(path) {
		return "", &ClientError{Message: "enter a full path to the Roblox Player binary or its folder"}
	}
	if strings.ContainsAny(path[len(filepath.VolumeName(path)):], `<>:"|?*`) {
		return "", &ClientError{Message: "the custom path contains characters that are not allowed"}
	}
	if strings.EqualFold(filepath.Ext(path), ".exe") {
		return path, nil
	}
	return filepath.Join(path, playerExecutableName), nil
}

func expandEnvironment(path string) (string, error) {
	source, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	size, err := windows.ExpandEnvironmentStrings(source, nil, 0)
	if err != nil {
		return "", fmt.Errorf("measure expanded path: %w", err)
	}
	buffer := make([]uint16, size)
	if _, err := windows.ExpandEnvironmentStrings(source, &buffer[0], size); err != nil {
		return "", fmt.Errorf("expand path: %w", err)
	}
	return windows.UTF16ToString(buffer), nil
}

func findRobloxPlayer() (string, error) {
	versions := PlayerVersions()
	if len(versions) == 0 {
		return "", &ClientError{Message: "no Roblox Player binary was found. Use Set location in Additional launch options"}
	}
	return versions[0].Path, nil
}

type installedPlayer struct {
	version  PlayerVersion
	modified time.Time
}

func PlayerVersions() []PlayerVersion {
	var found []installedPlayer
	for _, variable := range []string{"LOCALAPPDATA", "ProgramFiles", "ProgramFiles(x86)"} {
		found = append(found, installedPlayers(variable)...)
	}
	slices.SortFunc(found, func(left, right installedPlayer) int { return right.modified.Compare(left.modified) })
	versions := make([]PlayerVersion, len(found))
	for index, player := range found {
		versions[index] = player.version
	}
	return versions
}

func installedPlayers(variable string) []installedPlayer {
	base := os.Getenv(variable)
	if !filepath.IsAbs(base) {
		return nil
	}
	root := filepath.Join(base, "Roblox", "Versions")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var found []installedPlayer
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "version-") {
			continue
		}
		path := filepath.Join(root, entry.Name(), playerExecutableName)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			found = append(found, installedPlayer{
				version: PlayerVersion{
					Name:        entry.Name(),
					Path:        path,
					DisplayPath: filepath.Join("%"+variable+"%", "Roblox", "Versions", entry.Name(), playerExecutableName),
				},
				modified: info.ModTime(),
			})
		}
	}
	return found
}
