//go:build darwin

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

type darwinBrowserProcess struct {
	command       *exec.Cmd
	input         *os.File
	output        *os.File
	exited        chan error
	windowsClosed chan struct{}
	done          chan struct{}
	closeOnce     sync.Once

	mu       sync.Mutex
	finished bool
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

	// --use-mock-keychain keeps Chrome from asking for Keychain access.
	command := exec.Command(options.Executable,
		"--user-data-dir="+options.UserDataPath, "--remote-debugging-pipe",
		"--no-first-run", "--no-default-browser-check", "--disable-background-mode",
		"--disable-infobars", "--use-mock-keychain", "--incognito", "about:blank")
	command.Dir = options.UserDataPath
	// Chrome reads CDP commands from descriptor 3 and writes responses to 4.
	command.ExtraFiles = []*os.File{commandRead, commandWrite}
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		closeAll()
		return nil, fmt.Errorf("start browser: %w", err)
	}
	commandRead.Close()
	commandWrite.Close()

	process := &darwinBrowserProcess{
		command: command, input: parentWrite, output: parentRead,
		exited: make(chan error, 1), windowsClosed: make(chan struct{}, 1), done: make(chan struct{}),
	}
	go process.waitLoop()
	return process, nil
}

func (process *darwinBrowserProcess) CDPPipes() (io.WriteCloser, io.ReadCloser) {
	return process.input, process.output
}
func (process *darwinBrowserProcess) AllWindowsClosed() <-chan struct{} {
	return process.windowsClosed
}
func (process *darwinBrowserProcess) Exited() <-chan error { return process.exited }

// Chrome on macOS keeps running after its last window closes.
func (process *darwinBrowserProcess) PagesClosed() {
	process.mu.Lock()
	defer process.mu.Unlock()
	if process.finished {
		return
	}
	select {
	case process.windowsClosed <- struct{}{}:
	default:
	}
}

func (process *darwinBrowserProcess) waitLoop() {
	err := process.command.Wait()
	close(process.done)
	process.mu.Lock()
	process.finished = true
	// Chrome exits with code 0 when the user quits it.
	if process.command.ProcessState.ExitCode() == 0 {
		select {
		case process.windowsClosed <- struct{}{}:
		default:
		}
	}
	close(process.windowsClosed)
	process.mu.Unlock()
	process.exited <- err
	close(process.exited)
}

func (process *darwinBrowserProcess) ExitError() error {
	if !process.Wait(200 * time.Millisecond) {
		return nil
	}
	return fmt.Errorf("browser process exited with %s", process.command.ProcessState)
}

func (process *darwinBrowserProcess) Wait(timeout time.Duration) bool {
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

func (process *darwinBrowserProcess) Terminate() error {
	err := syscall.Kill(-process.command.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}

func (process *darwinBrowserProcess) Focus() error { return nil }

func (process *darwinBrowserProcess) Close() error {
	process.closeOnce.Do(func() {
		_ = process.input.Close()
		_ = process.output.Close()
	})
	return nil
}
