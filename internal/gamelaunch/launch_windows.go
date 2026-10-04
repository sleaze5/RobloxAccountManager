//go:build windows

package gamelaunch

import (
	"context"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type windowsLauncher struct{}

type preparedLaunch struct {
	protocolURL string
	executable  string
	arguments   []string
}

func New(func() string) Launcher {
	return windowsLauncher{}
}

func prepareLaunch(ctx context.Context, ticket, browserTrackerID string, request Request) (preparedLaunch, error) {
	if err := ctx.Err(); err != nil {
		return preparedLaunch{}, err
	}
	if request.DirectLaunch {
		return preparePlayerDirectly(ticket, browserTrackerID, request)
	}
	protocolURL, err := BuildProtocolURL(ticket, browserTrackerID, request, time.Now())
	return preparedLaunch{protocolURL: protocolURL}, err
}

func (windowsLauncher) Launch(ctx context.Context, ticket, browserTrackerID string, request Request) error {
	prepared, err := prepareLaunch(ctx, ticket, browserTrackerID, request)
	if err != nil {
		return err
	}
	if prepared.executable == "" {
		return launchFromExplorer(ctx, prepared.protocolURL, "", "")
	}
	arguments := make([]string, len(prepared.arguments))
	for index, argument := range prepared.arguments {
		arguments[index] = syscall.EscapeArg(argument)
	}
	return launchFromExplorer(ctx, prepared.executable, strings.Join(arguments, " "), filepath.Dir(prepared.executable))
}

func (windowsLauncher) LaunchProtocol(ctx context.Context, protocolURL string) error {
	if err := ValidateBrowserProtocolURL(protocolURL); err != nil {
		return err
	}
	return launchFromExplorer(ctx, protocolURL, "", "")
}

func (windowsLauncher) CommandLine(ctx context.Context, ticket, browserTrackerID string, request Request) (string, error) {
	prepared, err := prepareLaunch(ctx, ticket, browserTrackerID, request)
	if err != nil {
		return "", err
	}
	if prepared.executable == "" {
		return prepared.protocolURL, nil
	}
	parts := []string{quoteWindowsArgument(prepared.executable)}
	for _, argument := range prepared.arguments {
		escaped := syscall.EscapeArg(argument)
		if strings.ContainsAny(argument, "&|<>^()%!") {
			escaped = quoteWindowsArgument(argument)
		}
		parts = append(parts, escaped)
	}
	return strings.Join(parts, " "), nil
}

func quoteWindowsArgument(value string) string {
	escaped := syscall.EscapeArg(value)
	if strings.HasPrefix(escaped, `"`) {
		return escaped
	}
	trailingSlashes := len(escaped) - len(strings.TrimRight(escaped, `\`))
	return `"` + escaped + strings.Repeat(`\`, trailingSlashes) + `"`
}
