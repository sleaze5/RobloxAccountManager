package vault

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	sqlite3driver "github.com/ncruces/go-sqlite3/driver"
	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
)

const (
	backupTimeFormat = "20060102T150405.000000000Z"
	retainedBackups  = 3
)

type BackupInfo struct {
	Name        string `json:"name"`
	CreatedAtMS int64  `json:"createdAtMs"`
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// BumpRevision must run in every transaction that changes vault data, so that
// MaybeCreateBackup sees the change.
func BumpRevision(ctx context.Context, db execer) error {
	if _, err := db.ExecContext(ctx, `UPDATE vault_state SET storage_revision = storage_revision + 1 WHERE singleton = 1`); err != nil {
		return fmt.Errorf("update storage revision: %w", err)
	}
	return nil
}

func (manager *Manager) MaybeCreateBackup(ctx context.Context) error {
	db, release, err := manager.Lease(ctx)
	if err != nil {
		return err
	}
	var revision int64
	if err := db.QueryRowContext(ctx, `SELECT storage_revision FROM vault_state WHERE singleton = 1`).Scan(&revision); err != nil {
		release()
		return fmt.Errorf("read storage revision: %w", err)
	}
	release()
	backups, err := manager.ListBackups()
	if err != nil {
		return err
	}
	if len(backups) > 0 {
		newest := backups[0]
		if time.Since(time.UnixMilli(newest.CreatedAtMS)) < 24*time.Hour {
			return nil
		}
		backupRevision, verifyErr := manager.backupRevision(ctx, newest.Name)
		if verifyErr == nil && backupRevision == revision {
			return nil
		}
	}
	return manager.CreateBackup(ctx)
}

func (manager *Manager) CreateBackup(ctx context.Context) error {
	db, release, err := manager.Lease(ctx)
	if err != nil {
		return err
	}
	defer release()
	return manager.writeBackup(ctx, db)
}

func (manager *Manager) writeBackup(ctx context.Context, db *sql.DB) error {
	manager.backupMu.Lock()
	defer manager.backupMu.Unlock()
	dek, identifier := manager.currentKey()
	defer wipe(dek)
	if err := appdata.PreparePrivateDirectory(manager.paths.BackupsRoot); err != nil {
		return err
	}
	name := time.Now().UTC().Format(backupTimeFormat)
	temporary := filepath.Join(manager.paths.BackupsRoot, ".next-"+name)
	final := filepath.Join(manager.paths.BackupsRoot, name)
	if err := appdata.PreparePrivateDirectory(temporary); err != nil {
		return err
	}
	ok := false
	defer func() {
		if !ok {
			_ = manager.removeManaged(temporary, manager.paths.BackupsRoot)
		}
	}()
	destination := filepath.Join(temporary, "vault.db")
	if err := snapshotDatabase(ctx, db, destination, dek); err != nil {
		return err
	}
	if err := appdata.RestrictFile(destination); err != nil {
		return err
	}
	keyData, err := readBounded(manager.paths.VaultKey, keyFileLimit)
	if err != nil {
		return err
	}
	if err := appdata.WritePrivateFile(filepath.Join(temporary, "vault.key"), keyData); err != nil {
		return err
	}
	verified, err := openAndVerify(ctx, destination, dek, identifier, manager.logger)
	if err != nil {
		return fmt.Errorf("verify backup: %w", err)
	}
	if err := verified.Close(); err != nil {
		return fmt.Errorf("close verified backup: %w", err)
	}
	if err := os.Rename(temporary, final); err != nil {
		return fmt.Errorf("publish backup: %w", err)
	}
	ok = true
	return manager.trimBackups()
}

func snapshotDatabase(ctx context.Context, source *sql.DB, destinationPath string, dek []byte) error {
	destination, err := databaseDSN(destinationPath, dek)
	if err != nil {
		return err
	}
	sourceConn, err := source.Conn(ctx)
	if err != nil {
		return fmt.Errorf("open backup source: %w", err)
	}
	defer func() { _ = sourceConn.Close() }()
	err = sourceConn.Raw(func(driverConn any) error {
		sourceSQLite, ok := driverConn.(sqlite3driver.Conn)
		if !ok {
			return errors.New("SQLite backup source is unavailable")
		}
		backup, err := sourceSQLite.Raw().BackupInit("main", destination)
		if err != nil {
			return fmt.Errorf("start SQLite backup: %w", err)
		}
		for {
			done, stepErr := backup.Step(128)
			if stepErr != nil {
				_ = backup.Close()
				return fmt.Errorf("copy SQLite backup pages: %w", stepErr)
			}
			if done {
				return backup.Close()
			}
			select {
			case <-ctx.Done():
				_ = backup.Close()
				return ctx.Err()
			case <-time.After(5 * time.Millisecond):
			}
		}
	})
	if err != nil {
		return fmt.Errorf("copy vault snapshot: %w", err)
	}
	return nil
}

func (manager *Manager) ListBackups() ([]BackupInfo, error) {
	manager.backupMu.Lock()
	defer manager.backupMu.Unlock()
	return manager.listBackups()
}

func (manager *Manager) listBackups() ([]BackupInfo, error) {
	entries, err := os.ReadDir(manager.paths.BackupsRoot)
	if errors.Is(err, os.ErrNotExist) {
		return []BackupInfo{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list vault backups: %w", err)
	}
	result := make([]BackupInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), ".next-") {
			if err := manager.removeManaged(filepath.Join(manager.paths.BackupsRoot, entry.Name()), manager.paths.BackupsRoot); err != nil {
				return nil, err
			}
			continue
		}
		created, err := time.Parse(backupTimeFormat, entry.Name())
		if err != nil {
			continue
		}
		if !fileExists(filepath.Join(manager.paths.BackupsRoot, entry.Name(), "vault.db")) || !fileExists(filepath.Join(manager.paths.BackupsRoot, entry.Name(), "vault.key")) {
			continue
		}
		if _, err := readKeyFile(filepath.Join(manager.paths.BackupsRoot, entry.Name(), "vault.key")); errors.Is(err, ErrLegacyVault) {
			continue
		}
		result = append(result, BackupInfo{Name: entry.Name(), CreatedAtMS: created.UnixMilli()})
	}
	sort.Slice(result, func(left, right int) bool { return result[left].Name > result[right].Name })
	return result, nil
}

