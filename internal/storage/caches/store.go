package caches

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	"github.com/sleaze5/RobloxAccountManager/internal/games"
	"github.com/sleaze5/RobloxAccountManager/internal/logging"
)

const formatVersion = 2
const freshFor = 5 * time.Minute
const retention = 30 * 24 * time.Hour

var errInvalidCache = errors.New("invalid cache format")

var schemas = [...]string{`CREATE TABLE metadata (kind TEXT NOT NULL, id INTEGER NOT NULL, payload TEXT NOT NULL, fetched_at INTEGER NOT NULL, PRIMARY KEY(kind, id)); CREATE INDEX metadata_age ON metadata(fetched_at); PRAGMA user_version = 1;`, `DELETE FROM metadata WHERE kind = 'user'; PRAGMA user_version = 2;`}
var _ = [1]struct{}{}[len(schemas)-formatVersion]

type Store struct {
	mu      sync.Mutex
	path    string
	logger  *slog.Logger
	retryAt time.Time
	db      *sql.DB
	closed  bool
	pruneAt time.Time
	writes  int
}

func New(path string, logger *slog.Logger) *Store {
	logging.Diagnostic(logger, "cache format supported", "operation", "cache-load", "database", "caches.db", "supported_format_version", formatVersion, "encrypted", false)
	return &Store{path: path, logger: logger}
}

func (store *Store) access(ctx context.Context, action func(*sql.DB) error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.closed || time.Now().Before(store.retryAt) {
		return
	}
	err := store.run(ctx, action)
	if err == nil || ctx.Err() != nil {
		return
	}
	if closeErr := store.closeDatabase(); closeErr != nil {
		store.logger.WarnContext(ctx, "public cache could not be closed", "operation", "cache-recover", "error", closeErr)
		store.retryAt = time.Now().Add(time.Minute)
		return
	}
	if errors.Is(err, errInvalidCache) || errors.Is(err, sqlite3.CORRUPT) || errors.Is(err, sqlite3.NOTADB) {
		if removeErr := os.Remove(store.path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			store.logger.WarnContext(ctx, "damaged cache could not be removed; delete caches.db after closing the application", "operation", "cache-recover", "error", removeErr)
		}
	}
	store.retryAt = time.Now().Add(time.Minute)
	store.logger.WarnContext(ctx, "public metadata cache unavailable; using ID placeholders and retrying later", "operation", "cache-access", "error", err)
}

func (store *Store) run(ctx context.Context, action func(*sql.DB) error) error {
	if store.db != nil {
		if _, err := os.Stat(store.path); errors.Is(err, os.ErrNotExist) {
			if err := store.closeDatabase(); err != nil {
				return err
			}
		} else if err != nil {
			return fmt.Errorf("inspect cache file: %w", err)
		}
	}
	if store.db == nil {
		if err := store.open(ctx); err != nil {
			return err
		}
	}
	if time.Now().After(store.pruneAt) || store.writes >= 256 {
		if _, err := store.db.ExecContext(ctx, `DELETE FROM metadata WHERE fetched_at < ? OR rowid IN (SELECT rowid FROM metadata ORDER BY fetched_at DESC, rowid DESC LIMIT -1 OFFSET 10000)`, time.Now().Add(-retention).UnixMilli()); err != nil {
			return fmt.Errorf("expire public cache: %w", err)
		}
		store.pruneAt = time.Now().Add(5 * time.Minute)
		store.writes = 0
	}
	return action(store.db)
}

func (store *Store) Close() error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.closed = true
	return store.closeDatabase()
}

func (store *Store) closeDatabase() error {
	if store.db == nil {
		return nil
	}
	if err := store.db.Close(); err != nil {
		return fmt.Errorf("close public cache: %w", err)
	}
	store.db = nil
	return nil
}

func (store *Store) open(ctx context.Context) (err error) {
	if err := os.MkdirAll(filepath.Dir(store.path), 0o700); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}
	// The SQLite VFS opens the URI path as is, so a Windows path keeps its drive
	// letter first, as in file:C:/Users/..., without a leading slash.
	u := url.URL{Scheme: "file", Opaque: (&url.URL{Path: filepath.ToSlash(store.path)}).EscapedPath()}
	u.RawQuery = url.Values{"_pragma": {"busy_timeout(250)", "journal_mode(delete)", "secure_delete(1)"}}.Encode()
	db, err := sql.Open("sqlite3", u.String())
	if err != nil {
		return fmt.Errorf("open public cache: %w", err)
	}
	db.SetMaxOpenConns(1)
	defer func() {
		if err == nil {
			return
		}
		if closeErr := db.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close public cache: %w", closeErr))
		}
	}()
	var version int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return fmt.Errorf("read cache version: %w", err)
	}
	if version < 0 || version > formatVersion {
		return errInvalidCache
	}
	fromVersion := version
	for ; version < formatVersion; version++ {
		if _, err := db.ExecContext(ctx, "BEGIN;"+schemas[version]+"COMMIT;"); err != nil {
			return fmt.Errorf("initialize public cache: %w", err)
		}
	}
	store.db = db
	store.pruneAt = time.Time{}
	store.recordLoaded(fromVersion)
	return nil
}

func (store *Store) recordLoaded(fromVersion int) {
	if fromVersion == 0 {
		logging.Diagnostic(store.logger, "cache database created", "operation", "cache-load", "format_version", formatVersion)
	} else if fromVersion < formatVersion {
		logging.Diagnostic(store.logger, "cache database migrated", "operation", "cache-load", "from_format_version", fromVersion, "format_version", formatVersion)
	}
	logging.Diagnostic(store.logger, "cache database loaded", "operation", "cache-load", "database", "caches.db", "format_version", formatVersion, "source_format_version", fromVersion, "encrypted", false)
}

func (store *Store) read(ctx context.Context, kind string, id int64, target any) (fresh bool) {
	store.access(ctx, func(db *sql.DB) error {
		var payload string
		var fetched int64
		err := db.QueryRowContext(ctx, `SELECT payload, fetched_at FROM metadata WHERE kind = ? AND id = ?`, kind, id).Scan(&payload, &fetched)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read public metadata: %w", err)
		}
		if err := json.Unmarshal([]byte(payload), target); err != nil {
			return fmt.Errorf("%w: decode public metadata: %w", errInvalidCache, err)
		}
		fresh = time.Since(time.UnixMilli(fetched)) < freshFor
		return nil
	})
	return fresh
}

func (store *Store) write(ctx context.Context, kind string, id int64, value any) {
	store.access(ctx, func(db *sql.DB) error {
		store.writes++
		payload, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("encode public metadata: %w", err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO metadata(kind,id,payload,fetched_at) VALUES(?,?,?,?) ON CONFLICT(kind,id) DO UPDATE SET payload=excluded.payload,fetched_at=excluded.fetched_at`, kind, id, string(payload), time.Now().UnixMilli()); err != nil {
			return fmt.Errorf("write public metadata: %w", err)
		}
		return nil
	})
}

func (store *Store) Place(ctx context.Context, id int64) (games.Place, bool) {
	place := games.Place{PlaceID: id, Name: fmt.Sprintf("Place %d", id)}
	fresh := store.read(ctx, "place", id, &place)
	place.PlaceID, place.Nickname = id, ""
	return place, fresh
}

func (store *Store) SavePlace(ctx context.Context, place games.Place) {
	place.Nickname = ""
	store.write(ctx, "place", place.PlaceID, place)
}
