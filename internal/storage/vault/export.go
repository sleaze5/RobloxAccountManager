package vault

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
)

const (
	exportFormatVersion = uint16(1)
	exportFileLimit     = 64 * 1024 * 1024
	exportExtension     = ".ramexport"
	exportNamePrefix    = "RobloxAccountManager-"
	// 8-byte magic, 2-byte version, 4-byte Argon2 time, 4-byte memory, 1-byte threads, 16-byte salt, and 12-byte nonce.
	exportHeaderBytes = 8 + 2 + 4 + 4 + 1 + 16 + 12
)

var (
	ErrExportPasscode     = errors.New("export passcode is incorrect or the file is damaged")
	ErrUnsupportedExport  = errors.New("export file is from a newer version")
	ErrInvalidExport      = errors.New("file is not a valid export")
	ErrWeakExportPasscode = errors.New("export passcode does not meet every requirement")
	exportMagic           = [8]byte{'R', 'A', 'M', 'X', 'P', 'R', 'T', 0}
)

var exportDecoders = [...]func([]byte, string) ([]byte, error){openExportV1}

var _ = [1]struct{}{}[len(exportDecoders)-int(exportFormatVersion)]

type ExportInfo struct {
	Name        string `json:"name"`
	CreatedAtMS int64  `json:"createdAtMs"`
}

func (manager *Manager) ExportsDirectory() string { return manager.paths.ExportsRoot }

func (manager *Manager) WriteExport(passcode string, payload []byte) (ExportInfo, error) {
	sealed, err := sealExport(passcode, payload)
	if err != nil {
		return ExportInfo{}, err
	}
	if err := appdata.PreparePrivateDirectory(manager.paths.ExportsRoot); err != nil {
		return ExportInfo{}, err
	}
	now := time.Now()
	base := exportNamePrefix + now.Format("2006-01-02")
	name := base + exportExtension
	for suffix := 2; fileExists(filepath.Join(manager.paths.ExportsRoot, name)); suffix++ {
		name = fmt.Sprintf("%s-%d%s", base, suffix, exportExtension)
	}
	if err := appdata.WritePrivateFile(filepath.Join(manager.paths.ExportsRoot, name), sealed); err != nil {
		return ExportInfo{}, fmt.Errorf("write export file: %w", err)
	}
	return ExportInfo{Name: name, CreatedAtMS: now.UnixMilli()}, nil
}

func (manager *Manager) ListExports() ([]ExportInfo, error) {
	entries, err := os.ReadDir(manager.paths.ExportsRoot)
	if errors.Is(err, os.ErrNotExist) {
		return []ExportInfo{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list export files: %w", err)
	}
	result := make([]ExportInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !validExportName(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		result = append(result, ExportInfo{Name: entry.Name(), CreatedAtMS: info.ModTime().UnixMilli()})
	}
	sort.Slice(result, func(left, right int) bool { return result[left].CreatedAtMS > result[right].CreatedAtMS })
	return result, nil
}

func (manager *Manager) ReadExport(name, passcode string) ([]byte, error) {
	if !validExportName(name) {
		return nil, errors.New("invalid export file selection")
	}
	data, err := readBounded(filepath.Join(manager.paths.ExportsRoot, name), exportFileLimit)
	if err != nil {
		return nil, err
	}
	return openExport(data, passcode)
}

func validExportName(name string) bool {
	return strings.HasSuffix(name, exportExtension) && len(name) > len(exportExtension) && filepath.Base(name) == name && !strings.ContainsAny(name, `/\:`)
}

func validateExportPasscode(passcode string) error {
	if !utf8.ValidString(passcode) || !CheckPassword(passcode).Valid() {
		return ErrWeakExportPasscode
	}
	return nil
}

func sealExport(passcode string, payload []byte) ([]byte, error) {
	if err := validateExportPasscode(passcode); err != nil {
		return nil, err
	}
	params, err := calibrateKDF(passcode)
	if err != nil {
		return nil, err
	}
	salt, err := newRandomBytes(16, "export salt")
	if err != nil {
		return nil, err
	}
	aead, err := exportCipher(passcode, params, salt)
	if err != nil {
		return nil, err
	}
	nonce, err := newRandomBytes(aead.NonceSize(), "export nonce")
	if err != nil {
		return nil, err
	}
	header := exportHeader(exportFormatVersion, params, salt, nonce)
	return aead.Seal(bytes.Clone(header), nonce, payload, header), nil
}

func openExport(data []byte, passcode string) ([]byte, error) {
	if len(data) < len(exportMagic)+2 || !bytes.Equal(data[:len(exportMagic)], exportMagic[:]) {
		return nil, ErrInvalidExport
	}
	version := binary.BigEndian.Uint16(data[len(exportMagic):])
	if version == 0 {
		return nil, ErrInvalidExport
	}
	if version > exportFormatVersion {
		return nil, fmt.Errorf("%w: export version %d", ErrUnsupportedExport, version)
	}
	return exportDecoders[version-1](data, passcode)
}

func openExportV1(data []byte, passcode string) ([]byte, error) {
	if len(data) < exportHeaderBytes {
		return nil, ErrInvalidExport
	}
	offset := len(exportMagic) + 2
	params := argon2Parameters{
		Time:      binary.BigEndian.Uint32(data[offset:]),
		MemoryKiB: binary.BigEndian.Uint32(data[offset+4:]),
		Threads:   data[offset+8],
	}
	offset += 9
	salt := data[offset : offset+16]
	nonce := data[offset+16 : exportHeaderBytes]
	aead, err := exportCipher(passcode, params, salt)
	if errors.Is(err, ErrInvalidPassword) {
		return nil, ErrExportPasscode
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidExport, err)
	}
	payload, err := aead.Open(nil, nonce, data[exportHeaderBytes:], data[:exportHeaderBytes])
	if err != nil {
		return nil, ErrExportPasscode
	}
	return payload, nil
}

func exportCipher(passcode string, params argon2Parameters, salt []byte) (cipher.AEAD, error) {
	key, err := deriveKEK(passcode, params, salt)
	if err != nil {
		return nil, err
	}
	defer wipe(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create export cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create authenticated export cipher: %w", err)
	}
	return aead, nil
}

func exportHeader(version uint16, params argon2Parameters, salt, nonce []byte) []byte {
	header := make([]byte, 0, exportHeaderBytes)
	header = append(header, exportMagic[:]...)
	header = binary.BigEndian.AppendUint16(header, version)
	header = binary.BigEndian.AppendUint32(header, params.Time)
	header = binary.BigEndian.AppendUint32(header, params.MemoryKiB)
	header = append(header, params.Threads)
	header = append(header, salt...)
	return append(header, nonce...)
}
