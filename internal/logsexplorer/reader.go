package logsexplorer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
)

type Reader struct {
	directories func() ([]string, error)
	logger      *slog.Logger
	operation   atomic.Uint64
}

func NewReader(directories func() ([]string, error), logger *slog.Logger) *Reader {
	return &Reader{directories: directories, logger: logger}
}

func (reader *Reader) ReadAll(ctx context.Context) (Snapshot, error) {
	logger := reader.operationLogger("refresh-all")
	snapshot := Snapshot{Sessions: []Session{}}
	directories, err := reader.directories()
	if err != nil {
		logger.Warn("could not locate Roblox logs folders", "error", err)
		return snapshot, errors.New("could not open Roblox logs folder")
	}
	for _, directory := range directories {
		sessions, err := readDirectory(ctx, logger, directory)
		if ctx.Err() != nil {
			return Snapshot{}, ctx.Err()
		}
		if err != nil {
			return snapshot, err
		}
		snapshot.Sessions = append(snapshot.Sessions, sessions...)
	}
	slices.SortFunc(snapshot.Sessions, func(a, b Session) int {
		if a.StartedAtMs > b.StartedAtMs {
			return -1
		}
		if a.StartedAtMs < b.StartedAtMs {
			return 1
		}
		return strings.Compare(a.FileName, b.FileName)
	})
	logger.Info("Roblox logs refreshed", "sessions", len(snapshot.Sessions))
	return snapshot, nil
}

func (reader *Reader) ReadFile(ctx context.Context, name string) (Session, error) {
	if !validFileName(name) {
		return Session{}, errors.New("invalid Roblox Player log filename")
	}
	logger := reader.operationLogger("refresh-log").With("file", name)
	directories, err := reader.directories()
	if err != nil {
		logger.Warn("could not locate Roblox logs folders", "error", err)
		return Session{}, errors.New("could not open Roblox logs folder")
	}
	session, err := readFromDirectories(ctx, directories, name)
	if err != nil {
		logger.Warn("could not refresh Roblox log", "error", err)
		return Session{}, errors.New("could not read Roblox log")
	}
	logger.Info("Roblox log refreshed", "visits", len(session.Visits))
	return session, nil
}

func readDirectory(ctx context.Context, logger *slog.Logger, directory string) ([]Session, error) {
	root, err := os.OpenRoot(directory)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		logger.Warn("could not open Roblox logs folder", "error", err)
		return nil, errors.New("could not open Roblox logs folder")
	}
	defer root.Close()
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		logger.Warn("could not list Roblox logs", "error", err)
		return nil, errors.New("could not list Roblox logs")
	}
	var sessions []Session
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !entry.Type().IsRegular() || !validFileName(entry.Name()) {
			continue
		}
		session, err := readFile(ctx, root, entry.Name())
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err != nil {
			logger.Warn("could not fully parse Roblox log", "file", entry.Name(), "error", err)
			session.Issue = "This log could not be fully read. Refresh this session to try again."
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func readFromDirectories(ctx context.Context, directories []string, name string) (Session, error) {
	for _, directory := range directories {
		root, err := os.OpenRoot(directory)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return Session{}, err
		}
		session, err := readFile(ctx, root, name)
		root.Close()
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		return session, err
	}
	return Session{}, os.ErrNotExist
}

func (reader *Reader) operationLogger(operation string) *slog.Logger {
	return reader.logger.With("operation", operation, "operation_id", fmt.Sprintf("logs-explorer-%d", reader.operation.Add(1)))
}

func validFileName(name string) bool {
	lower := strings.ToLower(name)
	return filepath.IsLocal(name) && !strings.ContainsAny(name, `/\:`) &&
		strings.Contains(lower, "_player_") && strings.HasSuffix(lower, ".log")
}

func readFile(ctx context.Context, root *os.Root, name string) (Session, error) {
	session := newSession(name)
	file, err := root.Open(name)
	if err != nil {
		return session, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return session, err
	}
	if !info.Mode().IsRegular() {
		return session, errors.New("not a regular log file")
	}
	session.SizeBytes = info.Size()
	err = parse(ctx, io.LimitReader(file, info.Size()), &session)
	return session, err
}
