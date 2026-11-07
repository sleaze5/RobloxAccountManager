package vault

import (
	"bytes"
	"context"
	"crypto/hkdf"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/vfs/xts"
	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
	"github.com/sleaze5/RobloxAccountManager/internal/logging"
	"github.com/sleaze5/RobloxAccountManager/internal/platform/protection"
)

var (
	ErrLocked                     = errors.New("vault is locked")
	errAlreadyExists              = errors.New("vault already exists")
	ErrNotInitialized             = errors.New("vault is not initialized")
	ErrInvalidPassword            = errors.New("master password or vault key data is incorrect")
	ErrUnsupportedVersion         = errors.New("unsupported key-file version")
	ErrLegacyVault                = errors.New("vault was created by a version before 0.10.0")
	ErrUnsupportedSchema          = errors.New("unsupported vault schema")
	ErrVaultMismatch              = errors.New("database and key identifiers do not match")
	ErrAutomaticUnlockUnavailable = errors.New("automatic unlock is unavailable for this user")
	ErrIncomplete                 = errors.New("vault files are incomplete")
	errLockTimeout                = errors.New("vault operations did not stop before the lock timeout")
	errBusy                       = errors.New("another vault operation is in progress")
)

type FileState string

const (
	FileStateEmpty      FileState = "empty"
	FileStateReady      FileState = "ready"
	FileStateIncomplete FileState = "incomplete"
	FileStateLegacy     FileState = "legacy"
)

const lockTimeout = 5 * time.Second

type Manager struct {
	mutex                 sync.Mutex
	backupMutex           sync.Mutex
	paths                 appdata.Paths
	protector             protection.Protector
	database              *sql.DB
	vaultKey              []byte
	vaultID               [vaultIDBytes]byte
	active                int
	closing               bool
	drained               chan struct{}
	unlockedAutomatically bool
	logger                *slog.Logger
}

func NewManager(paths appdata.Paths, protector protection.Protector, logger *slog.Logger) *Manager {
	logging.Diagnostic(logger, "vault formats supported", "operation", "vault-load",
		"supported_schema_version", schemaVersion, "supported_key_file_version", keyFormatVersion,
		"supported_auto_unlock_version", autoFormatVersion)
	return &Manager{paths: paths, protector: protector, logger: logger}
}

func (manager *Manager) FileState() FileState {
	hasDatabase := fileExists(manager.paths.Vault)
	hasKey := fileExists(manager.paths.VaultKey)
	hasAutomaticUnlock := fileExists(manager.paths.AutoUnlockKey)
	hasStagedCreation := directoryHasEntries(manager.paths.CreateStaging)
	if !hasDatabase && !hasKey && !hasAutomaticUnlock && !hasStagedCreation {
		return FileStateEmpty
	}
	if hasDatabase && hasKey && !hasStagedCreation {
		_, err := readKeyFile(manager.paths.VaultKey)
		switch {
		case err == nil:
			return FileStateReady
		case errors.Is(err, ErrLegacyVault):
			return FileStateLegacy
		}
		return FileStateIncomplete
	}
	return FileStateIncomplete
}

func (manager *Manager) Initialized() bool { return manager.FileState() != FileStateEmpty }

func (manager *Manager) Unlocked() bool {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	return manager.database != nil && !manager.closing
}

func (manager *Manager) UnlockedAutomatically() bool {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	return manager.database != nil && manager.unlockedAutomatically
}

func (manager *Manager) AutomaticUnlockEnabled() bool { return fileExists(manager.paths.AutoUnlockKey) }

func (manager *Manager) ValidAutomaticUnlock() bool {
	data, err := readBounded(manager.paths.AutoUnlockKey, autoFileLimit)
	if err != nil {
		return false
	}
	identifier, vaultKey, err := decodeAutoUnlock(data, manager.protector)
	if err != nil {
		return false
	}
	defer wipe(vaultKey)
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	return manager.database != nil &&
		!manager.closing &&
		identifier == manager.vaultID &&
		len(vaultKey) == len(manager.vaultKey) &&
		subtle.ConstantTimeCompare(vaultKey, manager.vaultKey) == 1
}

func (manager *Manager) PasswordHint() string {
	key, err := readKeyFile(manager.paths.VaultKey)
	if errors.Is(err, ErrLegacyVault) {
		key, err = readLegacyKeyFile(manager.paths.VaultKey)
	}
	if err != nil {
		return ""
	}
	return key.Hint
}

