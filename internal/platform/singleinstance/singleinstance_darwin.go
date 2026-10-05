//go:build darwin

package singleinstance

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// Instance holds an exclusive lock on a file in the per-user temporary
// folder. The system releases the lock when the process exits.
type Instance struct {
	file *os.File
}

func Acquire(applicationID string) (*Instance, bool, error) {
	if applicationID == "" || strings.ContainsAny(applicationID, "/\x00") {
		return nil, false, errors.New("invalid single-instance name")
	}
	file, err := os.OpenFile(filepath.Join(os.TempDir(), applicationID+".lock"), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, false, fmt.Errorf("open single-instance lock: %w", err)
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, unix.EWOULDBLOCK) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("lock single-instance file: %w", err)
	}
	return &Instance{file: file}, true, nil
}

func ShowAlreadyRunning() {
	command := exec.Command("/usr/bin/osascript", "-e",
		`display alert "Roblox Account Manager is already running." as informational`)
	if err := command.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Roblox Account Manager is already running.")
	}
}

func (instance *Instance) Close() error {
	if instance == nil || instance.file == nil {
		return nil
	}
	err := instance.file.Close()
	instance.file = nil
	return err
}
