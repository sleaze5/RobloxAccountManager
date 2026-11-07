package vault

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/sleaze5/RobloxAccountManager/internal/logging"
)

const schemaVersion = 7

var schemaMigrations = [...]func(context.Context, *sql.Tx) error{addAccountPasswords, splitPublicMetadata, restoreAccountNames, addPendingSettingsImport, addSavedPrivateServers, saveOnePrivateServerCode}

var _ = [1]struct{}{}[len(schemaMigrations)-(schemaVersion-1)]

func splitPublicMetadata(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `
CREATE TABLE accounts_new (
    id INTEGER PRIMARY KEY,
    roblox_user_id INTEGER NOT NULL UNIQUE CHECK (roblox_user_id > 0),
    display_order INTEGER NOT NULL UNIQUE CHECK (display_order >= 0),
    created_at_ms INTEGER NOT NULL,
    updated_at_ms INTEGER NOT NULL
);
INSERT INTO accounts_new SELECT id, roblox_user_id, display_order, created_at_ms, updated_at_ms FROM accounts;
CREATE TABLE saved_sessions AS SELECT account_id, roblosecurity, cookie_expires_at_ms, state, state_reason, secret_version, imported_at_ms, replaced_at_ms, rotated_at_ms, last_validated_at_ms, last_success_at_ms, last_auth_failure_at_ms, updated_at_ms, password FROM account_sessions;
CREATE TABLE saved_account_tags AS SELECT account_id, tag_id, created_at_ms FROM account_tags;
DELETE FROM account_sessions;
DELETE FROM account_tags;
DROP TABLE accounts;
ALTER TABLE accounts_new RENAME TO accounts;
INSERT INTO account_sessions(account_id, roblosecurity, cookie_expires_at_ms, state, state_reason, secret_version, imported_at_ms, replaced_at_ms, rotated_at_ms, last_validated_at_ms, last_success_at_ms, last_auth_failure_at_ms, updated_at_ms, password) SELECT account_id, roblosecurity, cookie_expires_at_ms, state, state_reason, secret_version, imported_at_ms, replaced_at_ms, rotated_at_ms, last_validated_at_ms, last_success_at_ms, last_auth_failure_at_ms, updated_at_ms, password FROM saved_sessions;
INSERT INTO account_tags(account_id, tag_id, created_at_ms) SELECT account_id, tag_id, created_at_ms FROM saved_account_tags;
DROP TABLE saved_sessions;
DROP TABLE saved_account_tags;
CREATE TABLE favorite_places_new (
    place_id INTEGER PRIMARY KEY CHECK (place_id > 0),
    nickname TEXT NOT NULL DEFAULT '' CHECK (length(nickname) <= 50),
    display_order INTEGER NOT NULL,
    created_at_ms INTEGER NOT NULL
);
INSERT INTO favorite_places_new SELECT place_id, nickname, display_order, created_at_ms FROM favorite_places;
DROP TABLE favorite_places;
ALTER TABLE favorite_places_new RENAME TO favorite_places;
`); err != nil {
		return fmt.Errorf("remove rebuildable public metadata: %w", err)
	}
	return nil
}

func addAccountPasswords(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `ALTER TABLE account_sessions ADD COLUMN password TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("add saved account passwords: %w", err)
	}
	return nil
}

const initialSchema = `
CREATE TABLE vault_state (
    singleton                               INTEGER PRIMARY KEY CHECK (singleton = 1),
    vault_id                                BLOB NOT NULL CHECK (length(vault_id) = 16),
    schema_version                          INTEGER NOT NULL,
    storage_revision                        INTEGER NOT NULL DEFAULT 0 CHECK (storage_revision >= 0)
);

CREATE TABLE accounts (
    id                  INTEGER PRIMARY KEY,
    roblox_user_id      INTEGER NOT NULL UNIQUE CHECK (roblox_user_id > 0),
    username            TEXT NOT NULL CHECK (username <> ''),
    display_name        TEXT NOT NULL DEFAULT '',
    display_order       INTEGER NOT NULL UNIQUE CHECK (display_order >= 0),
    created_at_ms       INTEGER NOT NULL,
    updated_at_ms       INTEGER NOT NULL
);

CREATE TABLE account_sessions (
    account_id                  INTEGER PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    roblosecurity               TEXT NOT NULL CHECK (roblosecurity <> ''),
    cookie_expires_at_ms        INTEGER,
    state                       TEXT NOT NULL CHECK (state IN ('active', 'unknown', 'reauth_required', 'challenged', 'disabled')),
    state_reason                TEXT NOT NULL DEFAULT '',
    secret_version              INTEGER NOT NULL DEFAULT 1 CHECK (secret_version > 0),
    imported_at_ms              INTEGER NOT NULL,
    replaced_at_ms              INTEGER,
    rotated_at_ms               INTEGER,
    last_validated_at_ms        INTEGER,
    last_success_at_ms          INTEGER,
    last_auth_failure_at_ms     INTEGER,
    updated_at_ms               INTEGER NOT NULL
);

CREATE INDEX account_sessions_state_idx ON account_sessions(state);

CREATE TABLE tags (
    id                  INTEGER PRIMARY KEY,
    name                TEXT NOT NULL COLLATE NOCASE UNIQUE,
    kind                TEXT NOT NULL DEFAULT 'custom' CHECK (kind IN ('custom', 'favorite')),
    created_at_ms       INTEGER NOT NULL,
    updated_at_ms       INTEGER NOT NULL
);

CREATE UNIQUE INDEX tags_one_favorite_idx ON tags(kind) WHERE kind = 'favorite';

CREATE TABLE account_tags (
    account_id          INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    tag_id              INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at_ms       INTEGER NOT NULL,
    PRIMARY KEY (account_id, tag_id)
);

CREATE INDEX account_tags_tag_id_idx ON account_tags(tag_id, account_id);

CREATE TABLE favorite_places (
    place_id            INTEGER PRIMARY KEY CHECK (place_id > 0),
    universe_id         INTEGER NOT NULL CHECK (universe_id > 0),
    name                TEXT NOT NULL CHECK (name <> ''),
    creator_id          INTEGER NOT NULL DEFAULT 0 CHECK (creator_id >= 0),
    creator_name        TEXT NOT NULL DEFAULT '',
    creator_type        TEXT NOT NULL DEFAULT '' CHECK (creator_type IN ('', 'User', 'Group')),
    creator_verified    INTEGER NOT NULL DEFAULT 0 CHECK (creator_verified IN (0, 1)),
    icon_url            TEXT NOT NULL DEFAULT '',
    nickname            TEXT NOT NULL DEFAULT '' CHECK (length(nickname) <= 50),
    display_order       INTEGER NOT NULL,
    created_at_ms       INTEGER NOT NULL
);
`

func initializeSchema(ctx context.Context, db *sql.DB, vaultID [vaultIDBytes]byte) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin vault schema initialization: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, initialSchema); err != nil {
		return fmt.Errorf("initialize vault schema: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO vault_state(singleton, vault_id, schema_version) VALUES(1, ?, 1)`, vaultID[:]); err != nil {
		return fmt.Errorf("initialize vault state: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO tags(name, kind, created_at_ms, updated_at_ms) VALUES('Favorites', 'favorite', strftime('%s', 'now') * 1000, strftime('%s', 'now') * 1000)`); err != nil {
		return fmt.Errorf("initialize favorite tag: %w", err)
	}
	if err := runSchemaMigrations(ctx, tx, 1); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit vault schema initialization: %w", err)
	}
	return nil
}