func (manager *Manager) Create(ctx context.Context, password, hint string, automaticUnlock bool) error {
	if manager.FileState() != FileStateEmpty {
		return errAlreadyExists
	}
	if _, err := os.Stat(manager.paths.VaultRoot); err != nil {
		return fmt.Errorf("vault directory is unavailable: %w", err)
	}
	if err := validatePassword(password); err != nil {
		return err
	}
	params := calibrateKDF(password)
	identifier, err := newVaultID()
	if err != nil {
		return err
	}
	vaultKey, err := newVaultKey()
	if err != nil {
		return err
	}
	defer func() { wipe(vaultKey) }()
	key, err := createKeyFile(password, hint, identifier, vaultKey, params)
	if err != nil {
		return err
	}
	encodedKey, err := encodeKeyFile(key)
	if err != nil {
		return err
	}
	var encodedAuto []byte
	if automaticUnlock {
		encodedAuto, err = encodeAutoUnlock(manager.protector, identifier, vaultKey)
		if err != nil {
			return err
		}
	}
	if err := manager.stageCreatedVault(ctx, identifier, vaultKey, encodedKey, encodedAuto, automaticUnlock); err != nil {
		return err
	}
	if err := manager.installUnlocked(ctx, vaultKey, identifier, automaticUnlock); err != nil {
		return err
	}
	vaultKey = nil
	logging.Diagnostic(manager.logger, "vault created", "operation", "vault-create", "schema_version", schemaVersion,
		"key_file_version", keyFormatVersion, "automatic_unlock", automaticUnlock)
	return nil
}

func (manager *Manager) Unlock(ctx context.Context, password string) error {
	if !fileExists(manager.paths.Vault) || !fileExists(manager.paths.VaultKey) || directoryHasEntries(manager.paths.CreateStaging) {
		if manager.FileState() == FileStateEmpty {
			return ErrNotInitialized
		}
		return ErrIncomplete
	}
	usedNext := false
	err := manager.unlockPasswordCandidate(ctx, manager.paths.VaultKey, password)
	if err != nil && fileExists(manager.paths.VaultKey+".next") {
		if nextError := manager.unlockPasswordCandidate(ctx, manager.paths.VaultKey+".next", password); nextError == nil {
			usedNext = true
			err = nil
		}
	}
	if err != nil {
		return err
	}
	if err := manager.completeUnlock(ctx, usedNext, password); err != nil {
		return err
	}
	manager.logger.InfoContext(ctx, "vault unlocked with master password", "operation", "vault-unlock")
	return nil
}

func (manager *Manager) unlockPasswordCandidate(ctx context.Context, path, password string) error {
	key, err := readKeyFile(path)
	if err != nil {
		return fmt.Errorf("read vault key file: %w", err)
	}
	logging.Diagnostic(manager.logger, "vault key file loaded", "operation", "vault-load", "key_file_version", key.Version)
	vaultKey, err := key.unwrap(password)
	if err != nil {
		return err
	}
	return manager.installUnlocked(ctx, vaultKey, key.VaultID, false)
}

func (manager *Manager) AutoUnlock(ctx context.Context) error {
	logging.Diagnostic(manager.logger, "loading vault", "operation", "vault-load", "file_state", manager.FileState(), "schema_status", "not_loaded")
	if directoryHasEntries(manager.paths.CreateStaging) {
		return ErrIncomplete
	}
	key, keyError := readKeyFile(manager.paths.VaultKey)
	if keyError == nil {
		logging.Diagnostic(manager.logger, "vault key file loaded", "operation", "vault-load", "key_file_version", key.Version)
	}
	if !fileExists(manager.paths.Vault) || !fileExists(manager.paths.AutoUnlockKey) {
		if manager.FileState() == FileStateEmpty {
			return ErrNotInitialized
		}
		return ErrLocked
	}
	if errors.Is(keyError, ErrUnsupportedVersion) || errors.Is(keyError, ErrLegacyVault) {
		return keyError
	}
	usedNext := false
	err := manager.unlockAutoCandidate(ctx, manager.paths.AutoUnlockKey)
	if err != nil && fileExists(manager.paths.AutoUnlockKey+".next") {
		if nextError := manager.unlockAutoCandidate(ctx, manager.paths.AutoUnlockKey+".next"); nextError == nil {
			usedNext = true
			err = nil
		}
	}
	if err != nil {
		return err
	}
	if !usedNext && fileExists(manager.paths.VaultKey+".next") {
		usedNext = manager.rotationMatchesUnlockedKey()
	}
	if err := manager.completeUnlock(ctx, usedNext, ""); err != nil {
		return err
	}
	manager.logger.InfoContext(ctx, "vault unlocked with automatic unlock", "operation", "vault-unlock")
	return nil
}

