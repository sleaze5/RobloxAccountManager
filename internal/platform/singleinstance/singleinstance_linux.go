//go:build linux

package singleinstance

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/sleaze5/RobloxAccountManager/internal/appmeta"
)

type Instance struct {
	listener *net.UnixListener
}

func Acquire(applicationID string) (*Instance, bool, error) {
	if applicationID == "" || strings.ContainsRune(applicationID, 0) {
		return nil, false, errors.New("invalid single-instance name")
	}
	address, err := net.ResolveUnixAddr("unix", "@"+applicationID)
	if err != nil {
		return nil, false, fmt.Errorf("resolve single-instance socket: %w", err)
	}
	listener, err := net.ListenUnix("unix", address)
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("create single-instance socket: %w", err)
	}
	return &Instance{listener: listener}, true, nil
}

func ShowAlreadyRunning() {
	command := exec.CommandContext(context.Background(), "notify-send", "--app-name", appmeta.DisplayName, appmeta.DisplayName, alreadyRunningMessage)
	if err := command.Run(); err != nil {
		fmt.Fprintln(os.Stderr, alreadyRunningMessage)
	}
}

func (instance *Instance) Close() error {
	if instance == nil || instance.listener == nil {
		return nil
	}
	err := instance.listener.Close()
	instance.listener = nil
	if err != nil {
		return fmt.Errorf("close single-instance socket: %w", err)
	}
	return nil
}
