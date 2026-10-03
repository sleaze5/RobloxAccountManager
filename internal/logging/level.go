package logging

import (
	"fmt"
	"log/slog"
	"strings"
)

type Level string

const (
	LevelOff   Level = "off"
	LevelError Level = "error"
	LevelWarn  Level = "warn"
	LevelInfo  Level = "info"
	LevelDebug Level = "debug"
	LevelTrace Level = "trace"
)

const traceSlogLevel = slog.LevelDebug - 4

var allLevels = []Level{LevelTrace, LevelDebug, LevelInfo, LevelWarn, LevelError}

func ParseLevel(value string) (Level, error) {
	level := Level(strings.ToLower(strings.TrimSpace(value)))
	switch level {
	case LevelOff, LevelError, LevelWarn, LevelInfo, LevelDebug, LevelTrace:
		return level, nil
	default:
		return "", fmt.Errorf("unknown log level %q", value)
	}
}

func ParseLevels(values []string) ([]Level, error) {
	levels := make([]Level, 0, len(values))
	seen := make(map[Level]bool, len(values))
	for _, value := range values {
		level, err := ParseLevel(value)
		if err != nil {
			return nil, err
		}
		if level == LevelOff {
			return nil, fmt.Errorf("off is not an individual log level")
		}
		if seen[level] {
			return nil, fmt.Errorf("duplicate log level %q", level)
		}
		seen[level] = true
		levels = append(levels, level)
	}
	return levels, nil
}

func AllLevels() []Level {
	return append([]Level(nil), allLevels...)
}

func (level Level) slogLevel() slog.Level {
	switch level {
	case LevelError:
		return slog.LevelError
	case LevelWarn:
		return slog.LevelWarn
	case LevelInfo:
		return slog.LevelInfo
	case LevelDebug:
		return slog.LevelDebug
	case LevelTrace:
		return traceSlogLevel
	default:
		return slog.LevelError + 100
	}
}
