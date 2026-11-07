package vault

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"os"

	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
	"github.com/sleaze5/RobloxAccountManager/internal/logging"
)

func (manager *Manager) completeUnlock(ctx context.Context, usedNext bool, password string) error {
	err := os.Remove(manager.paths.Vault + ".next")
	if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	if err == nil {
		err = manager.promoteOrDiscardRotation(usedNext)
	}
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
	database, finish, err := manager.beginExclusive(ctx)
	if err != nil {
		return err
	}
	defer finish()
	var version int
	if err := database.QueryRowContext(ctx, `SELECT schema_version FROM vault_state WHERE singleton = 1`).Scan(&version); err != nil {
		return fmt.Errorf("read vault schema version: %w", err)
	}
	if version == schemaVersion {
		return nil
	}
	if err := manager.writeBackup(ctx, database); err != nil {
		return fmt.Errorf("back up vault before schema migration: %w", err)
	}
	if err := applySchemaMigrations(ctx, database, version); err != nil {
		return err
	}
	logging.Diagnostic(manager.logger, "vault schema migrated", "operation", "vault-migrate", "from_schema_version", version, "schema_version", schemaVersion)
	return nil
}

func applySchemaMigrations(ctx context.Context, database *sql.DB, from int) error {
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin vault schema migration: %w", err)
	}
	defer transaction.Rollback()
	if err := runSchemaMigrations(ctx, transaction, from); err != nil {
		return err
	}
	if err := BumpRevision(ctx, transaction); err != nil {
		return err
	}
	var violations int
	if err := transaction.QueryRowContext(ctx, `SELECT count(*) FROM pragma_foreign_key_check`).Scan(&violations); err != nil || violations != 0 {
		return errors.New("migrated vault schema failed the foreign key check")
	}
	var integrity string
	if err := transaction.QueryRowContext(ctx, `PRAGMA quick_check`).Scan(&integrity); err != nil || integrity != "ok" {
		return errors.New("migrated vault schema failed the integrity check")
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit vault schema migration: %w", err)
	}
	return nil
}

func runSchemaMigrations(ctx context.Context, transaction *sql.Tx, from int) error {
	if _, err := transaction.ExecContext(ctx, `PRAGMA defer_foreign_keys = ON`); err != nil {
		return fmt.Errorf("defer vault foreign key checks: %w", err)
	}
	if from < 3 {
		if _, err := transaction.ExecContext(ctx, `CREATE TABLE migration_account_names AS SELECT id, username, display_name FROM accounts`); err != nil {
			return fmt.Errorf("preserve saved account names: %w", err)
		}
	}
	for version := from; version < schemaVersion; version++ {
		if err := schemaMigrations[version-1](ctx, transaction); err != nil {
			return fmt.Errorf("migrate vault schema from version %d: %w", version, err)
		}
	}
	if from < 3 {
		if _, err := transaction.ExecContext(ctx, `UPDATE accounts SET username = (SELECT username FROM migration_account_names WHERE id = accounts.id), display_name = (SELECT display_name FROM migration_account_names WHERE id = accounts.id); DROP TABLE migration_account_names;`); err != nil {
			return fmt.Errorf("restore preserved account names: %w", err)
		}
	}
	if _, err := transaction.ExecContext(ctx, `UPDATE vault_state SET schema_version = ? WHERE singleton = 1`, schemaVersion); err != nil {
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
	params := calibrateKDF(password)
	vaultKey, identifier := manager.currentKey()
	defer wipe(vaultKey)
	upgraded, err := createKeyFile(password, key.Hint, identifier, vaultKey, params)
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
		return errors.New("automatic unlock file is corrupted")
	}
	version := binary.BigEndian.Uint16(data[len(autoMagic):])
	if version >= autoFormatVersion {
		return nil
	}
	vaultKey, identifier := manager.currentKey()
	defer wipe(vaultKey)
	encoded, err := encodeAutoUnlock(manager.protector, identifier, vaultKey)
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
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	return append([]byte(nil), manager.vaultKey...), manager.vaultID
}
