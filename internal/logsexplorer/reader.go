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
	"sync"
	"sync/atomic"
	"time"
)

// parsedLog is a parsed log file with the size and modification time that it
// had when it was parsed. Roblox only appends to a log, so a file with the same
// size and modification time parses to the same session.
type parsedLog struct {
	sizeBytes  int64
	modifiedAt time.Time
	session    Session
}

type Reader struct {
	directories     func() ([]string, error)
	logger          *slog.Logger
	operation       atomic.Uint64
	parsedLogMutex  sync.Mutex
	parsedLogByPath map[string]parsedLog
}

func NewReader(directories func() ([]string, error), logger *slog.Logger) *Reader {
	return &Reader{directories: directories, logger: logger, parsedLogByPath: map[string]parsedLog{}}
}

func (reader *Reader) ReadAll(ctx context.Context) (Snapshot, error) {
	logger := reader.operationLogger("refresh-all")
	sessions, err := reader.readSessions(ctx, logger)
	if ctx.Err() != nil {
		return Snapshot{}, ctx.Err()
	}
	snapshot := Snapshot{Sessions: sessions}
	if err != nil {
		return snapshot, err
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

func (reader *Reader) FindServerVisit(ctx context.Context, placeID, jobID string) (ServerVisit, error) {
	sessions, err := reader.readSessions(ctx, reader.operationLogger("server-visit"))
	if err != nil {
		return ServerVisit{}, err
	}
	var visit ServerVisit
	for _, session := range sessions {
		for _, recordedVisit := range session.Visits {
			if recordedVisit.PlaceID == placeID && strings.EqualFold(recordedVisit.JobID, jobID) {
				visit.HasVisited = true
				visit.LastVisitedAtMs = max(visit.LastVisitedAtMs, recordedVisit.StartedAtMs)
			}
		}
	}
	return visit, nil
}

func (reader *Reader) readSessions(ctx context.Context, logger *slog.Logger) ([]Session, error) {
	allSessions := []Session{}
	directories, err := reader.directories()
	if err != nil {
		logger.Warn("could not locate Roblox logs folders", "error", err)
		return allSessions, errors.New("could not open Roblox logs folder")
	}
	listedPaths := map[string]bool{}
	for _, directory := range directories {
		sessions, err := reader.readDirectory(ctx, logger, directory, listedPaths)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err != nil {
			return allSessions, err
		}
		allSessions = append(allSessions, sessions...)
	}
	reader.forgetUnlistedLogs(listedPaths)
	return allSessions, nil
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
	session, err := reader.readFromDirectories(ctx, directories, name)
	if err != nil {
		logger.Warn("could not refresh Roblox log", "error", err)
		return Session{}, errors.New("could not read Roblox log")
	}
	logger.Info("Roblox log refreshed", "visits", len(session.Visits))
	return session, nil
}

func (reader *Reader) readDirectory(ctx context.Context, logger *slog.Logger, directory string, listedPaths map[string]bool) ([]Session, error) {
	root, err := os.OpenRoot(directory)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		logger.Warn("could not open Roblox logs folder", "error", err)
		return nil, errors.New("could not open Roblox logs folder")
	}
	defer func() { _ = root.Close() }()
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
		listedPaths[filepath.Join(directory, entry.Name())] = true
		session, err := reader.readFile(ctx, root, directory, entry.Name())
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

func (reader *Reader) readFromDirectories(ctx context.Context, directories []string, name string) (Session, error) {
	for _, directory := range directories {
		root, err := os.OpenRoot(directory)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return Session{}, fmt.Errorf("open Roblox logs folder: %w", err)
		}
		session, err := reader.readFile(ctx, root, directory, name)
		_ = root.Close()
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
		strings.Contains(lower, "_player_") && !strings.Contains(lower, "_crashhandler") &&
		strings.HasSuffix(lower, ".log")
}

// readFile returns the parsed session of a log file, and parses the file again
// only when its size or modification time changed since the last parse.
func (reader *Reader) readFile(ctx context.Context, root *os.Root, directory, name string) (Session, error) {
	session := newSession(name)
	file, err := root.Open(name)
	if err != nil {
		return session, fmt.Errorf("open Roblox log: %w", err)
	}
	defer func() { _ = file.Close() }()
	logFileStatus, err := file.Stat()
	if err != nil {
		return session, fmt.Errorf("inspect Roblox log: %w", err)
	}
	if !logFileStatus.Mode().IsRegular() {
		return session, errors.New("not a regular log file")
	}
	path := filepath.Join(directory, name)
	if parsed, isParsed := reader.parsedLogAt(path); isParsed && parsed.sizeBytes == logFileStatus.Size() && parsed.modifiedAt.Equal(logFileStatus.ModTime()) {
		return parsed.session, nil
	}
	session.SizeBytes = logFileStatus.Size()
	if err := parse(ctx, io.LimitReader(file, logFileStatus.Size()), &session); err != nil {
		return session, err
	}
	reader.rememberParsedLog(path, parsedLog{sizeBytes: logFileStatus.Size(), modifiedAt: logFileStatus.ModTime(), session: session})
	return session, nil
}

func (reader *Reader) parsedLogAt(path string) (parsedLog, bool) {
	reader.parsedLogMutex.Lock()
	defer reader.parsedLogMutex.Unlock()
	parsed, isParsed := reader.parsedLogByPath[path]
	return parsed, isParsed
}

func (reader *Reader) rememberParsedLog(path string, parsed parsedLog) {
	reader.parsedLogMutex.Lock()
	defer reader.parsedLogMutex.Unlock()
	reader.parsedLogByPath[path] = parsed
}

// forgetUnlistedLogs drops the parsed logs of files that Roblox removed.
func (reader *Reader) forgetUnlistedLogs(listedPaths map[string]bool) {
	reader.parsedLogMutex.Lock()
	defer reader.parsedLogMutex.Unlock()
	for path := range reader.parsedLogByPath {
		if !listedPaths[path] {
			delete(reader.parsedLogByPath, path)
		}
	}
}
