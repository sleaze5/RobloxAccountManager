//go:build linux

package gamelaunch

import (
	"context"
	"io"
	"os/exec"
	"strings"
	"time"
)

type linuxLauncher struct {
	preferred func() string
}

func New(preferred func() string) Launcher {
	if preferred == nil {
		preferred = func() string { return "" }
	}
	return linuxLauncher{preferred: preferred}
}

func (launcher linuxLauncher) Launch(ctx context.Context, ticket, browserTrackerID string, request Request) error {
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

func (launcher linuxLauncher) LaunchProtocol(ctx context.Context, protocolURL string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ValidateBrowserProtocolURL(protocolURL); err != nil {
		return err
	}
	return launcher.open(ctx, protocolURL)
}

func (linuxLauncher) CommandLine(ctx context.Context, ticket, browserTrackerID string, request Request) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if request.DirectLaunch {
		return "", &ClientError{Message: "launching the Roblox Player binary directly is only available on Windows"}
	}
	return BuildProtocolURL(ticket, browserTrackerID, request, time.Now())
}

func (launcher linuxLauncher) open(ctx context.Context, protocolURL string) error {
	if strings.IndexFunc(protocolURL, func(character rune) bool {
		return character <= ' ' || character == '"' || character == '\\'
	}) >= 0 {
		return &ClientError{Message: "the Roblox launch URL is invalid"}
	}
	client, desktop, err := resolveLinuxClient(launcher.preferred())
	if err != nil {
		return err
	}
	gio, err := exec.LookPath("gio")
	if err != nil {
		return &ClientError{Message: "gio is required to open the Roblox client"}
	}
	command := exec.CommandContext(ctx, gio, "launch", desktop, protocolURL)
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return &ClientError{Message: strings.ToLower(client.name) + " could not be started"}
	}
	return nil
}
