//go:build linux

package gamelaunch

import (
	"context"
	"io"
	"os/exec"
	"strings"
)

func New(preferred func() string) Launcher {
	return protocolLauncher{open: func(ctx context.Context, protocolURL string) error {
		return openLinuxClient(ctx, preferred(), protocolURL)
	}}
}

func openLinuxClient(ctx context.Context, preferred, protocolURL string) error {
	client, desktop, err := resolveLinuxClient(preferred)
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
