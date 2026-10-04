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

func listClients() []ClientInfo {
	clients := make([]ClientInfo, 0, len(knownLinuxClients))
	for _, client := range knownLinuxClients {
		_, installed := findDesktop(client.desktop)
		clients = append(clients, ClientInfo{ID: client.id, Name: client.name, Installed: installed})
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
		return knownClient{}, "", &ClientError{Message: "install Sober or Mocktail to join a game"}
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
	for _, dir := range applicationDirectories() {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err == nil && info.Mode().IsRegular() {
			return path, true
		}
	}
	return "", false
}

func applicationDirectories() []string {
	seen := map[string]struct{}{}
	dirs := make([]string, 0, 6)
	add := func(dir string) {
		if dir == "" {
			return
		}
		dir = filepath.Clean(dir)
		if _, ok := seen[dir]; ok {
			return
		}
		seen[dir] = struct{}{}
		dirs = append(dirs, dir)
	}
	home, _ := os.UserHomeDir()
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		add(filepath.Join(dataHome, "applications"))
	} else if home != "" {
		add(filepath.Join(home, ".local", "share", "applications"))
	}
	dataDirs := os.Getenv("XDG_DATA_DIRS")
	if dataDirs == "" {
		dataDirs = "/usr/local/share" + string(filepath.ListSeparator) + "/usr/share"
	}
	for _, dir := range filepath.SplitList(dataDirs) {
		add(filepath.Join(dir, "applications"))
	}
	if home != "" {
		add(filepath.Join(home, ".local", "share", "flatpak", "exports", "share", "applications"))
	}
	add(filepath.Join("/var", "lib", "flatpak", "exports", "share", "applications"))
	return dirs
}

func defaultRobloxDesktop() string {
	xdg, err := exec.LookPath("xdg-mime")
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, xdg, "query", "default", "x-scheme-handler/roblox-player").Output()
	if err != nil {
		return ""
	}
	name := strings.TrimSpace(string(out))
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, `\`) || strings.Contains(name, "..") {
		return ""
	}
	return name
}
