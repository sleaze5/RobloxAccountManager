package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
)

const (
	defaultMaxBytes = 5 << 20
	defaultRetained = 5
)

type Config struct {
	EnabledLevels []Level
	Directory     string
	LaunchID      string
	Resume        bool
	MaxBytes      int64
	RetainedFiles int
	AddSource     bool
	components    *componentRegistry
}

type System struct {
	base   *slog.Logger
	writer *rotatingWriter
	filter *levelFilter
	mu     sync.Mutex
	crash  *crashCapture
}

func Open(config Config) (*System, error) {
	levels := config.EnabledLevels
	if levels == nil {
		levels = AllLevels()
	}
	for _, level := range levels {
		if level == LevelOff {
			return nil, fmt.Errorf("off is not an individual log level")
		}
		if _, err := ParseLevel(string(level)); err != nil {
			return nil, err
		}
	}
	header, err := launchHeader(config.LaunchID)
	if err != nil {
		return nil, err
	}
	if config.MaxBytes == 0 {
		config.MaxBytes = defaultMaxBytes
	}
	if config.RetainedFiles == 0 {
		config.RetainedFiles = defaultRetained
	}
	if strings.TrimSpace(config.Directory) == "" {
		return nil, fmt.Errorf("log directory is required")
	}
	if err := appdata.PreparePrivateDirectory(config.Directory); err != nil {
		return nil, err
	}
	writer, err := newRotatingWriter(filepath.Join(config.Directory, config.LaunchID+".log"), config.MaxBytes, config.RetainedFiles, header, config.Resume)
	if err != nil {
		return nil, err
	}
	if len(levels) > 0 {
		if err := writer.ensureOpen(); err != nil {
			return nil, err
		}
	}
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		AddSource:   config.AddSource,
		Level:       traceSlogLevel,
		ReplaceAttr: replaceLogAttribute,
	})
	filter := newLevelFilter(levels)
	base := slog.New(&redactingHandler{next: &componentHandler{next: &filteringHandler{next: handler, filter: filter}, registry: config.components}}).With("launch_id", config.LaunchID)
	return &System{base: base, writer: writer, filter: filter}, nil
}

func NewConsole(writer io.Writer, level Level) *slog.Logger {
	if writer == nil {
		writer = os.Stderr
	}
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: level.slogLevel(), ReplaceAttr: replaceLogAttribute})
	return slog.New(&redactingHandler{next: handler})
}

func (system *System) Module(name string) *slog.Logger {
	return system.base.With("module", name)
}

func (system *System) SetEnabledLevels(levels []Level) error {
	system.mu.Lock()
	defer system.mu.Unlock()
	for _, level := range levels {
		if level == LevelOff {
			return fmt.Errorf("off is not an individual log level")
		}
		if _, err := ParseLevel(string(level)); err != nil {
			return err
		}
	}
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
