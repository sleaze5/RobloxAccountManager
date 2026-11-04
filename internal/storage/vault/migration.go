package vault

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

// A migration file carries the vault data from 0.9.2 to 0.10.0, which cannot
// read the old vault format. Its key comes from the old vault key.
const (
	migrationFormatVersion = uint16(1)
	// An 8-byte magic, a 2-byte version, a 16-byte vault ID, and a 12-byte nonce.
	migrationHeaderBytes = 8 + 2 + vaultIDBytes + 12
)

var (
	ErrNoMigration      = errors.New("no migration file was found")
	ErrInvalidMigration = errors.New("the migration file is damaged")
	migrationMagic      = [8]byte{'R', 'A', 'M', 'M', 'I', 'G', 'R', 0}
)

// MigrateLegacyVault moves a vault from a version before 0.10.0 into the new
// format with the migration file that 0.9.2 wrote before the update. It sets
// the old vault aside, creates a new vault with the same master password, hint,
// and automatic unlock choice. accept checks the migrated data before any file
// changes, and the caller imports it into the new vault afterward.
func (manager *Manager) MigrateLegacyVault(ctx context.Context, password string, accept func([]byte) error) error {
	if manager.FileState() != FileStateLegacy {
		return errors.New("the vault is not from a version before 0.10.0")
	}
	legacy, err := readLegacyKeyFile(manager.paths.VaultKey)
	if err != nil {
		return err
	}
	dek, err := legacy.unwrap(password)
	if err != nil {
		return err
	}
	defer wipe(dek)
	payload, err := manager.readMigration(dek, legacy.VaultID)
	if err != nil {
		return err
	}
	err = accept(payload)
	clear(payload)
	if err != nil {
		return err
	}
	automaticUnlock := fileExists(manager.paths.AutoUnlockKey)
	recovery, err := manager.setAsideLegacyVault()
	if err != nil {
		return err
	}
	if err := manager.Create(ctx, password, legacy.Hint, automaticUnlock); err != nil {
		if restoreErr := manager.restoreLegacyVault(recovery); restoreErr != nil {
			return errors.Join(err, restoreErr)
		}
		return err
	}
	manager.logger.Info("legacy vault migrated", "operation", "vault-migration", "automatic_unlock", automaticUnlock)
	return nil
}

func (manager *Manager) MigrationAvailable() bool { return fileExists(manager.paths.Migration) }

func (manager *Manager) readMigration(dek []byte, identifier [vaultIDBytes]byte) ([]byte, error) {
	data, err := readBounded(manager.paths.Migration, exportFileLimit)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNoMigration
	}
	if err != nil {
		return nil, err
	}
	if len(data) < migrationHeaderBytes || !bytes.Equal(data[:len(migrationMagic)], migrationMagic[:]) {
		return nil, ErrInvalidMigration
	}
	if version := binary.BigEndian.Uint16(data[len(migrationMagic):]); version != migrationFormatVersion {
		return nil, fmt.Errorf("%w: migration format version %d", ErrInvalidMigration, version)
	}
	if !bytes.Equal(data[len(migrationMagic)+2:len(migrationMagic)+2+vaultIDBytes], identifier[:]) {
		return nil, fmt.Errorf("%w: the migration file belongs to another vault", ErrInvalidMigration)
	}
	key, err := migrationKey(dek)
	if err != nil {
		return nil, err
	}
	defer wipe(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create migration cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create migration cipher mode: %w", err)
	}
	header := data[:migrationHeaderBytes]
	payload, err := gcm.Open(nil, header[len(migrationMagic)+2+vaultIDBytes:], data[migrationHeaderBytes:], header)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidMigration, err)
	}
	return payload, nil
}

func (manager *Manager) RemoveMigration() error {
	if err := os.Remove(manager.paths.Migration); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove vault migration file: %w", err)
	}
	return nil
}

func migrationKey(dek []byte) ([]byte, error) {
	key, err := hkdf.Key(sha256.New, dek, nil, "RobloxAccountManager vault migration", 32)
	if err != nil {
		return nil, fmt.Errorf("derive vault migration key: %w", err)
	}
	return key, nil
}
