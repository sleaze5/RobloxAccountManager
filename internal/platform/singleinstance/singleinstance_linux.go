//go:build linux

package singleinstance

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
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
	command := exec.Command("notify-send", "--app-name", "Roblox Account Manager", "Roblox Account Manager", "Roblox Account Manager is already running.")
	if err := command.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Roblox Account Manager is already running.")
	}
}

func (instance *Instance) Close() error {
	if instance == nil || instance.listener == nil {
		return nil
	}
	err := instance.listener.Close()
	instance.listener = nil
	return err
}
