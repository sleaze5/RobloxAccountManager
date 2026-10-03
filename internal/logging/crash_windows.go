package logging

import (
	"os/exec"
	"syscall"
)

func hideCrashMonitor(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}
