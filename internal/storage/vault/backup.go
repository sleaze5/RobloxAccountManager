package vault

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	sqlite3 "github.com/mutecomm/go-sqlcipher/v4"
	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
)

const backupTimeFormat = "20060102T150405.000000000Z"

type BackupInfo struct {
	Name        string `json:"name"`
	CreatedAtMS int64  `json:"createdAtMs"`
}

func (manager *Manager) MaybeCreateBackup(ctx context.Context) error {
	db, release, err := manager.Lease(ctx)
	if err != nil {
		return err
	}
	var revision int64
	if err := db.QueryRowContext(ctx, `SELECT storage_revision FROM vault_state WHERE singleton = 1`).Scan(&revision); err != nil {
		release()
		return err
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
	manager.mu.Lock()
	dek := append([]byte(nil), manager.dek...)
	identifier := manager.vaultID
	manager.mu.Unlock()
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
	if err := snapshotDatabase(ctx, db, destination, dek, manager.logger); err != nil {
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

func snapshotDatabase(ctx context.Context, source *sql.DB, destinationPath string, dek []byte, logger *slog.Logger) error {
	destination, err := openDatabase(ctx, destinationPath, dek, logger)
	if err != nil {
		return err
	}
	defer destination.Close()
	sourceConn, err := source.Conn(ctx)
	if err != nil {
		return fmt.Errorf("open backup source: %w", err)
	}
	defer sourceConn.Close()
	destinationConn, err := destination.Conn(ctx)
	if err != nil {
		return fmt.Errorf("open backup destination: %w", err)
	}
	defer destinationConn.Close()
	return sourceConn.Raw(func(sourceDriver any) error {
		sourceSQLite, ok := sourceDriver.(*sqlite3.SQLiteConn)
		if !ok {
			return fmt.Errorf("SQLCipher backup source is unavailable")
		}
		return destinationConn.Raw(func(destinationDriver any) error {
			destinationSQLite, ok := destinationDriver.(*sqlite3.SQLiteConn)
			if !ok {
				return fmt.Errorf("SQLCipher backup destination is unavailable")
			}
			backup, err := destinationSQLite.Backup("main", sourceSQLite, "main")
			if err != nil {
				return fmt.Errorf("start SQLite backup: %w", err)
			}
			for {
				// Copy 128 pages per step.
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
	})
}

func (manager *Manager) ListBackups() ([]BackupInfo, error) {
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
		result = append(result, BackupInfo{Name: entry.Name(), CreatedAtMS: created.UnixMilli()})
	}
	sort.Slice(result, func(left, right int) bool { return result[left].Name > result[right].Name })
	return result, nil
}

func (manager *Manager) trimBackups() error {
	backups, err := manager.ListBackups()
	if err != nil {
		return err
	}
	// Keep the 3 newest backups.
	if len(backups) <= 3 {
		return nil
	}
	for _, backup := range backups[3:] {
		if err := manager.removeManaged(filepath.Join(manager.paths.BackupsRoot, backup.Name), manager.paths.BackupsRoot); err != nil {
			return err
		}
	}
	return nil
}

func (manager *Manager) PurgeBackups() error {
	entries, err := os.ReadDir(manager.paths.BackupsRoot)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
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
		return fmt.Errorf("refusing to remove an unmanaged path")
	}
	return os.RemoveAll(path)
}

func (manager *Manager) backupRevision(ctx context.Context, name string) (int64, error) {
	path, err := manager.backupPath(name)
	if err != nil {
		return 0, err
	}
	manager.mu.Lock()
	dek := append([]byte(nil), manager.dek...)
	identifier := manager.vaultID
	manager.mu.Unlock()
	defer wipe(dek)
	db, err := openAndVerify(ctx, filepath.Join(path, "vault.db"), dek, identifier, manager.logger)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	var revision int64
	err = db.QueryRowContext(ctx, `SELECT storage_revision FROM vault_state WHERE singleton = 1`).Scan(&revision)
	return revision, err
}

func (manager *Manager) RestoreBackup(ctx context.Context, name, password string) error {
	if manager.Unlocked() {
		return fmt.Errorf("lock the vault before restoring a backup")
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
		return err
	}
	if err := os.Remove(manager.paths.AutoUnlockKey); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove automatic unlock file: %w", err)
	}
	if err := appdata.PreparePrivateDirectory(manager.paths.RecoveryRoot); err != nil {
		return err
	}
	recovery := filepath.Join(manager.paths.RecoveryRoot, time.Now().UTC().Format(backupTimeFormat))
	if err := appdata.PreparePrivateDirectory(recovery); err != nil {
		return err
	}
	for _, source := range []string{manager.paths.Vault, manager.paths.VaultKey} {
		if !fileExists(source) {
			continue
		}
		if err := os.Rename(source, filepath.Join(recovery, filepath.Base(source))); err != nil {
			return fmt.Errorf("quarantine current %q: %w", filepath.Base(source), err)
		}
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
		return "", fmt.Errorf("invalid backup selection")
	}
	path := filepath.Join(manager.paths.BackupsRoot, name)
	relative, err := filepath.Rel(manager.paths.BackupsRoot, path)
	if err != nil || relative != name {
		return "", fmt.Errorf("invalid backup selection")
	}
	return path, nil
}

func copyPrivateFile(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.CreateTemp(filepath.Dir(destination), "."+filepath.Base(destination)+".restore-*")
	if err != nil {
		return err
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
		return err
	}
	if err := output.Sync(); err != nil {
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	if err := appdata.ReplaceFile(temporary, destination); err != nil {
		return err
	}
	ok = true
	return appdata.RestrictFile(destination)
}
