//go:build darwin

package gamelaunch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const robloxBundleID = "com.roblox.RobloxPlayer"

func New(func() string) Launcher {
	return protocolLauncher{open: openRoblox}
}

func openRoblox(ctx context.Context, protocolURL string) error {
	if !strings.HasPrefix(strings.ToLower(protocolURL), "roblox-player:") {
		return &ClientError{Message: "the Roblox launch URL is invalid"}
	}
	home, _ := os.UserHomeDir()
	if err := removeStoredLogin(home); err != nil {
		return &ClientError{Message: "the stored Roblox login could not be cleared"}
	}
	arguments := []string{protocolURL}
	// A bootstrapper that registered roblox-player: must not receive the ticket.
	if app := findRoblox(home); app != "" {
		arguments = []string{"-a", app, protocolURL}
	}
	command := exec.CommandContext(ctx, "/usr/bin/open", arguments...)
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if _, ok := errors.AsType[*exec.ExitError](err); ok {
			return &ClientError{Message: protocolUnavailable}
		}
		return &ClientError{Message: "roblox could not be started"}
	}
	return nil
}

// Roblox prefers its saved session over the launch ticket, which would join
// with the account that last signed in to the Roblox app.
func removeStoredLogin(home string) error {
	if home == "" || !filepath.IsAbs(home) {
		return nil
	}
	if err := os.Remove(filepath.Join(home, "Library", "HTTPStorages", robloxBundleID+".binarycookies")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove stored Roblox login: %w", err)
	}
	return nil
}

func findRoblox(home string) string {
	candidates := []string{"/Applications/Roblox.app"}
	if home != "" && filepath.IsAbs(home) {
		candidates = append(candidates, filepath.Join(home, "Applications", "Roblox.app"))
	}
	for _, candidate := range candidates {
		infoPlist, err := os.ReadFile(filepath.Join(candidate, "Contents", "Info.plist"))
		if err == nil && bytes.Contains(infoPlist, []byte(robloxBundleID)) {
			return candidate
		}
	}
	return ""
}