func verifySchema(ctx context.Context, db *sql.DB, expectedID [vaultIDBytes]byte, logger *slog.Logger) error {
	var identifier []byte
	var version int
	if err := db.QueryRowContext(ctx, `SELECT vault_id, schema_version FROM vault_state WHERE singleton = 1`).Scan(&identifier, &version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUnsupportedSchema
		}
		return fmt.Errorf("read vault state: %w", err)
	}
	logging.Diagnostic(logger, "database schema read", "operation", "vault-load", "schema_version", version, "supported_schema_version", schemaVersion)
	if version < 1 || version > schemaVersion {
		return fmt.Errorf("%w: schema version %d", ErrUnsupportedSchema, version)
	}
	if len(identifier) != vaultIDBytes || expectedID != [vaultIDBytes]byte(identifier) {
		return ErrVaultMismatch
	}
	return nil
}

func restoreAccountNames(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `ALTER TABLE accounts ADD COLUMN username TEXT NOT NULL DEFAULT ''; ALTER TABLE accounts ADD COLUMN display_name TEXT NOT NULL DEFAULT '';`); err != nil {
		return fmt.Errorf("restore account identity columns: %w", err)
	}
	return nil
}

// addPendingSettingsImport stores imported settings in the same transaction as the imported vault data,
// so the application can finish saving them after a crash.
func addPendingSettingsImport(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `
CREATE TABLE pending_settings_import (
    singleton   INTEGER PRIMARY KEY CHECK (singleton = 1),
    settings    TEXT NOT NULL,
    sections    TEXT NOT NULL
)`); err != nil {
		return fmt.Errorf("add pending settings import: %w", err)
	}
	return nil
}

func addSavedPrivateServers(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `
CREATE TABLE saved_private_servers (
    id              INTEGER PRIMARY KEY,
    place_id        INTEGER NOT NULL CHECK (place_id > 0),
    access_code     TEXT NOT NULL CHECK (access_code <> '' AND length(access_code) <= 512),
    link_code       TEXT NOT NULL DEFAULT '' CHECK (length(link_code) <= 512),
    name            TEXT NOT NULL CHECK (length(name) <= 50),
    created_at_ms   INTEGER NOT NULL,
    UNIQUE (place_id, access_code)
)`); err != nil {
		return fmt.Errorf("add saved private servers: %w", err)
	}
	return nil
}

// saveOnePrivateServerCode keeps the link code of each saved private server, and the
// access code only when the link code is unknown.
func saveOnePrivateServerCode(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `
CREATE TABLE saved_private_servers_new (
    id              INTEGER PRIMARY KEY,
    place_id        INTEGER NOT NULL CHECK (place_id > 0),
    access_code     TEXT NOT NULL DEFAULT '' CHECK (length(access_code) <= 512),
    link_code       TEXT NOT NULL DEFAULT '' CHECK (length(link_code) <= 512),
    name            TEXT NOT NULL CHECK (length(name) <= 50),
    created_at_ms   INTEGER NOT NULL,
    CHECK ((access_code = '') <> (link_code = ''))
);
CREATE UNIQUE INDEX saved_private_servers_access_code ON saved_private_servers_new(place_id, access_code) WHERE access_code <> '';
CREATE UNIQUE INDEX saved_private_servers_link_code ON saved_private_servers_new(place_id, link_code) WHERE link_code <> '';
INSERT OR IGNORE INTO saved_private_servers_new(id, place_id, access_code, link_code, name, created_at_ms)
SELECT id, place_id, CASE WHEN link_code = '' THEN access_code ELSE '' END, link_code, name, created_at_ms FROM saved_private_servers ORDER BY id;
DROP TABLE saved_private_servers;
ALTER TABLE saved_private_servers_new RENAME TO saved_private_servers;
`); err != nil {
		return fmt.Errorf("save one code per private server: %w", err)
	}
	return nil
}
