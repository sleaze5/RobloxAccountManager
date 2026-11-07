package logging

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
)

const (
	maxLogBytes      = 5 << 20
	retainedLogFiles = 5
)

type config struct {
	EnabledLevels []Level
	Directory     string
	LaunchID      string
	Resume        bool
	AddSource     bool
	// Hold keeps entries in memory and leaves Directory untouched until Release.
	Hold       bool
	components *componentRegistry
}

type System struct {
	base   *slog.Logger
	writer *rotatingWriter
	filter *levelFilter
	mu     sync.Mutex
	crash  *crashCapture
}

func open(config config) (*System, error) {
	levels := config.EnabledLevels
	header, err := launchHeader(config.LaunchID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(config.Directory) == "" {
		return nil, errors.New("log directory is required")
	}
	if !config.Hold {
		if err := appdata.PreparePrivateDirectory(config.Directory); err != nil {
			return nil, err
		}
	}
	writer := newRotatingWriter(filepath.Join(config.Directory, config.LaunchID+".log"), header, config.Resume)
	writer.held = config.Hold
	if len(levels) > 0 {
		if err := writer.ensureOpen(); err != nil {
			return nil, err
		}
	}
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		AddSource:   config.AddSource,
		Level:       TraceLevel,
		ReplaceAttr: replaceLogAttribute,
	})
	filter := newLevelFilter(levels)
	base := slog.New(&redactingHandler{next: &componentHandler{next: &filteringHandler{next: handler, filter: filter}, registry: config.components}}).With("launch_id", config.LaunchID)
	return &System{base: base, writer: writer, filter: filter}, nil
}

func NewConsole() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError, ReplaceAttr: replaceLogAttribute})
	return slog.New(&redactingHandler{next: handler})
}

func (system *System) Module(name string) *slog.Logger {
	return system.base.With("module", name)
}

func (system *System) SetEnabledLevels(levels []Level) error {
	system.mu.Lock()
	defer system.mu.Unlock()
	if len(levels) > 0 {
		if err := system.writer.ensureOpen(); err != nil {
			return err
		}
	}
	if system.crash != nil {
		if err := system.crash.setEnabled(newLevelFilter(levels).includes(slog.LevelError)); err != nil {
			return err
		}
	}
	system.filter.set(levels)
	return nil
}

func (system *System) Release(directory string) error {
	system.mu.Lock()
	defer system.mu.Unlock()
	if err := appdata.PreparePrivateDirectory(directory); err != nil {
		return err
	}
	return system.writer.release(directory, system.filter.any())
}

func (system *System) Close() error {
	if system == nil || system.writer == nil {
		return nil
	}
	system.mu.Lock()
	defer system.mu.Unlock()
	if system.crash != nil {
		if err := system.crash.close(); err != nil {
			system.Module("logging").Error("close crash capture", "error", err)
		}
		system.crash = nil
	}
	return system.writer.Close()
}

func (system *System) Sync() error { return system.writer.Sync() }
