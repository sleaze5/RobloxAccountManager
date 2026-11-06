//go:build unix

package gamelaunch

import (
	"context"
	"strings"
	"time"
)

type protocolLauncher struct {
	open func(context.Context, string) error
}

func (launcher protocolLauncher) Launch(ctx context.Context, ticket, browserTrackerID string, request Request) error {
	protocolURL, err := launcher.CommandLine(ctx, ticket, browserTrackerID, request)
	if err != nil {
		return err
	}
	return launcher.openChecked(ctx, protocolURL)
}

func (launcher protocolLauncher) LaunchProtocol(ctx context.Context, protocolURL string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ValidateBrowserProtocolURL(protocolURL); err != nil {
		return err
	}
	return launcher.openChecked(ctx, protocolURL)
}

func (protocolLauncher) CommandLine(ctx context.Context, ticket, browserTrackerID string, request Request) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if request.DirectLaunch {
		return "", &ClientError{Message: "launching the Roblox Player binary directly is only available on Windows"}
	}
	return buildProtocolURL(ticket, browserTrackerID, request, time.Now())
}

func (launcher protocolLauncher) openChecked(ctx context.Context, protocolURL string) error {
	if strings.IndexFunc(protocolURL, func(character rune) bool {
		return character <= ' ' || character == '"' || character == '\\'
	}) >= 0 {
		return &ClientError{Message: "the Roblox launch URL is invalid"}
	}
	return launcher.open(ctx, protocolURL)
}
