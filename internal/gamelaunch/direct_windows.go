//go:build windows

package gamelaunch

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"
)

func preparePlayerDirectly(ticket, browserTrackerID string, request Request) (preparedLaunch, error) {
	if err := validateTicket(ticket); err != nil {
		return preparedLaunch{}, err
	}
	joinURL, err := buildPlaceLauncherURL(browserTrackerID, request)
	if err != nil {
		return preparedLaunch{}, err
	}
	executable, err := findRobloxPlayer()
	if err != nil {
		return preparedLaunch{}, err
	}
	return preparedLaunch{executable: executable, arguments: []string{"--app", "-t", ticket, "-j", joinURL}}, nil
}

func findRobloxPlayer() (string, error) {
	for _, protocol := range []string{"roblox-player", "roblox"} {
		if executable := registeredPlayer(protocol); executable != "" {
			return executable, nil
		}
	}
	var newest time.Time
	var executable string
	for _, root := range robloxVersionDirectories() {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "version-") {
				continue
			}
			candidate := filepath.Join(root, entry.Name(), "RobloxPlayerBeta.exe")
			if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() && info.ModTime().After(newest) {
				newest, executable = info.ModTime(), candidate
			}
		}
	}
	if executable == "" {
		return "", fmt.Errorf("installed Roblox Player was not found")
	}
	return executable, nil
}

func registeredPlayer(protocol string) string {
	key, err := registry.OpenKey(registry.CLASSES_ROOT, protocol+`\DefaultIcon`, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer key.Close()
	icon, _, err := key.GetStringValue("")
	if err != nil {
		return ""
	}
	icon = strings.TrimSpace(icon)
	if index := strings.LastIndex(icon, ","); index >= 0 {
		if _, err := strconv.Atoi(strings.TrimSpace(icon[index+1:])); err == nil {
			icon = icon[:index]
		}
	}
	icon = strings.Trim(strings.TrimSpace(icon), `"`)
	if !filepath.IsAbs(icon) {
		return ""
	}
	candidate := filepath.Join(filepath.Dir(icon), "RobloxPlayerBeta.exe")
	if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
		return candidate
	}
	return ""
}

func robloxVersionDirectories() []string {
	var directories []string
	if local := os.Getenv("LOCALAPPDATA"); filepath.IsAbs(local) {
		for _, folder := range []string{"Roblox", "Bloxstrap", "Fishstrap", "Voidstrap"} {
			directories = append(directories, filepath.Join(local, folder, "Versions"))
		}
	}
	for _, variable := range []string{"ProgramFiles", "ProgramFiles(x86)"} {
		if root := os.Getenv(variable); filepath.IsAbs(root) {
			directories = append(directories, filepath.Join(root, "Roblox", "Versions"))
		}
	}
	return directories
}
