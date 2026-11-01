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
	"strings"
	"sync"
	"time"

	sqlite3 "github.com/mutecomm/go-sqlcipher/v4"
	"github.com/sleaze5/RobloxAccountManager/internal/games"
)

const formatVersion = 1
const freshFor = 5 * time.Minute
const retention = 30 * 24 * time.Hour

var errInvalidCache = errors.New("invalid cache format")

var schemas = [...]string{`CREATE TABLE metadata (kind TEXT NOT NULL, id INTEGER NOT NULL, payload TEXT NOT NULL, fetched_at INTEGER NOT NULL, PRIMARY KEY(kind, id)); CREATE INDEX metadata_age ON metadata(fetched_at); PRAGMA user_version = 1;`}
var _ = [1]struct{}{}[len(schemas)-formatVersion]

type User struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

type Store struct {
	mu      sync.Mutex
	path    string
	logger  *slog.Logger
	retryAt time.Time
}

func New(path string, logger *slog.Logger) *Store {
	return &Store{path: path, logger: logger}
}

func (store *Store) access(ctx context.Context, action func(*sql.DB) error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if time.Now().Before(store.retryAt) {
		return
	}
	err := store.run(ctx, action)
	if err == nil || ctx.Err() != nil {
		return
	}
	var sqliteErr sqlite3.Error
	if errors.Is(err, errInvalidCache) || errors.As(err, &sqliteErr) && (sqliteErr.Code == sqlite3.ErrCorrupt || sqliteErr.Code == sqlite3.ErrNotADB || sqliteErr.Code == sqlite3.ErrSchema || sqliteErr.Code == sqlite3.ErrError) {
		if removeErr := os.Remove(store.path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			store.logger.WarnContext(ctx, "damaged cache could not be removed; delete caches.db after closing the application", "operation", "cache-recover")
		}
	}
	store.retryAt = time.Now().Add(time.Minute)
	store.logger.WarnContext(ctx, "public metadata cache unavailable; using ID placeholders and retrying later", "operation", "cache-access")
}

func (store *Store) run(ctx context.Context, action func(*sql.DB) error) (err error) {
	if err := os.MkdirAll(filepath.Dir(store.path), 0o700); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}
	uriPath := filepath.ToSlash(store.path)
	if filepath.VolumeName(store.path) != "" && !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	u := url.URL{Scheme: "file", Path: uriPath}
	u.RawQuery = "_busy_timeout=250&_journal_mode=DELETE"
	db, err := sql.Open("sqlite3", u.String())
	if err != nil {
		return fmt.Errorf("open public cache: %w", err)
	}
	db.SetMaxOpenConns(1)
	defer func() {
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
	if version == 0 {
		if _, err := db.ExecContext(ctx, "BEGIN;"+schemas[0]+"COMMIT;"); err != nil {
			return fmt.Errorf("initialize public cache: %w", err)
		}
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM metadata WHERE fetched_at < ? OR rowid IN (SELECT rowid FROM metadata ORDER BY fetched_at DESC, rowid DESC LIMIT -1 OFFSET 10000)`, time.Now().Add(-retention).UnixMilli()); err != nil {
		return fmt.Errorf("expire public cache: %w", err)
	}
	return action(db)
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

func (store *Store) User(ctx context.Context, id int64) (User, bool) {
	user := User{Username: fmt.Sprintf("User %d", id)}
	fresh := store.read(ctx, "user", id, &user)
	return user, fresh
}

func (store *Store) SaveUser(ctx context.Context, id int64, username, displayName string) {
	store.write(ctx, "user", id, User{Username: username, DisplayName: displayName})
}

func (store *Store) MatchingUsers(ctx context.Context, search string) []int64 {
	ids := []int64{}
	store.access(ctx, func(db *sql.DB) error {
		rows, err := db.QueryContext(ctx, `SELECT id,payload FROM metadata WHERE kind='user'`)
		if err != nil {
			return fmt.Errorf("search cached users: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var id int64
			var payload string
			if err := rows.Scan(&id, &payload); err != nil {
				return fmt.Errorf("read cached user: %w", err)
			}
			var user User
			if err := json.Unmarshal([]byte(payload), &user); err != nil {
				return fmt.Errorf("%w: decode cached user: %w", errInvalidCache, err)
			}
			if strings.Contains(strings.ToLower(user.Username), strings.ToLower(search)) || strings.Contains(strings.ToLower(user.DisplayName), strings.ToLower(search)) {
				ids = append(ids, id)
			}
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("read matching users: %w", err)
		}
		return nil
	})
	return ids
}
