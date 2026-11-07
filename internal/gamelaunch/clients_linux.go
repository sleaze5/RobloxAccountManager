//go:build linux

package gamelaunch

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type knownClient struct {
	id      string
	name    string
	desktop string
}

var knownLinuxClients = []knownClient{
	{id: "sober", name: "Sober", desktop: "org.vinegarhq.Sober.desktop"},
	{id: "mocktail", name: "Mocktail", desktop: "space.bigrat.mocktail.desktop"},
}

func listClients() []RobloxClient {
	clients := make([]RobloxClient, 0, len(knownLinuxClients))
	for _, client := range knownLinuxClients {
		_, installed := findDesktop(client.desktop)
		clients = append(clients, RobloxClient{ID: client.id, Name: client.name, Installed: installed})
	}
	return clients
}

func resolveLinuxClient(preferred string) (knownClient, string, error) {
	type found struct {
		client knownClient
		path   string
	}
	installed := make([]found, 0, len(knownLinuxClients))
	byID := map[string]found{}
	for _, client := range knownLinuxClients {
		path, ok := findDesktop(client.desktop)
		if !ok {
			continue
		}
		match := found{client: client, path: path}
		installed = append(installed, match)
		byID[client.id] = match
	}
	if preferred != "" {
		for _, client := range knownLinuxClients {
			if client.id != preferred {
				continue
			}
			match, ok := byID[client.id]
			if !ok {
				return knownClient{}, "", &ClientError{Message: strings.ToLower(client.name) + " is selected, but it is not installed"}
			}
			return match.client, match.path, nil
		}
		return knownClient{}, "", &ClientError{Message: "the selected Roblox client is not supported"}
	}
	switch len(installed) {
	case 0:
		return knownClient{}, "", &ClientError{Message: protocolUnavailable}
	case 1:
		return installed[0].client, installed[0].path, nil
	}
	preferredDesktop := defaultRobloxDesktop()
	for _, match := range installed {
		if match.client.desktop == preferredDesktop {
			return match.client, match.path, nil
		}
	}
	return knownClient{}, "", &ClientError{Message: "sober and Mocktail are both installed; choose one in Settings under Roblox"}
}

func findDesktop(name string) (string, bool) {
	for _, directory := range applicationDirectories() {
		path := filepath.Join(directory, name)
		info, err := os.Stat(path)
		if err == nil && info.Mode().IsRegular() {
			return path, true
		}
	}
	return "", false
}

func applicationDirectories() []string {
	seen := map[string]struct{}{}
	directories := make([]string, 0, 6)
	add := func(directory string) {
		if directory == "" {
			return
		}
		directory = filepath.Clean(directory)
		if _, ok := seen[directory]; ok {
			return
		}
		seen[directory] = struct{}{}
		directories = append(directories, directory)
	}
	home, _ := os.UserHomeDir()
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		add(filepath.Join(dataHome, "applications"))
	} else if home != "" {
		add(filepath.Join(home, ".local", "share", "applications"))
	}
	dataDirectories := os.Getenv("XDG_DATA_DIRS")
	if dataDirectories == "" {
		dataDirectories = "/usr/local/share" + string(filepath.ListSeparator) + "/usr/share"
	}
	for _, directory := range filepath.SplitList(dataDirectories) {
		add(filepath.Join(directory, "applications"))
	}
	if home != "" {
		add(filepath.Join(home, ".local", "share", "flatpak", "exports", "share", "applications"))
	}
	add("/var/lib/flatpak/exports/share/applications")
	return directories
}

func defaultRobloxDesktop() string {
	xdgMimePath, err := exec.LookPath("xdg-mime")
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	defaultClientOutput, err := exec.CommandContext(ctx, xdgMimePath, "query", "default", "x-scheme-handler/roblox-player").Output()
	if err != nil {
		return ""
	}
	name := strings.TrimSpace(string(defaultClientOutput))
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, `\`) || strings.Contains(name, "..") {
		return ""
	}
	return name
}
