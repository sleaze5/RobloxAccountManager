//go:build linux

package protection

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/godbus/dbus/v5"
	"github.com/sleaze5/RobloxAccountManager/internal/appmeta"
	"github.com/zalando/go-keyring"
)

const (
	autoUnlockKeyAccount = "autounlock/v1"
	protectedVersion     = 1
	secretsService       = "org.freedesktop.secrets"
	secretsAlias         = "/org/freedesktop/secrets/aliases/default"
	collectionInterface  = "org.freedesktop.Secret.Collection"
)

type linuxProtector struct{}

// New returns the Linux data protector.
func New() Protector {
	return linuxProtector{}
}

func (linuxProtector) Protect(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, fmt.Errorf("protect data: plaintext is empty")
	}
	key, err := encryptionKey()
	if err != nil {
		return nil, fmt.Errorf("protect data with the Secret Service: %w", err)
	}
	defer wipe(key)
	sealed, nonce, err := seal(key, plaintext)
	if err != nil {
		return nil, fmt.Errorf("protect data with the Secret Service: %w", err)
	}
	out := make([]byte, 1+len(nonce)+len(sealed))
	out[0] = protectedVersion
	copy(out[1:], nonce)
	copy(out[1+len(nonce):], sealed)
	return out, nil
}

func (linuxProtector) Unprotect(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("unprotect data: ciphertext is empty")
	}
	if ciphertext[0] != protectedVersion {
		return nil, fmt.Errorf("unprotect data with the Secret Service: protected data version %d is unsupported", ciphertext[0])
	}
	key, err := encryptionKey()
	if err != nil {
		return nil, fmt.Errorf("unprotect data with the Secret Service: %w", err)
	}
	defer wipe(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("unprotect data with the Secret Service: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("unprotect data with the Secret Service: %w", err)
	}
	if len(ciphertext) < 1+gcm.NonceSize()+gcm.Overhead() {
		return nil, fmt.Errorf("unprotect data with the Secret Service: protected data is truncated")
	}
	nonce := ciphertext[1 : 1+gcm.NonceSize()]
	sealed := ciphertext[1+gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, sealed, []byte(appmeta.Identifier+"/autounlock/v1"))
	if err != nil {
		return nil, fmt.Errorf("unprotect data with the Secret Service: %w", err)
	}
	return plain, nil
}

func seal(key, plaintext []byte) (sealed, nonce []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	sealed = gcm.Seal(nil, nonce, plaintext, []byte(appmeta.Identifier+"/autounlock/v1"))
	return sealed, nonce, nil
}

func encryptionKey() ([]byte, error) {
	if err := rejectLockedKeyring(); err != nil {
		return nil, err
	}
	encoded, err := keyring.Get(appmeta.Identifier, autoUnlockKeyAccount)
	if errors.Is(err, keyring.ErrNotFound) {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
		if err := keyring.Set(appmeta.Identifier, autoUnlockKeyAccount, base64.StdEncoding.EncodeToString(key)); err != nil {
			wipe(key)
			return nil, err
		}
		return key, nil
	}
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

func rejectLockedKeyring() error {
	connection, err := dbus.SessionBus()
	if err != nil {
		return fmt.Errorf("connect to the session bus: %w", err)
	}
	value, err := connection.Object(secretsService, dbus.ObjectPath(secretsAlias)).GetProperty(collectionInterface + ".Locked")
	if err != nil {
		return fmt.Errorf("read the Secret Service collection: %w", err)
	}
	locked, ok := value.Value().(bool)
	if !ok {
		return fmt.Errorf("read the Secret Service collection lock state")
	}
	if locked {
		return fmt.Errorf("the Secret Service collection is locked")
	}
	return nil
}

func wipe(data []byte) {
	for i := range data {
		data[i] = 0
	}
}
