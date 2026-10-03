package vault

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/sleaze5/RobloxAccountManager/internal/logging"
)

const schemaVersion = 1

// schemaMigrations[i] migrates a version i+1 vault database to version i+2.
var schemaMigrations = [...]func(context.Context, *sql.Tx) error{}

// This fails to compile unless every earlier schemaVersion has one migration.
var _ = [1]struct{}{}[len(schemaMigrations)-(schemaVersion-1)]

// initialSchema is the version 1 schema. Do not change it. Change the schema
// through schemaMigrations.
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
	// A new vault starts as version 1 and reaches schemaVersion through the same
	// migrations as an existing vault.
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
	if len(identifier) != vaultIDBytes || !sameVaultID(expectedID, [vaultIDBytes]byte(identifier)) {
		return ErrVaultMismatch
	}
	return nil
}
