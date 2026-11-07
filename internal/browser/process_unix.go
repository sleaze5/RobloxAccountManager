//go:build linux || darwin

package browser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type unixBrowserProcess struct {
	command       *exec.Cmd
	input         *os.File
	output        *os.File
	exited        chan error
	windowsClosed chan struct{}
	done          chan struct{}
	closeOnce     sync.Once

	mutex    sync.Mutex
	finished bool
}

func startBrowserProcess(options ProcessOptions) (BrowserProcess, error) {
	if options.Executable == "" || options.UserDataPath == "" {
		return nil, errors.New("invalid browser process options")
	}
	commandRead, parentWrite, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("create browser command pipe: %w", err)
	}
	parentRead, commandWrite, err := os.Pipe()
	if err != nil {
		_ = commandRead.Close()
		_ = parentWrite.Close()
		return nil, fmt.Errorf("create browser response pipe: %w", err)
	}
	closeAll := func() {
		_ = commandRead.Close()
		_ = parentWrite.Close()
		_ = parentRead.Close()
		_ = commandWrite.Close()
	}

	command := exec.CommandContext(context.Background(), options.Executable,
		"--user-data-dir="+options.UserDataPath, "--remote-debugging-pipe",
		"--no-first-run", "--no-default-browser-check", "--disable-background-mode",
		"--disable-infobars", "--window-size=1080,600", credentialStoreFlag, "--incognito", "about:blank")
	command.Dir = options.UserDataPath
	// Chrome reads CDP commands from descriptor 3 and writes responses to 4.
	command.ExtraFiles = []*os.File{commandRead, commandWrite}
	command.SysProcAttr = browserProcessAttributes()
	if err := command.Start(); err != nil {
		closeAll()
		return nil, fmt.Errorf("start browser: %w", err)
	}
	_ = commandRead.Close()
	_ = commandWrite.Close()

	process := &unixBrowserProcess{
		command: command, input: parentWrite, output: parentRead,
		exited: make(chan error, 1), windowsClosed: make(chan struct{}, 1), done: make(chan struct{}),
	}
	go process.waitLoop()
	return process, nil
}

func (process *unixBrowserProcess) CDPPipes() (io.WriteCloser, io.ReadCloser) {
	return process.input, process.output
}
func (process *unixBrowserProcess) AllWindowsClosed() <-chan struct{} {
	return process.windowsClosed
}
func (process *unixBrowserProcess) Exited() <-chan error { return process.exited }

func (process *unixBrowserProcess) waitLoop() {
	err := process.command.Wait()
	close(process.done)
	process.mutex.Lock()
	process.finished = true
	// Chrome exits with code 0 when the user closes its last window on Linux, or
	// quits it on macOS. This is how the coordinator tells a user close from a crash.
	if process.command.ProcessState.ExitCode() == 0 {
		select {
		case process.windowsClosed <- struct{}{}:
		default:
		}
	}
	close(process.windowsClosed)
	process.mutex.Unlock()
	process.exited <- err
	close(process.exited)
}

func (process *unixBrowserProcess) ExitError() error {
	if !process.Wait(200 * time.Millisecond) {
		return nil
	}
	return fmt.Errorf("browser process exited with %s", process.command.ProcessState)
}

func (process *unixBrowserProcess) Wait(timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-process.done:
		return true
	case <-timer.C:
		return false
	}
}

func (process *unixBrowserProcess) Terminate() error {
	if err := syscall.Kill(-process.command.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("stop browser process: %w", err)
	}
	return nil
}

func (process *unixBrowserProcess) Focus() error { return nil }

func (process *unixBrowserProcess) Close() error {
	process.closeOnce.Do(func() {
		_ = process.input.Close()
		_ = process.output.Close()
	})
	return nil
}
