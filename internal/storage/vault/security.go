package vault

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
)

func (manager *Manager) ChangePassword(ctx context.Context, currentPassword, newPassword, confirmation, hint string, automaticUnlock bool) error {
	if newPassword != confirmation {
		return errors.New("new passwords do not match")
	}
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	if currentPassword != "" {
		if err := manager.TestPassword(ctx, currentPassword); err != nil {
			return err
		}
	} else if !manager.ValidAutomaticUnlock() {
		return errors.New("the current password or automatic unlock is required")
	}
	return manager.rotateDatabaseKey(ctx, newPassword, hint, automaticUnlock)
}

func (manager *Manager) SetAutomaticUnlock(enabled bool) error {
	manager.mutex.Lock()
	if manager.database == nil || manager.closing {
		manager.mutex.Unlock()
		return ErrLocked
	}
	identifier := manager.vaultID
	vaultKey := append([]byte(nil), manager.vaultKey...)
	manager.mutex.Unlock()
	defer wipe(vaultKey)
	if !enabled {
		if err := os.Remove(manager.paths.AutoUnlockKey); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove automatic unlock file: %w", err)
		}
		return nil
	}
	encodedAutoUnlock, err := encodeAutoUnlock(manager.protector, identifier, vaultKey)
	if err != nil {
		return err
	}
	return appdata.WritePrivateFile(manager.paths.AutoUnlockKey, encodedAutoUnlock)
}

func (manager *Manager) rotateDatabaseKey(ctx context.Context, password, hint string, automaticUnlock bool) error {
	params := calibrateKDF(password)
	manager.mutex.Lock()
	if manager.database == nil || manager.closing {
		manager.mutex.Unlock()
		return ErrLocked
	}
	identifier := manager.vaultID
	manager.mutex.Unlock()
	newVaultKey, err := newVaultKey()
	if err != nil {
		return err
	}
	defer wipe(newVaultKey)
	key, err := createKeyFile(password, hint, identifier, newVaultKey, params)
	if err != nil {
		return err
	}
	keyData, err := encodeKeyFile(key)
	if err != nil {
		return err
	}
	var autoData []byte
	if automaticUnlock {
		autoData, err = encodeAutoUnlock(manager.protector, identifier, newVaultKey)
		if err != nil {
			return err
		}
	}
	if err := manager.stageRotatedKeys(keyData, autoData, automaticUnlock); err != nil {
		return err
	}
	database, finish, err := manager.beginExclusive(ctx)
	if err != nil {
		return err
	}
	finished := false
	defer func() {
		if !finished {
			finish()
		}
	}()
	if err := manager.replaceDatabaseKey(ctx, database, newVaultKey, identifier); err != nil {
		return err
	}
	if err := manager.publishRotatedKeys(automaticUnlock); err != nil {
		return err
	}
	finish()
	finished = true
	if err := manager.CreateBackup(ctx); err != nil {
		return fmt.Errorf("password changed, but the fresh backup failed: %w", err)
	}
	return nil
}

func (manager *Manager) beginExclusive(ctx context.Context) (*sql.DB, func(), error) {
	manager.mutex.Lock()
	if manager.database == nil {
		manager.mutex.Unlock()
		return nil, nil, ErrLocked
	}
	if manager.closing {
		manager.mutex.Unlock()
		return nil, nil, errBusy
	}
	manager.closing = true
	var drained <-chan struct{}
	if manager.active > 0 {
		manager.drained = make(chan struct{})
		drained = manager.drained
	}
	database := manager.database
	manager.mutex.Unlock()
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
			return nil, nil, errLockTimeout
		}
	}
	return database, manager.endExclusive, nil
}

func (manager *Manager) endExclusive() {
	manager.mutex.Lock()
	manager.closing = false
	manager.drained = nil
	manager.mutex.Unlock()
}

