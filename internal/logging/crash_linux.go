//go:build linux

package logging

import "os/exec"

func hideCrashMonitor(*exec.Cmd) {}