func (manager *Manager) trimBackups() error {
	backups, err := manager.listBackups()
	if err != nil {
		return err
	}
	if len(backups) <= retainedBackups {
		return nil
	}
	for _, backup := range backups[retainedBackups:] {
		if err := manager.removeManaged(filepath.Join(manager.paths.BackupsRoot, backup.Name), manager.paths.BackupsRoot); err != nil {
			return err
		}
	}
	return nil
}

func (manager *Manager) PurgeBackups() error {
	manager.backupMu.Lock()
	defer manager.backupMu.Unlock()
	entries, err := os.ReadDir(manager.paths.BackupsRoot)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read backups: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		_, timestampErr := time.Parse(backupTimeFormat, entry.Name())
		if timestampErr != nil && !strings.HasPrefix(entry.Name(), ".next-") {
			continue
		}
		if err := manager.removeManaged(filepath.Join(manager.paths.BackupsRoot, entry.Name()), manager.paths.BackupsRoot); err != nil {
			return err
		}
	}
	return nil
}

func (manager *Manager) removeManaged(path, root string) error {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == "." || relative == ".." || filepath.IsAbs(relative) || len(relative) >= 3 && relative[:3] == ".."+string(os.PathSeparator) {
		return errors.New("refusing to remove an unmanaged path")
	}
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove managed path: %w", err)
	}
	return nil
}

func (manager *Manager) backupRevision(ctx context.Context, name string) (int64, error) {
	path, err := manager.backupPath(name)
	if err != nil {
		return 0, err
	}
	dek, identifier := manager.currentKey()
	defer wipe(dek)
	db, err := openAndVerify(ctx, filepath.Join(path, "vault.db"), dek, identifier, manager.logger)
	if err != nil {
		return 0, err
	}
	defer func() { _ = db.Close() }()
	var revision int64
	if err := db.QueryRowContext(ctx, `SELECT storage_revision FROM vault_state WHERE singleton = 1`).Scan(&revision); err != nil {
		return 0, fmt.Errorf("read backup storage revision: %w", err)
	}
	return revision, nil
}

func (manager *Manager) RestoreBackup(ctx context.Context, name, password string) error {
	if manager.Unlocked() {
		return errors.New("lock the vault before restoring a backup")
	}
	backup, err := manager.backupPath(name)
	if err != nil {
		return err
	}
	key, err := readKeyFile(filepath.Join(backup, "vault.key"))
	if err != nil {
		return err
	}
	dek, err := key.unwrap(password)
	if err != nil {
		return err
	}
	defer wipe(dek)
	db, err := openAndVerify(ctx, filepath.Join(backup, "vault.db"), dek, key.VaultID, manager.logger)
	if err != nil {
		return err
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("close restored vault: %w", err)
	}
	if err := os.Remove(manager.paths.AutoUnlockKey); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove automatic unlock file: %w", err)
	}
	if _, err := manager.quarantineCurrentVault(manager.paths.Vault, manager.paths.VaultKey); err != nil {
		return err
	}
	if err := copyPrivateFile(filepath.Join(backup, "vault.db"), manager.paths.Vault); err != nil {
		return err
	}
	if err := copyPrivateFile(filepath.Join(backup, "vault.key"), manager.paths.VaultKey); err != nil {
		return err
	}
	return nil
}

