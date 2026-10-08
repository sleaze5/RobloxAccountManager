package logging

import (
	"os/exec"
	"syscall"
)

func hideCrashMonitor(command *exec.Cmd) {
	// 0x08000000 is CREATE_NO_WINDOW.
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}