func (manager *Manager) unlockAutoCandidate(ctx context.Context, path string) error {
	data, err := readBounded(path, autoFileLimit)
	if err != nil {
		return err
	}
	identifier, vaultKey, err := decodeAutoUnlock(data, manager.protector)
	if err != nil {
		return err
	}
	logging.Diagnostic(manager.logger, "automatic unlock file loaded", "operation", "vault-load", "auto_unlock_version", autoFormatVersion)
	return manager.installUnlocked(ctx, vaultKey, identifier, true)
}

func (manager *Manager) installUnlocked(ctx context.Context, vaultKey []byte, identifier [vaultIDBytes]byte, automatic bool) error {
	if len(vaultKey) != vaultKeyBytes {
		wipe(vaultKey)
		return errors.New("invalid database key length")
	}
	database, err := openAndVerify(ctx, manager.paths.Vault, vaultKey, identifier, manager.logger)
	if err != nil {
		wipe(vaultKey)
		return err
	}
	manager.mutex.Lock()
	if manager.database != nil {
		manager.mutex.Unlock()
		_ = database.Close()
		wipe(vaultKey)
		return nil
	}
	manager.database = database
	manager.vaultKey = vaultKey
	manager.vaultID = identifier
	manager.unlockedAutomatically = automatic
	manager.closing = false
	manager.mutex.Unlock()
	return nil
}

func (manager *Manager) Lease(_ context.Context) (*sql.DB, func(), error) {
	manager.mutex.Lock()
	if manager.database == nil || manager.closing {
		manager.mutex.Unlock()
		return nil, nil, ErrLocked
	}
	manager.active++
	database := manager.database
	manager.mutex.Unlock()
	var once sync.Once
	release := func() {
		once.Do(func() {
			manager.mutex.Lock()
			manager.active--
			if manager.closing && manager.active == 0 && manager.drained != nil {
				close(manager.drained)
				manager.drained = nil
			}
			manager.mutex.Unlock()
		})
	}
	return database, release, nil
}

func (manager *Manager) Lock() error { return manager.close(true) }

func (manager *Manager) Close() error { return manager.close(false) }

func (manager *Manager) close(manual bool) error {
	database, finish, err := manager.beginExclusive(context.Background())
	if errors.Is(err, ErrLocked) {
		return nil
	}
	if err != nil {
		return err
	}
	manager.dropDatabase()
	finish()
	if err := database.Close(); err != nil {
		return fmt.Errorf("close vault: %w", err)
	}
	if manual {
		if err := os.Remove(manager.paths.AutoUnlockKey); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove automatic unlock file: %w", err)
		}
	}
	manager.logger.Info("vault locked", "operation", "vault-lock", "automatic_unlock_revoked", manual)
	return nil
}

func (manager *Manager) TestPassword(ctx context.Context, password string) error {
	key, err := readKeyFile(manager.paths.VaultKey)
	if err != nil {
		return err
	}
	vaultKey, err := key.unwrap(password)
	if err != nil {
		return err
	}
	defer wipe(vaultKey)
	manager.mutex.Lock()
	valid := manager.database != nil && key.VaultID == manager.vaultID && len(manager.vaultKey) == len(vaultKey) && subtle.ConstantTimeCompare(manager.vaultKey, vaultKey) == 1
	manager.mutex.Unlock()
	if !valid {
		return ErrInvalidPassword
	}
	return nil
}

func (manager *Manager) ResetVault() error {
	if manager.Unlocked() {
		return errors.New("lock the vault before resetting it")
	}
	for _, path := range []string{manager.paths.Vault, manager.paths.VaultKey, manager.paths.AutoUnlockKey, manager.paths.Migration, manager.paths.Vault + ".next", manager.paths.VaultKey + ".next", manager.paths.AutoUnlockKey + ".next"} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove %q: %w", filepath.Base(path), err)
		}
	}
	if err := manager.removeManaged(manager.paths.CreateStaging, manager.paths.VaultRoot); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove vault staging data: %w", err)
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func directoryHasEntries(path string) bool {
	entries, err := os.ReadDir(path)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return err != nil || len(entries) > 0
}

