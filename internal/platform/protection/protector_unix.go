//go:build linux || darwin

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

type keyringProtector struct{}

func New() Protector {
	return keyringProtector{}
}

func (keyringProtector) Protect(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, errors.New("protect data: plaintext is empty")
	}
	key, err := encryptionKey()
	if err != nil {
		return nil, fmt.Errorf("protect data with %s: %w", keyringName, err)
	}
	defer clear(key)
	gcm, err := newGCM(key)
	if err != nil {
		return nil, fmt.Errorf("protect data with %s: %w", keyringName, err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("protect data with %s: %w", keyringName, err)
	}
	header := append([]byte{protectedVersion}, nonce...)
	return gcm.Seal(header, nonce, plaintext, []byte(purpose)), nil
}

func (keyringProtector) Unprotect(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, errors.New("unprotect data: ciphertext is empty")
	}
	if ciphertext[0] != protectedVersion {
		return nil, fmt.Errorf("unprotect data with %s: protected data version %d is unsupported", keyringName, ciphertext[0])
	}
	key, err := existingKey()
	if err != nil {
		return nil, fmt.Errorf("unprotect data with %s: %w", keyringName, err)
	}
	defer clear(key)
	gcm, err := newGCM(key)
	if err != nil {
		return nil, fmt.Errorf("unprotect data with %s: %w", keyringName, err)
	}
	if len(ciphertext) < 1+gcm.NonceSize()+gcm.Overhead() {
		return nil, fmt.Errorf("unprotect data with %s: protected data is truncated", keyringName)
	}
	nonce := ciphertext[1 : 1+gcm.NonceSize()]
	plain, err := gcm.Open(nil, nonce, ciphertext[1+gcm.NonceSize():], []byte(purpose))
	if err != nil {
		return nil, fmt.Errorf("unprotect data with %s: %w", keyringName, err)
	}
	return plain, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create automatic unlock cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create automatic unlock cipher mode: %w", err)
	}
	return gcm, nil
}

func encryptionKey() ([]byte, error) {
	key, err := existingKey()
	if !errors.Is(err, keyring.ErrNotFound) {
		return key, err
	}
	key = make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate automatic unlock key: %w", err)
	}
	if err := keyring.Set(appmeta.Identifier, autoUnlockKeyAccount, base64.StdEncoding.EncodeToString(key)); err != nil {
		clear(key)
		return nil, fmt.Errorf("store automatic unlock key in %s: %w", keyringName, err)
	}
	return key, nil
}

func existingKey() ([]byte, error) {
	if err := checkKeyring(); err != nil {
		return nil, err
	}
	encoded, err := keyring.Get(appmeta.Identifier, autoUnlockKeyAccount)
	if err != nil {
		return nil, fmt.Errorf("read automatic unlock key from %s: %w", keyringName, err)
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(key) != 32 {
		clear(key)
		return nil, errors.New("automatic unlock key is invalid")
	}
	return key, nil
}
