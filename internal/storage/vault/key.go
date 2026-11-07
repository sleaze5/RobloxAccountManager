package vault

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/platform/protection"
	"golang.org/x/crypto/argon2"
)

const (
	keyFormatVersion  = uint16(2)
	autoFormatVersion = uint16(1)
	dekBytes          = 32
	vaultIDBytes      = 16
	keyFileLimit      = 1024
	autoFileLimit     = 16 * 1024
	maxHintRunes      = 120
	maxHintBytes      = 480
	minArgonMemoryKiB = 64 * 1024
	maxArgonMemoryKiB = 512 * 1024
	maxArgonTime      = 10
	maxArgonThreads   = 16
	argonKeyBytes     = 32
	saltBytes         = 16
	nonceBytes        = 12
	wrappedDEKBytes   = dekBytes + 16
	autoHeaderBytes   = 14
)

var (
	keyMagic       = [8]byte{'R', 'A', 'M', 'V', 'K', 'E', 'Y', 0}
	autoMagic      = [8]byte{'R', 'A', 'M', 'A', 'U', 'T', 'O', 0}
	autoPlainMagic = [8]byte{'R', 'A', 'M', 'A', 'P', 'L', 'N', 0}
)

// Version 1 key files belong to SQLCipher vaults, which this build cannot open.
// Version 2 keeps the same layout for vaults encrypted with AES-256-XTS.
var keyFileDecoders = [...]func([]byte) (keyFile, error){decodeLegacyKeyFile, decodeKeyFileFields}

var autoUnlockDecoders = [...]func([]byte) ([vaultIDBytes]byte, []byte, error){decodeAutoUnlockV1}

var (
	_ = [1]struct{}{}[len(keyFileDecoders)-int(keyFormatVersion)]
	_ = [1]struct{}{}[len(autoUnlockDecoders)-int(autoFormatVersion)]
)

type argon2Parameters struct {
	Time      uint32
	MemoryKiB uint32
	Threads   uint8
}

type keyFile struct {
	Version    uint16
	VaultID    [vaultIDBytes]byte
	KDF        argon2Parameters
	Salt       []byte
	Nonce      []byte
	WrappedDEK []byte
	Hint       string
}

func meetsPasswordRules(password string) bool {
	runeCount := utf8.RuneCountInString(password)
	return utf8.ValidString(password) && runeCount >= 8 && runeCount <= 128 && len(password) <= 1024 && hasEveryCharacterClass(password)
}

func hasEveryCharacterClass(password string) bool {
	var uppercase, lowercase, digit, special bool
	for _, character := range password {
		switch {
		case character >= 'A' && character <= 'Z':
			uppercase = true
		case character >= 'a' && character <= 'z':
			lowercase = true
		case character >= '0' && character <= '9':
			digit = true
		default:
			special = true
		}
	}
	return uppercase && lowercase && digit && special
}

func validatePassword(password string) error {
	if !meetsPasswordRules(password) {
		return errors.New("master password does not meet every requirement")
	}
	return nil
}

func validateHint(hint string) error {
	if !utf8.ValidString(hint) || utf8.RuneCountInString(hint) > maxHintRunes || len(hint) > maxHintBytes {
		return errors.New("password hint is too long")
	}
	return nil
}

func newRandomBytes(size int, label string) ([]byte, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return nil, fmt.Errorf("generate %s: %w", label, err)
	}
	return value, nil
}

func newDEK() ([]byte, error) { return newRandomBytes(dekBytes, "database key") }

func newVaultID() ([vaultIDBytes]byte, error) {
	var id [vaultIDBytes]byte
	_, err := rand.Read(id[:])
	if err != nil {
		return id, fmt.Errorf("generate vault identifier: %w", err)
	}
	return id, nil
}

