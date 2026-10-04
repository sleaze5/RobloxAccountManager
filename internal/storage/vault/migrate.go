package vault

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"os"

	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
)

func (manager *Manager) completeUnlock(ctx context.Context, usedNext bool, password string) error {
	err := manager.promoteOrDiscardRotation(usedNext)
	if err == nil {
		err = manager.migrateSchema(ctx)
	}
	if err != nil {
		_ = manager.Close()
		return err
	}
	manager.upgradeKeyFiles(password)
	return nil
}

func (manager *Manager) migrateSchema(ctx context.Context) error {
	db, finish, err := manager.beginExclusive(ctx)
	if err != nil {
		return err
	}
	defer finish()
	var version int
	if err := db.QueryRowContext(ctx, `SELECT schema_version FROM vault_state WHERE singleton = 1`).Scan(&version); err != nil {
		return fmt.Errorf("read vault schema version: %w", err)
	}
	if version == schemaVersion {
		return nil
	}
	if err := manager.writeBackup(ctx, db); err != nil {
		return fmt.Errorf("back up vault before schema migration: %w", err)
	}
	if err := applySchemaMigrations(ctx, db, version); err != nil {
		return err
	}
	manager.logger.Info("vault schema migrated", "operation", "vault-migrate", "from_schema_version", version, "schema_version", schemaVersion)
	return nil
}

func applySchemaMigrations(ctx context.Context, db *sql.DB, from int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin vault schema migration: %w", err)
	}
	defer tx.Rollback()
	if err := runSchemaMigrations(ctx, tx, from); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE vault_state SET storage_revision = storage_revision + 1 WHERE singleton = 1`); err != nil {
		return fmt.Errorf("record vault storage revision: %w", err)
	}
	var violations int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM pragma_foreign_key_check`).Scan(&violations); err != nil || violations != 0 {
		return fmt.Errorf("migrated vault schema failed the foreign key check")
	}
	var integrity string
	if err := tx.QueryRowContext(ctx, `PRAGMA quick_check`).Scan(&integrity); err != nil || integrity != "ok" {
		return fmt.Errorf("migrated vault schema failed the integrity check")
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit vault schema migration: %w", err)
	}
	return nil
}

func runSchemaMigrations(ctx context.Context, tx *sql.Tx, from int) error {
	if _, err := tx.ExecContext(ctx, `PRAGMA defer_foreign_keys = ON`); err != nil {
		return fmt.Errorf("defer vault foreign key checks: %w", err)
	}
	for version := from; version < schemaVersion; version++ {
		if err := schemaMigrations[version-1](ctx, tx); err != nil {
			return fmt.Errorf("migrate vault schema from version %d: %w", version, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE vault_state SET schema_version = ? WHERE singleton = 1`, schemaVersion); err != nil {
		return fmt.Errorf("record vault schema version: %w", err)
	}
	return nil
}

func (manager *Manager) upgradeKeyFiles(password string) {
	if err := manager.upgradeAutoUnlock(); err != nil {
		manager.logger.Warn("automatic unlock file upgrade failed", "operation", "vault-migrate", "error", err)
	}
	if password == "" {
		return
	}
	if err := manager.upgradeKeyFile(password); err != nil {
		manager.logger.Warn("vault key file upgrade failed", "operation", "vault-migrate", "error", err)
	}
}

func (manager *Manager) upgradeKeyFile(password string) error {
	key, err := readKeyFile(manager.paths.VaultKey)
	if err != nil || key.Version == keyFormatVersion {
		return err
	}
	params, err := calibrateKDF(password)
	if err != nil {
		return err
	}
	dek, identifier := manager.currentKey()
	defer wipe(dek)
	upgraded, err := createKeyFile(password, key.Hint, identifier, dek, params)
	if err != nil {
		return err
	}
	encoded, err := encodeKeyFile(upgraded)
	if err != nil {
		return err
	}
	if err := appdata.WritePrivateFile(manager.paths.VaultKey, encoded); err != nil {
		return err
	}
	manager.logger.Info("vault key file upgraded", "operation", "vault-migrate", "from_key_file_version", key.Version, "key_file_version", keyFormatVersion)
	return nil
}

func (manager *Manager) upgradeAutoUnlock() error {
	data, err := readBounded(manager.paths.AutoUnlockKey, autoFileLimit)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) < len(autoMagic)+2 {
		return fmt.Errorf("automatic unlock file is corrupted")
	}
	version := binary.BigEndian.Uint16(data[len(autoMagic):])
	if version >= autoFormatVersion {
		return nil
	}
	dek, identifier := manager.currentKey()
	defer wipe(dek)
	encoded, err := encodeAutoUnlock(manager.protector, identifier, dek)
	if err != nil {
		return err
	}
	if err := appdata.WritePrivateFile(manager.paths.AutoUnlockKey, encoded); err != nil {
		return err
	}
	manager.logger.Info("automatic unlock file upgraded", "operation", "vault-migrate", "from_auto_unlock_version", version, "auto_unlock_version", autoFormatVersion)
	return nil
}

func (manager *Manager) currentKey() ([]byte, [vaultIDBytes]byte) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	return append([]byte(nil), manager.dek...), manager.vaultID
}
