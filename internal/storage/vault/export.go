package vault

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
)

const (
	exportFormatVersion = uint16(2)
	exportFileLimit     = 64 * 1024 * 1024
	exportExtension     = ".ramexport"
	exportNamePrefix    = "RobloxAccountManager-"
	// Every version starts with an 8-byte magic, a 2-byte version, and an 8-byte creation time in Unix milliseconds,
	// so the creation time never depends on filesystem timestamps.
	exportPrefixBytes = 8 + 2 + 8
	// Version 1 continues with a 4-byte Argon2 time, 4-byte memory, 1-byte threads, 16-byte salt, and 12-byte nonce.
	exportHeaderBytes = exportPrefixBytes + 4 + 4 + 1 + 16 + 12
	// Version 2 continues with a 2-byte hint length and the plaintext password hint. The header authenticates the hint.
	exportHintLengthBytes = 2
)

var (
	ErrExportPassword     = errors.New("export password is incorrect or the file is damaged")
	ErrUnsupportedExport  = errors.New("export file is from a newer version")
	ErrInvalidExport      = errors.New("file is not a valid export")
	errWeakExportPassword = errors.New("export password does not meet every requirement")
	errExportHint         = errors.New("export password hint is too long")
	ErrExportHintPassword = errors.New("export password hint contains the export password")
	exportMagic           = [8]byte{'R', 'A', 'M', 'X', 'P', 'R', 'T', 0}
)

var exportDecoders = [...]func([]byte, string) ([]byte, error){openExportV1, openExportV2}

var _ = [1]struct{}{}[len(exportDecoders)-int(exportFormatVersion)]

type ExportFile struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	CreatedAtMs int64  `json:"createdAtMs"`
}

// ExportsDirectory creates the export folder and returns it. If the folder cannot be created, it returns the data folder,
// so a file dialog still opens in a useful place.
func (manager *Manager) ExportsDirectory() string {
	if err := appdata.PreparePrivateDirectory(manager.paths.ExportsRoot); err != nil {
		return filepath.Dir(manager.paths.ExportsRoot)
	}
	return manager.paths.ExportsRoot
}

func (manager *Manager) DefaultExportName() string {
	base := exportNamePrefix + time.Now().Format("2006-01-02")
	name := base + exportExtension
	for suffix := 2; fileExists(filepath.Join(manager.paths.ExportsRoot, name)); suffix++ {
		name = fmt.Sprintf("%s-%d%s", base, suffix, exportExtension)
	}
	return name
}

func (manager *Manager) WriteExport(path, password, hint string, payload []byte) (ExportFile, error) {
	if !strings.EqualFold(filepath.Ext(path), exportExtension) {
		path += exportExtension
	}
	now := time.Now()
	sealed, err := sealExport(password, hint, payload, now.UnixMilli())
	if err != nil {
		return ExportFile{}, err
	}
	if err := writeExportFile(path, sealed); err != nil {
		return ExportFile{}, err
	}
	return ExportFile{Name: filepath.Base(path), Path: path, CreatedAtMs: now.UnixMilli()}, nil
}

// writeExportFile does not set a private access list, because the user can choose a drive that has none, such as FAT32.
// The file is encrypted, and the export folder that the app creates is already private.
func writeExportFile(path string, encodedExport []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create export file: %w", err)
	}
	temporary := file.Name()
	ok := false
	defer func() {
		_ = file.Close()
		if !ok {
			_ = os.Remove(temporary)
		}
	}()
	if _, err := file.Write(encodedExport); err != nil {
		return fmt.Errorf("write export file: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("flush export file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close export file: %w", err)
	}
	if err := appdata.ReplaceFile(temporary, path); err != nil {
		return fmt.Errorf("save export file: %w", err)
	}
	ok = true
	return nil
}

func (manager *Manager) ReadExport(path, password string) (ExportFile, []byte, error) {
	encodedExport, err := readBounded(path, exportFileLimit)
	if err != nil {
		return ExportFile{}, nil, err
	}
	payload, err := openExport(encodedExport, password)
	if err != nil {
		return ExportFile{}, nil, err
	}
	return ExportFile{Name: filepath.Base(path), Path: path, CreatedAtMs: int64(binary.BigEndian.Uint64(encodedExport[len(exportMagic)+2:]))}, payload, nil
}

// ExportDetails describes an export file before it is opened with its password.
type ExportDetails struct {
	FormatVersion    int    `json:"formatVersion"`
	MinFormatVersion int    `json:"minFormatVersion"`
	MaxFormatVersion int    `json:"maxFormatVersion"`
	Supported        bool   `json:"supported"`
	Hint             string `json:"hint,omitempty"`
}

// ReadExportDetails reads the format version and password hint of an export
// file without its password. Files from format version 1 have no hint, and
// files from a newer format version are reported as unsupported.
func (manager *Manager) ReadExportDetails(path string) (ExportDetails, error) {
	encodedExport, err := readBounded(path, exportFileLimit)
	if err != nil {
		return ExportDetails{}, err
	}
	if len(encodedExport) < exportPrefixBytes || !bytes.Equal(encodedExport[:len(exportMagic)], exportMagic[:]) {
		return ExportDetails{}, ErrInvalidExport
	}
	version := binary.BigEndian.Uint16(encodedExport[len(exportMagic):])
	if version == 0 {
		return ExportDetails{}, ErrInvalidExport
	}
	details := ExportDetails{FormatVersion: int(version), MinFormatVersion: 1, MaxFormatVersion: int(exportFormatVersion), Supported: version <= exportFormatVersion}
	if version >= 2 && details.Supported {
		headerEnd, err := exportHeaderEnd(encodedExport)
		if err != nil {
			return ExportDetails{}, err
		}
		details.Hint = string(encodedExport[exportHeaderBytes+exportHintLengthBytes : headerEnd])
	}
	return details, nil
}

