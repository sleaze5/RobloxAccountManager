//go:build darwin

package logging

import "os/exec"

func hideCrashMonitor(*exec.Cmd) {}