func calibrateKDF(password string) argon2Parameters {
	threads := max(min(runtime.NumCPU(), 4), 1)
	params := argon2Parameters{Time: 1, MemoryKiB: minArgonMemoryKiB, Threads: uint8(threads)}
	started := time.Now()
	wipe(argon2.IDKey([]byte(password), make([]byte, saltBytes), 1, params.MemoryKiB, params.Threads, argonKeyBytes))
	if elapsed := time.Since(started); elapsed > 0 {
		// Aim for about 500 ms per key derivation.
		params.Time = min(uint32((500*time.Millisecond+elapsed-1)/elapsed), uint32(maxArgonTime))
	}
	return params
}

func validateArgon2(params argon2Parameters) error {
	if params.Time == 0 || params.Time > maxArgonTime || params.MemoryKiB < minArgonMemoryKiB || params.MemoryKiB > maxArgonMemoryKiB || params.Threads == 0 || params.Threads > maxArgonThreads {
		return errors.New("invalid vault password derivation parameters")
	}
	return nil
}

func deriveKEK(password string, params argon2Parameters, salt []byte) ([]byte, error) {
	if len(password) > 1024 || !utf8.ValidString(password) {
		return nil, ErrInvalidPassword
	}
	if err := validateArgon2(params); err != nil {
		return nil, err
	}
	if len(salt) != saltBytes {
		return nil, errors.New("invalid vault password salt")
	}
	return argon2.IDKey([]byte(password), salt, params.Time, params.MemoryKiB, params.Threads, argonKeyBytes), nil
}

func passwordCipher(password string, params argon2Parameters, salt []byte) (cipher.AEAD, error) {
	key, err := deriveKEK(password, params, salt)
	if err != nil {
		return nil, err
	}
	defer wipe(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create password cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create authenticated password cipher: %w", err)
	}
	return aead, nil
}

func createKeyFile(password, hint string, vaultID [vaultIDBytes]byte, dek []byte, params argon2Parameters) (keyFile, error) {
	if err := validatePassword(password); err != nil {
		return keyFile{}, err
	}
	if err := validateHint(hint); err != nil {
		return keyFile{}, err
	}
	if len(dek) != dekBytes {
		return keyFile{}, errors.New("invalid database key length")
	}
	salt, err := newRandomBytes(saltBytes, "password salt")
	if err != nil {
		return keyFile{}, err
	}
	key := keyFile{Version: keyFormatVersion, VaultID: vaultID, KDF: params, Salt: salt, Hint: hint}
	aead, err := passwordCipher(password, params, salt)
	if err != nil {
		return keyFile{}, err
	}
	key.Nonce, err = newRandomBytes(aead.NonceSize(), "key wrapper nonce")
	if err != nil {
		return keyFile{}, err
	}
	key.WrappedDEK = aead.Seal(nil, key.Nonce, dek, key.additionalData())
	return key, nil
}

func (key keyFile) unwrap(password string) ([]byte, error) {
	aead, err := passwordCipher(password, key.KDF, key.Salt)
	if err != nil {
		return nil, err
	}
	if len(key.Nonce) != aead.NonceSize() || len(key.WrappedDEK) != dekBytes+aead.Overhead() {
		return nil, errors.New("invalid wrapped database key")
	}
	dek, err := aead.Open(nil, key.Nonce, key.WrappedDEK, key.additionalData())
	if err != nil {
		return nil, ErrInvalidPassword
	}
	return dek, nil
}

func (key keyFile) additionalData() []byte {
	var buffer bytes.Buffer
	buffer.Write(keyMagic[:])
	_ = binary.Write(&buffer, binary.BigEndian, key.Version)
	buffer.Write(key.VaultID[:])
	_ = binary.Write(&buffer, binary.BigEndian, key.KDF.Time)
	_ = binary.Write(&buffer, binary.BigEndian, key.KDF.MemoryKiB)
	buffer.WriteByte(key.KDF.Threads)
	buffer.WriteByte(byte(len(key.Salt)))
	_ = binary.Write(&buffer, binary.BigEndian, uint16(len(key.Hint)))
	buffer.Write(key.Salt)
	buffer.WriteString(key.Hint)
	return buffer.Bytes()
}

