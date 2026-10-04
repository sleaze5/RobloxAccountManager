package logging

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
	"github.com/sleaze5/RobloxAccountManager/internal/appmeta"
)

const (
	crashMonitorArgument = "--internal-crash-monitor"
	cleanShutdownRecord  = "application shutdown completed\n"
)

type crashCapture struct {
	launchID string
	pipe     *os.File
	command  *exec.Cmd
}

func (system *System) CaptureCrashes(launchID string) error {
	system.mu.Lock()
	defer system.mu.Unlock()
	capture := &crashCapture{launchID: launchID}
	if err := capture.setEnabled(system.filter.includes(slog.LevelError)); err != nil {
		return err
	}
	system.crash = capture
	return nil
}

func (capture *crashCapture) setEnabled(enabled bool) error {
	if !enabled {
		return capture.close()
	}
	if capture.pipe != nil {
		return nil
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	command := exec.Command(executable, crashMonitorArgument, capture.launchID)
	hideCrashMonitor(command)
	pipe, err := command.StdinPipe()
	if err != nil {
		return err
	}
	if err := command.Start(); err != nil {
		_ = pipe.Close()
		return err
	}
	capture.pipe, capture.command = pipe.(*os.File), command
	if err := debug.SetCrashOutput(capture.pipe, debug.CrashOptions{}); err != nil {
		_ = capture.close()
		return err
	}
	return nil
}

func (capture *crashCapture) close() error {
	if capture.pipe == nil {
		return nil
	}
	if err := debug.SetCrashOutput(nil, debug.CrashOptions{}); err != nil {
		return err
	}
	_, writeErr := io.WriteString(capture.pipe, cleanShutdownRecord)
	closeErr := capture.pipe.Close()
	waitErr := capture.command.Wait()
	capture.pipe, capture.command = nil, nil
	return errors.Join(writeErr, closeErr, waitErr)
}

func RunCrashMonitor() (bool, error) {
	if len(os.Args) != 3 || os.Args[1] != crashMonitorArgument {
		return false, nil
	}
	data, err := io.ReadAll(io.LimitReader(os.Stdin, 1<<20))
	if err != nil {
		return true, err
	}
	if _, err := io.Copy(io.Discard, os.Stdin); err != nil {
		return true, err
	}
	if string(data) == cleanShutdownRecord {
		return true, nil
	}
	root, err := appdata.ExecutableDirectory()
	if err != nil {
		return true, err
	}
	system, err := Open(Config{
		Directory: filepath.Join(root, "logs"), LaunchID: os.Args[2], Resume: true,
		EnabledLevels: []Level{LevelError},
	})
	if err != nil {
		return true, err
	}
	lines := strings.Split(string(data), "\n")
	reason := "unhandled Go panic or fatal runtime error (payload omitted)"
	message := "application crashed"
	if len(data) == 0 {
		message = "application exited unexpectedly"
		reason = "process terminated without a shutdown record or Go crash report; cause unavailable"
	}
	for _, line := range lines {
		if safe := crashReason(line); safe != "" {
			reason = safe
			break
		}
	}
	system.Module("runtime.crash").Error(message,
		"operation", "crash", "error", reason,
		"version", appmeta.Version, "go_version", runtime.Version(),
		"stack", crashFrames(lines), "truncated", len(data) == 1<<20)
	return true, errors.Join(system.Sync(), system.Close())
}

func crashReason(line string) string {
	for _, reason := range []string{
		"runtime error: invalid memory address or nil pointer dereference",
		"runtime error: index out of range", "runtime error: slice bounds out of range",
		"runtime error: integer divide by zero", "runtime error: makeslice: len out of range",
		"runtime error: makeslice: cap out of range", "assignment to entry in nil map",
		"send on closed channel", "close of closed channel", "close of nil channel",
		"concurrent map writes", "concurrent map read and map write",
		"concurrent map iteration and map write", "all goroutines are asleep - deadlock!",
		"runtime: out of memory", "stack overflow", "unexpected fault address",
	} {
		if strings.HasPrefix(line, "panic: "+reason) || strings.HasPrefix(line, "fatal error: "+reason) {
			return reason
		}
	}
	return ""
}

var (
	crashGoroutine = regexp.MustCompile(`^goroutine [0-9]+ \[[^\r\n]+\]:$`)
	crashLocation  = regexp.MustCompile(`^\t([^\r\n]+\.go:[0-9]+)(?: \+0x[0-9a-f]+.*)?$`)
	crashFunction  = regexp.MustCompile(`^[A-Za-z0-9_./\\@*()\[\]{}·-]+$`)
)

func crashFrames(lines []string) []string {
	var frames []string
	stackStarted := false
	for index, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		if crashGoroutine.MatchString(line) {
			stackStarted = true
		}
		location := crashLocation.FindStringSubmatch(line)
		if !stackStarted || index == 0 || location == nil {
			continue
		}
		function := lines[index-1]
		if args := strings.LastIndexByte(function, '('); args >= 0 {
			function = function[:args]
		}
		if crashFunction.MatchString(function) {
			frames = append(frames, function+" at "+location[1])
		}
	}
	return frames
}

// PanicReason deliberately avoids formatting arbitrary panic values: they can
// contain credentials or account data. Runtime faults have safe built-in errors.
func PanicReason(value any) string {
	if err, ok := value.(runtime.Error); ok {
		return err.Error()
	}
	return fmt.Sprintf("panic of type %T (payload omitted)", value)
}

func PanicStack() []string {
	pcs := make([]uintptr, 64)
	count := runtime.Callers(2, pcs)
	iterator := runtime.CallersFrames(pcs[:count])
	var frames []string
	for {
		frame, more := iterator.Next()
		frames = append(frames, fmt.Sprintf("%s at %s:%d", frame.Function, frame.File, frame.Line))
		if !more {
			return frames
		}
	}
}
