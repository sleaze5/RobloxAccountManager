//go:build darwin

package gamelaunch

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const robloxBundleID = "com.roblox.RobloxPlayer"

type darwinLauncher struct{}

func New(func() string) Launcher {
	return darwinLauncher{}
}

func (launcher darwinLauncher) Launch(ctx context.Context, ticket, browserTrackerID string, request Request) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if request.DirectLaunch {
		return &ClientError{Message: "launching the Roblox Player binary directly is only available on Windows"}
	}
	protocolURL, err := BuildProtocolURL(ticket, browserTrackerID, request, time.Now())
	if err != nil {
		return err
	}
	return launcher.open(ctx, protocolURL)
}

func (launcher darwinLauncher) LaunchProtocol(ctx context.Context, protocolURL string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ValidateBrowserProtocolURL(protocolURL); err != nil {
		return err
	}
	return launcher.open(ctx, protocolURL)
}

func (darwinLauncher) CommandLine(ctx context.Context, ticket, browserTrackerID string, request Request) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if request.DirectLaunch {
		return "", &ClientError{Message: "launching the Roblox Player binary directly is only available on Windows"}
	}
	return BuildProtocolURL(ticket, browserTrackerID, request, time.Now())
}

func (darwinLauncher) open(ctx context.Context, protocolURL string) error {
	if !strings.HasPrefix(strings.ToLower(protocolURL), "roblox-player:") || strings.IndexFunc(protocolURL, func(character rune) bool {
		return character <= ' ' || character == '"' || character == '\\'
	}) >= 0 {
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
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return &ClientError{Message: "install Roblox to join a game"}
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
	err := os.Remove(filepath.Join(home, "Library", "HTTPStorages", robloxBundleID+".binarycookies"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func findRoblox(home string) string {
	candidates := []string{filepath.Join("/Applications", "Roblox.app")}
	if home != "" && filepath.IsAbs(home) {
		candidates = append(candidates, filepath.Join(home, "Applications", "Roblox.app"))
	}
	for _, candidate := range candidates {
		info, err := os.ReadFile(filepath.Join(candidate, "Contents", "Info.plist"))
		if err == nil && bytes.Contains(info, []byte(robloxBundleID)) {
			return candidate
		}
	}
	return ""
}