func ValidateExportPassword(password string) error {
	if !meetsPasswordRules(password) {
		return errWeakExportPassword
	}
	return nil
}

func ValidateExportHint(hint, password string) error {
	if validateHint(hint) != nil {
		return errExportHint
	}
	if password != "" && strings.Contains(strings.ToLower(hint), strings.ToLower(password)) {
		return ErrExportHintPassword
	}
	return nil
}

func sealExport(password, hint string, payload []byte, createdAtMs int64) ([]byte, error) {
	if err := ValidateExportPassword(password); err != nil {
		return nil, err
	}
	if err := ValidateExportHint(hint, password); err != nil {
		return nil, err
	}
	parameters := calibrateKDF(password)
	salt, err := newRandomBytes(saltBytes, "export salt")
	if err != nil {
		return nil, err
	}
	aead, err := passwordCipher(password, parameters, salt)
	if err != nil {
		return nil, err
	}
	nonce, err := newRandomBytes(aead.NonceSize(), "export nonce")
	if err != nil {
		return nil, err
	}
	header := exportHeader(createdAtMs, parameters, salt, nonce)
	header = binary.BigEndian.AppendUint16(header, uint16(len(hint)))
	header = append(header, hint...)
	return aead.Seal(bytes.Clone(header), nonce, payload, header), nil
}

func exportVersion(encodedExport []byte) (uint16, error) {
	if len(encodedExport) < exportPrefixBytes || !bytes.Equal(encodedExport[:len(exportMagic)], exportMagic[:]) {
		return 0, ErrInvalidExport
	}
	version := binary.BigEndian.Uint16(encodedExport[len(exportMagic):])
	if version == 0 {
		return 0, ErrInvalidExport
	}
	if version > exportFormatVersion {
		return 0, fmt.Errorf("%w: export version %d", ErrUnsupportedExport, version)
	}
	return version, nil
}

func openExport(encodedExport []byte, password string) ([]byte, error) {
	version, err := exportVersion(encodedExport)
	if err != nil {
		return nil, err
	}
	return exportDecoders[version-1](encodedExport, password)
}

func openExportV1(encodedExport []byte, password string) ([]byte, error) {
	if len(encodedExport) < exportHeaderBytes {
		return nil, ErrInvalidExport
	}
	return openSealedExport(encodedExport, password, exportHeaderBytes)
}

func openExportV2(encodedExport []byte, password string) ([]byte, error) {
	headerEnd, err := exportHeaderEnd(encodedExport)
	if err != nil {
		return nil, err
	}
	return openSealedExport(encodedExport, password, headerEnd)
}

// exportHeaderEnd returns where the version 2 header, which ends with the
// password hint, stops and the encrypted payload starts.
func exportHeaderEnd(encodedExport []byte) (int, error) {
	if len(encodedExport) < exportHeaderBytes+exportHintLengthBytes {
		return 0, ErrInvalidExport
	}
	hintLength := int(binary.BigEndian.Uint16(encodedExport[exportHeaderBytes:]))
	headerEnd := exportHeaderBytes + exportHintLengthBytes + hintLength
	if len(encodedExport) < headerEnd || validateHint(string(encodedExport[exportHeaderBytes+exportHintLengthBytes:headerEnd])) != nil {
		return 0, ErrInvalidExport
	}
	return headerEnd, nil
}

// openSealedExport decrypts the payload after headerEnd. The whole header,
// including the version 2 hint, is the authenticated data.
func openSealedExport(encodedExport []byte, password string, headerEnd int) ([]byte, error) {
	offset := exportPrefixBytes
	parameters := argon2Parameters{
		Time:      binary.BigEndian.Uint32(encodedExport[offset:]),
		MemoryKiB: binary.BigEndian.Uint32(encodedExport[offset+4:]),
		Threads:   encodedExport[offset+8],
	}
	offset += 9
	salt := encodedExport[offset : offset+saltBytes]
	nonce := encodedExport[offset+saltBytes : exportHeaderBytes]
	aead, err := passwordCipher(password, parameters, salt)
	if errors.Is(err, ErrInvalidPassword) {
		return nil, ErrExportPassword
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidExport, err)
	}
	payload, err := aead.Open(nil, nonce, encodedExport[headerEnd:], encodedExport[:headerEnd])
	if err != nil {
		return nil, ErrExportPassword
	}
	return payload, nil
}

func exportHeader(createdAtMs int64, parameters argon2Parameters, salt, nonce []byte) []byte {
	header := make([]byte, 0, exportHeaderBytes)
	header = append(header, exportMagic[:]...)
	header = binary.BigEndian.AppendUint16(header, exportFormatVersion)
	header = binary.BigEndian.AppendUint64(header, uint64(createdAtMs))
	header = binary.BigEndian.AppendUint32(header, parameters.Time)
	header = binary.BigEndian.AppendUint32(header, parameters.MemoryKiB)
	header = append(header, parameters.Threads)
	header = append(header, salt...)
	return append(header, nonce...)
}