func openAndVerify(ctx context.Context, path string, vaultKey []byte, identifier [vaultIDBytes]byte, logger *slog.Logger) (*sql.DB, error) {
	database, err := openDatabase(ctx, path, vaultKey, logger)
	if err != nil {
		return nil, err
	}
	if err := verifyOpenDatabase(ctx, database, identifier, logger); err != nil {
		_ = database.Close()
		return nil, err
	}
	return database, nil
}

func verifyOpenDatabase(ctx context.Context, database *sql.DB, identifier [vaultIDBytes]byte, logger *slog.Logger) error {
	if err := verifySchema(ctx, database, identifier, logger); err != nil {
		return err
	}
	var integrity string
	if err := database.QueryRowContext(ctx, `PRAGMA quick_check`).Scan(&integrity); err != nil || integrity != "ok" {
		return errors.New("vault integrity check failed")
	}
	return nil
}

func openDatabase(ctx context.Context, path string, vaultKey []byte, logger *slog.Logger) (*sql.DB, error) {
	dataSourceName, err := databaseDSN(path, vaultKey)
	if err != nil {
		return nil, err
	}
	database, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("open encrypted vault: %w", err)
	}
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)
	database.SetConnMaxLifetime(0)
	closeOnError := func(err error) (*sql.DB, error) {
		_ = database.Close()
		return nil, err
	}
	openContext, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var sqliteVersion string
	if err := database.QueryRowContext(openContext, `SELECT sqlite_version()`).Scan(&sqliteVersion); err != nil {
		return closeOnError(fmt.Errorf("read SQLite version: %w", err))
	}
	logging.Diagnostic(logger, "database engine loaded", "operation", "vault-load", "sqlite_version", sqliteVersion, "encryption", "aes-256-xts")
	if err := database.QueryRowContext(openContext, `SELECT count(*) FROM sqlite_master`).Scan(new(int)); err != nil {
		return closeOnError(fmt.Errorf("verify encrypted vault key: %w", err))
	}
	if err := requireEncrypted(path); err != nil {
		return closeOnError(err)
	}
	var foreignKeys, secureDelete int
	var journalMode string
	if err := database.QueryRowContext(openContext, `PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		return closeOnError(errors.New("foreign key enforcement is unavailable"))
	}
	if err := database.QueryRowContext(openContext, `PRAGMA secure_delete`).Scan(&secureDelete); err != nil || secureDelete != 1 {
		return closeOnError(errors.New("secure deletion is unavailable"))
	}
	if err := database.QueryRowContext(openContext, `PRAGMA journal_mode`).Scan(&journalMode); err != nil || !strings.EqualFold(journalMode, "delete") {
		return closeOnError(errors.New("portable journal mode is unavailable"))
	}
	return database, nil
}

// requireEncrypted rejects a database file that SQLite wrote without the
// encrypting VFS, because its first bytes would be the plaintext header.
func requireEncrypted(path string) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open vault database: %w", err)
	}
	defer func() { _ = file.Close() }()
	header := make([]byte, len(plaintextHeader))
	if _, err := io.ReadFull(file, header); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil
		}
		return fmt.Errorf("read vault database header: %w", err)
	}
	if bytes.Equal(header, plaintextHeader) {
		return errors.New("vault database is not encrypted")
	}
	return nil
}

var plaintextHeader = []byte("SQLite format 3\x00")

// databaseKey derives the 64-byte AES-256-XTS key for vault.db from the
// vault key, which stays 32 bytes so the key files keep their format.
func databaseKey(vaultKey []byte) ([]byte, error) {
	key, err := hkdf.Key(sha256.New, vaultKey, nil, "RobloxAccountManager vault.db AES-256-XTS", 64)
	if err != nil {
		return nil, fmt.Errorf("derive vault database key: %w", err)
	}
	return key, nil
}

func databaseDSN(path string, vaultKey []byte) (string, error) {
	key, err := databaseKey(vaultKey)
	if err != nil {
		return "", err
	}
	defer wipe(key)
	// The SQLite VFS opens the URI path as is, so a Windows path keeps its drive
	// letter first, as in file:C:/Users/..., without a leading slash.
	databaseURL := &url.URL{Scheme: "file", Opaque: (&url.URL{Path: filepath.ToSlash(path)}).EscapedPath()}
	query := url.Values{}
	query.Set("vfs", "xts")
	query.Set("hexkey", hex.EncodeToString(key))
	query["_pragma"] = []string{"busy_timeout(5000)", "foreign_keys(1)", "journal_mode(delete)", "secure_delete(1)", "temp_store(memory)"}
	databaseURL.RawQuery = query.Encode()
	return databaseURL.String(), nil
}

func (manager *Manager) stageCreatedVault(ctx context.Context, identifier [vaultIDBytes]byte, vaultKey, encodedKey, encodedAuto []byte, automaticUnlock bool) error {
	if err := appdata.PreparePrivateDirectory(manager.paths.CreateStaging); err != nil {
		return err
	}
	stagedDatabase := filepath.Join(manager.paths.CreateStaging, "vault.db")
	stagedKey := filepath.Join(manager.paths.CreateStaging, "vault.key")
	stagedAuto := filepath.Join(manager.paths.CreateStaging, "autounlock.key")
	if fileExists(stagedDatabase) || fileExists(stagedKey) || fileExists(stagedAuto) {
		return ErrIncomplete
	}
	if err := manager.initializeStagedDatabase(ctx, stagedDatabase, vaultKey, identifier); err != nil {
		return err
	}
	if err := appdata.WritePrivateFile(stagedKey, encodedKey); err != nil {
		return err
	}
	if automaticUnlock {
		if err := manager.writeStagedAutomaticUnlock(stagedAuto, encodedAuto, identifier, vaultKey); err != nil {
			return err
		}
	}
	if err := os.Rename(stagedDatabase, manager.paths.Vault); err != nil {
		return fmt.Errorf("publish vault database: %w", err)
	}
	if automaticUnlock {
		if err := os.Rename(stagedAuto, manager.paths.AutoUnlockKey); err != nil {
			return fmt.Errorf("publish automatic unlock file: %w", err)
		}
	}
	if err := os.Rename(stagedKey, manager.paths.VaultKey); err != nil {
		return fmt.Errorf("publish vault key file: %w", err)
	}
	if err := os.Remove(manager.paths.CreateStaging); err != nil {
		return fmt.Errorf("remove vault staging directory: %w", err)
	}
	return nil
}

func (manager *Manager) initializeStagedDatabase(ctx context.Context, stagedDatabase string, vaultKey []byte, identifier [vaultIDBytes]byte) error {
	database, err := openDatabase(ctx, stagedDatabase, vaultKey, manager.logger)
	if err != nil {
		return err
	}
	if err := initializeSchema(ctx, database, identifier); err != nil {
		_ = database.Close()
		return err
	}
	if err := verifyOpenDatabase(ctx, database, identifier, manager.logger); err != nil {
		_ = database.Close()
		return err
	}
	if err := database.Close(); err != nil {
		return fmt.Errorf("close staged vault: %w", err)
	}
	if err := appdata.RestrictFile(stagedDatabase); err != nil {
		return err
	}
	return nil
}

func (manager *Manager) writeStagedAutomaticUnlock(stagedAuto string, encodedAuto []byte, identifier [vaultIDBytes]byte, vaultKey []byte) error {
	if err := appdata.WritePrivateFile(stagedAuto, encodedAuto); err != nil {
		return err
	}
	storedID, storedVaultKey, err := decodeAutoUnlock(encodedAuto, manager.protector)
	if err != nil || storedID != identifier || subtle.ConstantTimeCompare(storedVaultKey, vaultKey) != 1 {
		wipe(storedVaultKey)
		return errors.New("verify automatic unlock file")
	}
	wipe(storedVaultKey)
	return nil
}

func (manager *Manager) rotationMatchesUnlockedKey() bool {
	if !fileExists(manager.paths.AutoUnlockKey + ".next") {
		return true
	} else if data, readError := readBounded(manager.paths.AutoUnlockKey+".next", autoFileLimit); readError == nil {
		_, nextVaultKey, decodeError := decodeAutoUnlock(data, manager.protector)
		manager.mutex.Lock()
		matches := decodeError == nil && len(nextVaultKey) == len(manager.vaultKey) && subtle.ConstantTimeCompare(nextVaultKey, manager.vaultKey) == 1
		manager.mutex.Unlock()
		wipe(nextVaultKey)
		return matches
	}
	return false
}