func encodeKeyFile(key keyFile) ([]byte, error) {
	if key.Version != keyFormatVersion || len(key.Salt) != saltBytes || len(key.Nonce) != nonceBytes || len(key.WrappedDEK) != wrappedDEKBytes {
		return nil, errors.New("invalid vault key fields")
	}
	if err := validateArgon2(key.KDF); err != nil {
		return nil, err
	}
	if err := validateHint(key.Hint); err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	buffer.Write(keyMagic[:])
	_ = binary.Write(&buffer, binary.BigEndian, key.Version)
	buffer.Write(key.VaultID[:])
	_ = binary.Write(&buffer, binary.BigEndian, key.KDF.Time)
	_ = binary.Write(&buffer, binary.BigEndian, key.KDF.MemoryKiB)
	buffer.WriteByte(key.KDF.Threads)
	buffer.WriteByte(byte(len(key.Salt)))
	buffer.WriteByte(byte(len(key.Nonce)))
	_ = binary.Write(&buffer, binary.BigEndian, uint16(len(key.WrappedDEK)))
	_ = binary.Write(&buffer, binary.BigEndian, uint16(len(key.Hint)))
	buffer.Write(key.Salt)
	buffer.Write(key.Nonce)
	buffer.Write(key.WrappedDEK)
	buffer.WriteString(key.Hint)
	if buffer.Len() > keyFileLimit {
		return nil, errors.New("vault key file is too large")
	}
	return buffer.Bytes(), nil
}

func readKeyFile(path string) (keyFile, error) {
	data, err := readBounded(path, keyFileLimit)
	if err != nil {
		return keyFile{}, err
	}
	return decodeKeyFile(data)
}

func decodeKeyFile(data []byte) (keyFile, error) {
	if len(data) > keyFileLimit {
		return keyFile{}, errors.New("vault key file is too large")
	}
	if len(data) < len(keyMagic)+2 {
		return keyFile{}, errors.New("vault key file is truncated")
	}
	if !bytes.Equal(data[:len(keyMagic)], keyMagic[:]) {
		return keyFile{}, errors.New("vault key file has invalid magic")
	}
	version := binary.BigEndian.Uint16(data[len(keyMagic):])
	if version == 0 || version > keyFormatVersion {
		return keyFile{}, fmt.Errorf("%w: vault key version %d", ErrUnsupportedVersion, version)
	}
	return keyFileDecoders[version-1](data)
}

func decodeLegacyKeyFile([]byte) (keyFile, error) { return keyFile{}, ErrLegacyVault }

// readLegacyKeyFile reads a version 1 key file, which has the same fields as
// version 2, so a legacy vault key can still be unwrapped for migration.
func readLegacyKeyFile(path string) (keyFile, error) {
	data, err := readBounded(path, keyFileLimit)
	if err != nil {
		return keyFile{}, err
	}
	if len(data) < len(keyMagic)+2 || !bytes.Equal(data[:len(keyMagic)], keyMagic[:]) || binary.BigEndian.Uint16(data[len(keyMagic):]) != 1 {
		return keyFile{}, errors.New("vault key file is not from a version before 0.10.0")
	}
	return decodeKeyFileFields(data)
}

func decodeKeyFileFields(data []byte) (keyFile, error) {
	reader := bytes.NewReader(data)
	var magic [8]byte
	var key keyFile
	var saltLength, nonceLength uint8
	var wrappedLength, hintLength uint16
	fields := []any{&magic, &key.Version, &key.VaultID, &key.KDF.Time, &key.KDF.MemoryKiB, &key.KDF.Threads, &saltLength, &nonceLength, &wrappedLength, &hintLength}
	for _, field := range fields {
		if err := binary.Read(reader, binary.BigEndian, field); err != nil {
			return keyFile{}, errors.New("vault key file is truncated")
		}
	}
	if err := validateArgon2(key.KDF); err != nil {
		return keyFile{}, err
	}
	if saltLength != saltBytes || nonceLength != nonceBytes || wrappedLength != wrappedDEKBytes || hintLength > maxHintBytes {
		return keyFile{}, errors.New("vault key file has invalid lengths")
	}
	key.Salt = make([]byte, saltLength)
	key.Nonce = make([]byte, nonceLength)
	key.WrappedDEK = make([]byte, wrappedLength)
	hint := make([]byte, hintLength)
	for _, value := range [][]byte{key.Salt, key.Nonce, key.WrappedDEK, hint} {
		if _, err := io.ReadFull(reader, value); err != nil {
			return keyFile{}, errors.New("vault key file is truncated")
		}
	}
	if reader.Len() != 0 {
		return keyFile{}, errors.New("vault key file has trailing data")
	}
	key.Hint = string(hint)
	if err := validateHint(key.Hint); err != nil {
		return keyFile{}, err
	}
	return key, nil
}

