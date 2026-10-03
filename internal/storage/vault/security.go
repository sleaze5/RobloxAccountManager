package vault

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
)

func (manager *Manager) ChangePassword(ctx context.Context, currentPassword, newPassword, confirmation, hint string, automaticUnlock bool) error {
	if newPassword != confirmation {
		return fmt.Errorf("new passwords do not match")
	}
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	if currentPassword != "" {
		if err := manager.TestPassword(ctx, currentPassword); err != nil {
			return err
		}
	} else if !manager.ValidAutomaticUnlock() {
		return fmt.Errorf("the current password or Windows automatic unlock is required")
	}
	return manager.rotateDatabaseKey(ctx, newPassword, hint, automaticUnlock)
}

func (manager *Manager) SetAutomaticUnlock(enabled bool) error {
	manager.mu.Lock()
	if manager.db == nil || manager.closing {
		manager.mu.Unlock()
		return ErrLocked
	}
	identifier := manager.vaultID
	dek := append([]byte(nil), manager.dek...)
	manager.mu.Unlock()
	defer wipe(dek)
	if !enabled {
		if err := os.Remove(manager.paths.AutoUnlockKey); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove automatic unlock file: %w", err)
		}
		return nil
	}
	data, err := encodeAutoUnlock(manager.protector, identifier, dek)
	if err != nil {
		return err
	}
	return appdata.WritePrivateFile(manager.paths.AutoUnlockKey, data)
}

func (manager *Manager) rotateDatabaseKey(ctx context.Context, password, hint string, automaticUnlock bool) error {
	params, err := calibrateKDF(password)
	if err != nil {
		return err
	}
	manager.mu.Lock()
	if manager.db == nil || manager.closing {
		manager.mu.Unlock()
		return ErrLocked
	}
	identifier := manager.vaultID
	manager.mu.Unlock()
	newDEK, err := newDEK()
	if err != nil {
		return err
	}
	defer wipe(newDEK)
	key, err := createKeyFile(password, hint, identifier, newDEK, params)
	if err != nil {
		return err
	}
	keyData, err := encodeKeyFile(key)
	if err != nil {
		return err
	}
	var autoData []byte
	if automaticUnlock {
		autoData, err = encodeAutoUnlock(manager.protector, identifier, newDEK)
		if err != nil {
			return err
		}
	}
	if err := manager.PurgeBackups(); err != nil {
		return err
	}
	if err := appdata.WritePrivateFile(manager.paths.VaultKey+".next", keyData); err != nil {
		return err
	}
	if automaticUnlock {
		if err := appdata.WritePrivateFile(manager.paths.AutoUnlockKey+".next", autoData); err != nil {
			return err
		}
	} else {
		if err := os.Remove(manager.paths.AutoUnlockKey + ".next"); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove stale automatic unlock candidate: %w", err)
		}
	}
	db, finish, err := manager.beginExclusive(ctx)
	if err != nil {
		return err
	}
	finished := false
	defer func() {
		if !finished {
			finish()
		}
	}()
	if _, err := db.ExecContext(ctx, `PRAGMA rekey = "x'`+hex.EncodeToString(newDEK)+`'"`); err != nil {
		return fmt.Errorf("rekey vault database: %w", err)
	}
	if err := verifyOpenDatabase(ctx, db, identifier, manager.logger); err != nil {
		return fmt.Errorf("verify rekeyed vault database: %w", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE vault_state SET storage_revision = storage_revision + 1 WHERE singleton = 1`); err != nil {
		return fmt.Errorf("record new master password: %w", err)
	}
	manager.mu.Lock()
	oldDEK := manager.dek
	manager.dek = append([]byte(nil), newDEK...)
	manager.mu.Unlock()
	wipe(oldDEK)
	if automaticUnlock {
		if err := appdata.ReplaceFile(manager.paths.AutoUnlockKey+".next", manager.paths.AutoUnlockKey); err != nil {
			return err
		}
	} else if err := os.Remove(manager.paths.AutoUnlockKey); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove automatic unlock file: %w", err)
	}
	if err := appdata.ReplaceFile(manager.paths.VaultKey+".next", manager.paths.VaultKey); err != nil {
		return err
	}
	manager.mu.Lock()
	manager.unlockedWithDPAPI = automaticUnlock
	manager.mu.Unlock()
	finish()
	finished = true
	if err := manager.CreateBackup(ctx); err != nil {
		return fmt.Errorf("password changed, but the fresh backup failed: %w", err)
	}
	return nil
}

func (manager *Manager) beginExclusive(ctx context.Context) (*sql.DB, func(), error) {
	manager.mu.Lock()
	if manager.db == nil || manager.closing {
		manager.mu.Unlock()
		return nil, nil, ErrLocked
	}
	manager.closing = true
	var drained <-chan struct{}
	if manager.active > 0 {
		manager.drained = make(chan struct{})
		drained = manager.drained
	}
	db := manager.db
	manager.mu.Unlock()
	if drained != nil {
		timer := time.NewTimer(lockTimeout)
		defer timer.Stop()
		select {
		case <-drained:
		case <-ctx.Done():
			manager.endExclusive()
			return nil, nil, ctx.Err()
		case <-timer.C:
			manager.endExclusive()
			return nil, nil, ErrLockTimeout
		}
	}
	return db, manager.endExclusive, nil
}

func (manager *Manager) endExclusive() {
	manager.mu.Lock()
	manager.closing = false
	manager.drained = nil
	manager.mu.Unlock()
}

func (manager *Manager) promoteOrDiscardRotation(useNext bool) error {
	if useNext {
		if fileExists(manager.paths.AutoUnlockKey + ".next") {
			if err := appdata.ReplaceFile(manager.paths.AutoUnlockKey+".next", manager.paths.AutoUnlockKey); err != nil {
				return err
			}
		} else if data, err := readBounded(manager.paths.AutoUnlockKey, autoFileLimit); err == nil {
			identifier, dek, decodeErr := decodeAutoUnlock(data, manager.protector)
			if decodeErr == nil {
				manager.mu.Lock()
				matches := identifier == manager.vaultID && subtle.ConstantTimeCompare(dek, manager.dek) == 1
				manager.mu.Unlock()
				wipe(dek)
				if !matches {
					if err := os.Remove(manager.paths.AutoUnlockKey); err != nil && !errors.Is(err, os.ErrNotExist) {
						return err
					}
				}
			}
		}
		if fileExists(manager.paths.VaultKey + ".next") {
			return appdata.ReplaceFile(manager.paths.VaultKey+".next", manager.paths.VaultKey)
		}
		return nil
	}
	for _, path := range []string{manager.paths.VaultKey + ".next", manager.paths.AutoUnlockKey + ".next"} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}
