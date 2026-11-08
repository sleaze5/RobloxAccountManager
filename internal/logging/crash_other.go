//go:build !windows

package logging

import "os/exec"

func hideCrashMonitor(*exec.Cmd) {}