func (manager *Manager) promoteOrDiscardRotation(useNext bool) error {
	if useNext {
		if fileExists(manager.paths.AutoUnlockKey + ".next") {
			if err := appdata.ReplaceFile(manager.paths.AutoUnlockKey+".next", manager.paths.AutoUnlockKey); err != nil {
				return err
			}
		} else if encodedAutoUnlock, err := readBounded(manager.paths.AutoUnlockKey, autoFileLimit); err == nil {
			identifier, vaultKey, decodeError := decodeAutoUnlock(encodedAutoUnlock, manager.protector)
			if decodeError == nil {
				manager.mutex.Lock()
				matches := identifier == manager.vaultID && subtle.ConstantTimeCompare(vaultKey, manager.vaultKey) == 1
				manager.mutex.Unlock()
				wipe(vaultKey)
				if !matches {
					if err := os.Remove(manager.paths.AutoUnlockKey); err != nil && !errors.Is(err, os.ErrNotExist) {
						return fmt.Errorf("remove automatic unlock file: %w", err)
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
			return fmt.Errorf("remove rotated key file: %w", err)
		}
	}
	return nil
}

// stageRotatedKeys removes the current automatic unlock file before it stages a
// rotation without one. Otherwise, after a crash before the database swap, that
// file would unlock the old database and the unlock would promote vault.key.next.
func (manager *Manager) stageRotatedKeys(keyData, autoData []byte, automaticUnlock bool) error {
	if err := manager.PurgeBackups(); err != nil {
		return err
	}
	if !automaticUnlock {
		for _, path := range []string{manager.paths.AutoUnlockKey + ".next", manager.paths.AutoUnlockKey} {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("remove automatic unlock file: %w", err)
			}
		}
	}
	if err := appdata.WritePrivateFile(manager.paths.VaultKey+".next", keyData); err != nil {
		return err
	}
	if automaticUnlock {
		return appdata.WritePrivateFile(manager.paths.AutoUnlockKey+".next", autoData)
	}
	return nil
}

// replaceDatabaseKey copies the vault into a database encrypted with newDEK and
// swaps it into place. Until the swap, the current vault.key opens vault.db.
// After it, vault.key.next does, so an interrupted rotation recovers on unlock.
func (manager *Manager) replaceDatabaseKey(ctx context.Context, database *sql.DB, newVaultKey []byte, identifier [vaultIDBytes]byte) error {
	next := manager.paths.Vault + ".next"
	if err := os.Remove(next); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove stale rotated vault: %w", err)
	}
	if err := manager.stageRotatedDatabase(ctx, database, next, newVaultKey, identifier); err != nil {
		_ = os.Remove(next)
		return err
	}
	if err := database.Close(); err != nil {
		_ = os.Remove(next)
		return manager.reopenAfterFailedRotation(ctx, fmt.Errorf("close vault before rotation: %w", err))
	}
	if err := appdata.ReplaceFile(next, manager.paths.Vault); err != nil {
		_ = os.Remove(next)
		return manager.reopenAfterFailedRotation(ctx, err)
	}
	rotated, err := openAndVerify(ctx, manager.paths.Vault, newVaultKey, identifier, manager.logger)
	if err != nil {
		manager.dropDatabase()
		return fmt.Errorf("open rotated vault database: %w", err)
	}
	manager.mutex.Lock()
	oldVaultKey := manager.vaultKey
	manager.database = rotated
	manager.vaultKey = append([]byte(nil), newVaultKey...)
	manager.mutex.Unlock()
	wipe(oldVaultKey)
	return nil
}

func (manager *Manager) stageRotatedDatabase(ctx context.Context, database *sql.DB, path string, newVaultKey []byte, identifier [vaultIDBytes]byte) error {
	if err := snapshotDatabase(ctx, database, path, newVaultKey); err != nil {
		return err
	}
	staged, err := openAndVerify(ctx, path, newVaultKey, identifier, manager.logger)
	if err != nil {
		return fmt.Errorf("verify rotated vault database: %w", err)
	}
	if err := bumpRevision(ctx, staged); err != nil {
		_ = staged.Close()
		return fmt.Errorf("record new master password: %w", err)
	}
	if err := staged.Close(); err != nil {
		return fmt.Errorf("close rotated vault database: %w", err)
	}
	return appdata.RestrictFile(path)
}

// reopenAfterFailedRotation reopens vault.db with the current key after the
// rotation closed it. If that fails too, the vault locks.
func (manager *Manager) reopenAfterFailedRotation(ctx context.Context, cause error) error {
	vaultKey, identifier := manager.currentKey()
	defer wipe(vaultKey)
	database, err := openAndVerify(ctx, manager.paths.Vault, vaultKey, identifier, manager.logger)
	if err != nil {
		manager.dropDatabase()
		return errors.Join(cause, fmt.Errorf("reopen vault after failed rotation: %w", err))
	}
	manager.mutex.Lock()
	manager.database = database
	manager.mutex.Unlock()
	return cause
}

func (manager *Manager) dropDatabase() {
	manager.mutex.Lock()
	vaultKey := manager.vaultKey
	manager.database = nil
	manager.vaultKey = nil
	manager.vaultID = [vaultIDBytes]byte{}
	manager.unlockedAutomatically = false
	manager.mutex.Unlock()
	wipe(vaultKey)
}

func (manager *Manager) publishRotatedKeys(automaticUnlock bool) error {
	if automaticUnlock {
		if err := appdata.ReplaceFile(manager.paths.AutoUnlockKey+".next", manager.paths.AutoUnlockKey); err != nil {
			return err
		}
	}
	if err := appdata.ReplaceFile(manager.paths.VaultKey+".next", manager.paths.VaultKey); err != nil {
		return err
	}
	manager.mutex.Lock()
	manager.unlockedAutomatically = automaticUnlock
	manager.mutex.Unlock()
	return nil
}
