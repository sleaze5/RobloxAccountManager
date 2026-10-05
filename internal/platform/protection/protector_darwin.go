//go:build darwin

package protection

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/sleaze5/RobloxAccountManager/internal/appmeta"
	"github.com/zalando/go-keyring"
)

const (
	autoUnlockKeyAccount = "autounlock/v1"
	protectedVersion     = 1
)

// darwinProtector seals data with a random key that it keeps in the login
// Keychain, so a copied autounlock.key cannot be opened on another Mac or by
// another user.
type darwinProtector struct{}

func New() Protector {
	return darwinProtector{}
}

func (darwinProtector) Protect(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, fmt.Errorf("protect data: plaintext is empty")
	}
	key, err := encryptionKey()
	if err != nil {
		return nil, fmt.Errorf("protect data with the Keychain: %w", err)
	}
	defer wipe(key)
	gcm, err := newGCM(key)
	if err != nil {
		return nil, fmt.Errorf("protect data with the Keychain: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("protect data with the Keychain: %w", err)
	}
	out := append([]byte{protectedVersion}, nonce...)
	return gcm.Seal(out, nonce, plaintext, []byte(appmeta.Identifier+"/autounlock/v1")), nil
}

func (darwinProtector) Unprotect(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("unprotect data: ciphertext is empty")
	}
	if ciphertext[0] != protectedVersion {
		return nil, fmt.Errorf("unprotect data with the Keychain: protected data version %d is unsupported", ciphertext[0])
	}
	key, err := existingKey()
	if err != nil {
		return nil, fmt.Errorf("unprotect data with the Keychain: %w", err)
	}
	defer wipe(key)
	gcm, err := newGCM(key)
	if err != nil {
		return nil, fmt.Errorf("unprotect data with the Keychain: %w", err)
	}
	if len(ciphertext) < 1+gcm.NonceSize()+gcm.Overhead() {
		return nil, fmt.Errorf("unprotect data with the Keychain: protected data is truncated")
	}
	nonce := ciphertext[1 : 1+gcm.NonceSize()]
	plain, err := gcm.Open(nil, nonce, ciphertext[1+gcm.NonceSize():], []byte(appmeta.Identifier+"/autounlock/v1"))
	if err != nil {
		return nil, fmt.Errorf("unprotect data with the Keychain: %w", err)
	}
	return plain, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// encryptionKey returns the Keychain key and creates it on first use.
func encryptionKey() ([]byte, error) {
	key, err := existingKey()
	if !errors.Is(err, keyring.ErrNotFound) {
		return key, err
	}
	key = make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := keyring.Set(appmeta.Identifier, autoUnlockKeyAccount, base64.StdEncoding.EncodeToString(key)); err != nil {
		wipe(key)
		return nil, err
	}
	return key, nil
}

// existingKey never creates a key, so a missing Keychain item makes an
// existing autounlock.key fail instead of being replaced.
func existingKey() ([]byte, error) {
	encoded, err := keyring.Get(appmeta.Identifier, autoUnlockKeyAccount)
	if err != nil {
		return nil, err
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(key) != 32 {
		wipe(key)
		return nil, fmt.Errorf("automatic unlock key is invalid")
	}
	return key, nil
}

func wipe(data []byte) {
	for i := range data {
		data[i] = 0
	}
}
