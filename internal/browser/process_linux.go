//go:build linux

package browser

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type linuxBrowserProcess struct {
	command       *exec.Cmd
	input         *os.File
	output        *os.File
	exited        chan error
	windowsClosed chan struct{}
	done          chan struct{}
	closeOnce     sync.Once
}

func startBrowserProcess(options ProcessOptions) (BrowserProcess, error) {
	if options.Executable == "" || options.UserDataPath == "" {
		return nil, errors.New("invalid browser process options")
	}
	commandRead, parentWrite, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	parentRead, commandWrite, err := os.Pipe()
	if err != nil {
		commandRead.Close()
		parentWrite.Close()
		return nil, err
	}
	closeAll := func() {
		commandRead.Close()
		parentWrite.Close()
		parentRead.Close()
		commandWrite.Close()
	}

	command := exec.Command(options.Executable,
		"--user-data-dir="+options.UserDataPath, "--remote-debugging-pipe",
		"--no-first-run", "--no-default-browser-check", "--disable-background-mode",
		"--disable-infobars", "--password-store=basic", "--incognito", "about:blank")
	command.Dir = options.UserDataPath
	// Chrome reads CDP commands from descriptor 3 and writes responses to 4.
	command.ExtraFiles = []*os.File{commandRead, commandWrite}
	// A separate process group lets Terminate stop Chrome and its helpers
	// together. Pdeathsig covers an application crash.
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
	if err := command.Start(); err != nil {
		closeAll()
		return nil, fmt.Errorf("start browser: %w", err)
	}
	commandRead.Close()
	commandWrite.Close()

	process := &linuxBrowserProcess{
		command: command, input: parentWrite, output: parentRead,
		exited: make(chan error, 1), windowsClosed: make(chan struct{}, 1), done: make(chan struct{}),
	}
	go process.waitLoop()
	return process, nil
}

func (process *linuxBrowserProcess) CDPPipes() (io.WriteCloser, io.ReadCloser) {
	return process.input, process.output
}
func (process *linuxBrowserProcess) AllWindowsClosed() <-chan struct{} {
	return process.windowsClosed
}
func (process *linuxBrowserProcess) Exited() <-chan error { return process.exited }

func (process *linuxBrowserProcess) waitLoop() {
	err := process.command.Wait()
	close(process.done)
	// Chrome exits with code 0 when its last window closes, which is how the
	// coordinator tells a user close from a crash.
	if process.command.ProcessState.ExitCode() == 0 {
		process.windowsClosed <- struct{}{}
	}
	close(process.windowsClosed)
	process.exited <- err
	close(process.exited)
}

func (process *linuxBrowserProcess) ExitError() error {
	if !process.Wait(200 * time.Millisecond) {
		return nil
	}
	return fmt.Errorf("browser process exited with %s", process.command.ProcessState)
}

func (process *linuxBrowserProcess) Wait(timeout time.Duration) bool {
	if timeout <= 0 {
		select {
		case <-process.done:
			return true
		default:
			return false
		}
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-process.done:
		return true
	case <-timer.C:
		return false
	}
}

func (process *linuxBrowserProcess) Terminate() error {
	err := syscall.Kill(-process.command.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}

func (process *linuxBrowserProcess) Focus() error { return nil }

func (process *linuxBrowserProcess) Close() error {
	process.closeOnce.Do(func() {
		_ = process.input.Close()
		_ = process.output.Close()
	})
	return nil
}