func encodeAutoUnlock(protector protection.Protector, vaultID [vaultIDBytes]byte, dek []byte) ([]byte, error) {
	if protector == nil || len(dek) != dekBytes {
		return nil, errors.New("automatic unlock is unavailable")
	}
	plain := append([]byte(nil), autoPlainMagic[:]...)
	plain = binary.BigEndian.AppendUint16(plain, autoFormatVersion)
	plain = append(plain, vaultID[:]...)
	plain = append(plain, dek...)
	ciphertext, err := protector.Protect(plain)
	wipe(plain)
	if err != nil {
		return nil, err
	}
	defer wipe(ciphertext)
	if len(ciphertext) == 0 || len(ciphertext) > autoFileLimit-autoHeaderBytes {
		return nil, errors.New("automatic unlock data is too large")
	}
	result := append([]byte(nil), autoMagic[:]...)
	result = binary.BigEndian.AppendUint16(result, autoFormatVersion)
	result = binary.BigEndian.AppendUint32(result, uint32(len(ciphertext)))
	result = append(result, ciphertext...)
	return result, nil
}

func decodeAutoUnlock(data []byte, protector protection.Protector) ([vaultIDBytes]byte, []byte, error) {
	var vaultID [vaultIDBytes]byte
	// The header holds an 8-byte magic, a 2-byte version, and a 4-byte length.
	if len(data) < autoHeaderBytes || len(data) > autoFileLimit || !bytes.Equal(data[:8], autoMagic[:]) {
		return vaultID, nil, errors.New("automatic unlock file is corrupted")
	}
	version := binary.BigEndian.Uint16(data[8:10])
	if version == 0 || version > autoFormatVersion {
		return vaultID, nil, fmt.Errorf("%w: automatic unlock version %d", ErrUnsupportedVersion, version)
	}
	length := binary.BigEndian.Uint32(data[10:14])
	if length == 0 || int(length) != len(data)-autoHeaderBytes || protector == nil {
		return vaultID, nil, errors.New("automatic unlock file is corrupted")
	}
	plain, err := protector.Unprotect(data[autoHeaderBytes:])
	if err != nil {
		return vaultID, nil, fmt.Errorf("%w: %w", ErrAutomaticUnlockUnavailable, err)
	}
	defer wipe(plain)
	return autoUnlockDecoders[version-1](plain)
}

func decodeAutoUnlockV1(plain []byte) ([vaultIDBytes]byte, []byte, error) {
	var vaultID [vaultIDBytes]byte
	// 8-byte magic and 2-byte version, then the vault ID and the key.
	if len(plain) != 8+2+vaultIDBytes+dekBytes || !bytes.Equal(plain[:8], autoPlainMagic[:]) || binary.BigEndian.Uint16(plain[8:10]) != 1 {
		return vaultID, nil, errors.New("automatic unlock file is corrupted")
	}
	copy(vaultID[:], plain[10:10+vaultIDBytes])
	dek := append([]byte(nil), plain[10+vaultIDBytes:]...)
	return vaultID, dek, nil
}

func readBounded(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open vault file: %w", err)
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", filepath.Base(path), err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%q is too large", filepath.Base(path))
	}
	return data, nil
}

func wipe(data []byte) {
	for index := range data {
		data[index] = 0
	}
	runtime.KeepAlive(data)
}