func (manager *Manager) backupPath(name string) (string, error) {
	if _, err := time.Parse(backupTimeFormat, name); err != nil {
		return "", errors.New("invalid backup selection")
	}
	path := filepath.Join(manager.paths.BackupsRoot, name)
	relative, err := filepath.Rel(manager.paths.BackupsRoot, path)
	if err != nil || relative != name {
		return "", errors.New("invalid backup selection")
	}
	return path, nil
}

func copyPrivateFile(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open backup file: %w", err)
	}
	defer func() { _ = input.Close() }()
	output, err := os.CreateTemp(filepath.Dir(destination), "."+filepath.Base(destination)+".restore-*")
	if err != nil {
		return fmt.Errorf("create restore file: %w", err)
	}
	temporary := output.Name()
	ok := false
	defer func() {
		_ = output.Close()
		if !ok {
			_ = os.Remove(temporary)
		}
	}()
	if err := appdata.RestrictFile(temporary); err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		return fmt.Errorf("copy backup file: %w", err)
	}
	if err := output.Sync(); err != nil {
		return fmt.Errorf("flush restore file: %w", err)
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("close restore file: %w", err)
	}
	if err := appdata.ReplaceFile(temporary, destination); err != nil {
		return err
	}
	ok = true
	return appdata.RestrictFile(destination)
}

func (manager *Manager) quarantineCurrentVault(sources ...string) (string, error) {
	if err := appdata.PreparePrivateDirectory(manager.paths.RecoveryRoot); err != nil {
		return "", err
	}
	recovery := filepath.Join(manager.paths.RecoveryRoot, time.Now().UTC().Format(backupTimeFormat))
	if err := appdata.PreparePrivateDirectory(recovery); err != nil {
		return "", err
	}
	for _, source := range sources {
		if !fileExists(source) {
			continue
		}
		if err := os.Rename(source, filepath.Join(recovery, filepath.Base(source))); err != nil {
			return "", fmt.Errorf("quarantine current %q: %w", filepath.Base(source), err)
		}
	}
	return recovery, nil
}

func (manager *Manager) legacyVaultFiles() []string {
	return []string{manager.paths.Vault, manager.paths.VaultKey, manager.paths.AutoUnlockKey, manager.paths.Migration,
		manager.paths.VaultKey + ".next", manager.paths.AutoUnlockKey + ".next"}
}

// SetAsideLegacyVault moves a vault from a version before 0.10.0 into the
// recovery folder, so a new vault can be created. Nothing is deleted.
func (manager *Manager) SetAsideLegacyVault() error {
	_, err := manager.setAsideLegacyVault()
	return err
}

func (manager *Manager) setAsideLegacyVault() (string, error) {
	if manager.Unlocked() {
		return "", errors.New("lock the vault before setting it aside")
	}
	if manager.FileState() != FileStateLegacy {
		return "", errors.New("the vault is not from a version before 0.10.0")
	}
	recovery, err := manager.quarantineCurrentVault(manager.legacyVaultFiles()...)
	if err != nil {
		return "", err
	}
	manager.logger.Info("legacy vault set aside", "operation", "vault-legacy")
	return recovery, nil
}

// restoreLegacyVault moves a legacy vault back from the recovery folder after
// a migration failed to create the new vault.
func (manager *Manager) restoreLegacyVault(recovery string) error {
	if fileExists(manager.paths.Vault) || fileExists(manager.paths.VaultKey) {
		return errors.New("a vault exists, so the old vault stays in the recovery folder")
	}
	if err := manager.removeManaged(manager.paths.CreateStaging, manager.paths.VaultRoot); err != nil {
		return fmt.Errorf("remove vault staging data: %w", err)
	}
	for _, destination := range manager.legacyVaultFiles() {
		source := filepath.Join(recovery, filepath.Base(destination))
		if !fileExists(source) {
			continue
		}
		if err := os.Rename(source, destination); err != nil {
			return fmt.Errorf("restore %q: %w", filepath.Base(destination), err)
		}
	}
	if err := os.Remove(recovery); err != nil {
		manager.logger.Warn("empty recovery folder could not be removed", "operation", "vault-legacy", "error", err)
	}
	manager.logger.Info("legacy vault restored from the recovery folder", "operation", "vault-legacy")
	return nil
}
