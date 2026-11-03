package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"os"

	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
)

// A migration file carries the vault data to the first launch of a version
// that cannot read this vault format. Its key comes from the vault key, so the
// master password or automatic unlock of this vault opens it.
const (
	migrationFormatVersion = uint16(1)
	// An 8-byte magic, a 2-byte version, a 16-byte vault ID, and a 12-byte nonce.
	migrationHeaderBytes = 8 + 2 + vaultIDBytes + 12
)

var migrationMagic = [8]byte{'R', 'A', 'M', 'M', 'I', 'G', 'R', 0}

func (manager *Manager) WriteMigration(payload []byte) error {
	dek, identifier := manager.currentKey()
	defer wipe(dek)
	if !manager.Unlocked() || len(dek) != dekBytes {
		return ErrLocked
	}
	key, err := migrationKey(dek)
	if err != nil {
		return err
	}
	defer wipe(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("create migration cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("create migration cipher mode: %w", err)
	}
	header := make([]byte, migrationHeaderBytes)
	copy(header, migrationMagic[:])
	binary.BigEndian.PutUint16(header[len(migrationMagic):], migrationFormatVersion)
	copy(header[len(migrationMagic)+2:], identifier[:])
	if _, err := rand.Read(header[len(migrationMagic)+2+vaultIDBytes:]); err != nil {
		return fmt.Errorf("generate migration nonce: %w", err)
	}
	sealed := gcm.Seal(header, header[len(migrationMagic)+2+vaultIDBytes:], payload, header)
	if err := appdata.WritePrivateFile(manager.paths.Migration, sealed); err != nil {
		return err
	}
	manager.logger.Info("vault migration file written", "operation", "vault-migration", "migration_format_version", migrationFormatVersion)
	return nil
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
