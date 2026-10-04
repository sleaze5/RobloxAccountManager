package vault

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mutecomm/go-sqlcipher/v4"
	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
	"github.com/sleaze5/RobloxAccountManager/internal/logging"
	"github.com/sleaze5/RobloxAccountManager/internal/platform/protection"
)

var (
	ErrLocked             = errors.New("vault is locked")
	ErrAlreadyExists      = errors.New("vault already exists")
	ErrNotInitialized     = errors.New("vault is not initialized")
	ErrInvalidPassword    = errors.New("master password or vault key data is incorrect")
	ErrUnsupportedVersion = errors.New("unsupported key-file version")
	ErrUnsupportedSchema  = errors.New("unsupported vault schema")
	ErrVaultMismatch      = errors.New("database and key identifiers do not match")
	ErrDPAPIUnavailable   = errors.New("automatic unlock is unavailable for this Windows user")
	ErrIncomplete         = errors.New("vault files are incomplete")
	ErrLockTimeout        = errors.New("vault operations did not stop before the lock timeout")
)

type FileState string

const (
	FileStateEmpty      FileState = "empty"
	FileStateReady      FileState = "ready"
	FileStateIncomplete FileState = "incomplete"
)

const lockTimeout = 5 * time.Second

type Manager struct {
	mu                sync.Mutex
	paths             appdata.Paths
	protector         protection.Protector
	db                *sql.DB
	dek               []byte
	vaultID           [vaultIDBytes]byte
	active            int
	closing           bool
	drained           chan struct{}
	unlockedWithDPAPI bool
	logger            *slog.Logger
}

func NewManager(paths appdata.Paths, protector protection.Protector, loggers ...*slog.Logger) *Manager {
	logger := slog.New(slog.DiscardHandler)
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	}
	logging.Diagnostic(logger, "vault formats supported", "operation", "vault-load",
		"supported_schema_version", schemaVersion, "supported_key_file_version", keyFormatVersion,
		"supported_auto_unlock_version", autoFormatVersion)
	return &Manager{paths: paths, protector: protector, logger: logger}
}

func (manager *Manager) FileState() FileState {
	database := fileExists(manager.paths.Vault)
	key := fileExists(manager.paths.VaultKey)
	automatic := fileExists(manager.paths.AutoUnlockKey)
	staged := directoryHasEntries(manager.paths.CreateStaging)
	if !database && !key && !automatic && !staged {
		return FileStateEmpty
	}
	if database && key && !staged {
		if _, err := readKeyFile(manager.paths.VaultKey); err == nil {
			return FileStateReady
		}
		return FileStateIncomplete
	}
	return FileStateIncomplete
}

func (manager *Manager) Initialized() bool { return manager.FileState() != FileStateEmpty }

func (manager *Manager) Unlocked() bool {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	return manager.db != nil && !manager.closing
}

func (manager *Manager) UnlockedWithDPAPI() bool {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	return manager.db != nil && manager.unlockedWithDPAPI
}

func (manager *Manager) AutomaticUnlockEnabled() bool { return fileExists(manager.paths.AutoUnlockKey) }

func (manager *Manager) ValidAutomaticUnlock() bool {
	data, err := readBounded(manager.paths.AutoUnlockKey, autoFileLimit)
	if err != nil {
		return false
	}
	identifier, dek, err := decodeAutoUnlock(data, manager.protector)
	if err != nil {
		return false
	}
	defer wipe(dek)
	manager.mu.Lock()
	defer manager.mu.Unlock()
	return manager.db != nil &&
		!manager.closing &&
		identifier == manager.vaultID &&
		len(dek) == len(manager.dek) &&
		subtle.ConstantTimeCompare(dek, manager.dek) == 1
}

func (manager *Manager) PasswordHint() string {
	key, err := readKeyFile(manager.paths.VaultKey)
	if err != nil {
		return ""
	}
	return key.Hint
}

func (manager *Manager) Create(ctx context.Context, password, hint string, automaticUnlock bool) error {
	if manager.FileState() != FileStateEmpty {
		return ErrAlreadyExists
	}
	if err := ValidatePassword(password); err != nil {
		return err
	}
	params, err := calibrateKDF(password)
	if err != nil {
		return err
	}
	identifier, err := newVaultID()
	if err != nil {
		return err
	}
	dek, err := newDEK()
	if err != nil {
		return err
	}
	defer func() {
		if dek != nil {
			wipe(dek)
		}
	}()
	key, err := createKeyFile(password, hint, identifier, dek, params)
	if err != nil {
		return err
	}
	encodedKey, err := encodeKeyFile(key)
	if err != nil {
		return err
	}
	var encodedAuto []byte
	if automaticUnlock {
		encodedAuto, err = encodeAutoUnlock(manager.protector, identifier, dek)
		if err != nil {
			return err
		}
	}
	if err := appdata.PreparePrivateDirectory(manager.paths.CreateStaging); err != nil {
		return err
	}
	stagedDatabase := filepath.Join(manager.paths.CreateStaging, "vault.db")
	stagedKey := filepath.Join(manager.paths.CreateStaging, "vault.key")
	stagedAuto := filepath.Join(manager.paths.CreateStaging, "autounlock.key")
	if fileExists(stagedDatabase) || fileExists(stagedKey) || fileExists(stagedAuto) {
		return ErrIncomplete
	}
	db, err := openDatabase(ctx, stagedDatabase, dek, manager.logger)
	if err != nil {
		return err
	}
	if err := initializeSchema(ctx, db, identifier); err != nil {
		_ = db.Close()
		return err
	}
	if err := verifyOpenDatabase(ctx, db, identifier, manager.logger); err != nil {
		_ = db.Close()
		return err
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("close staged vault: %w", err)
	}
	if err := appdata.RestrictFile(stagedDatabase); err != nil {
		return err
	}
	if err := appdata.WritePrivateFile(stagedKey, encodedKey); err != nil {
		return err
	}
	if automaticUnlock {
		if err := appdata.WritePrivateFile(stagedAuto, encodedAuto); err != nil {
			return err
		}
		storedID, storedDEK, err := decodeAutoUnlock(encodedAuto, manager.protector)
		if err != nil || storedID != identifier || subtle.ConstantTimeCompare(storedDEK, dek) != 1 {
			wipe(storedDEK)
			return fmt.Errorf("verify automatic unlock file")
		}
		wipe(storedDEK)
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
	db, err = openAndVerify(ctx, manager.paths.Vault, dek, identifier, manager.logger)
	if err != nil {
		return err
	}
	manager.install(db, dek, identifier, automaticUnlock)
	dek = nil
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
		if nextErr := manager.unlockPasswordCandidate(ctx, manager.paths.VaultKey+".next", password); nextErr == nil {
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
	manager.logger.Info("vault unlocked with master password")
	return nil
}

func (manager *Manager) unlockPasswordCandidate(ctx context.Context, path, password string) error {
	key, err := readKeyFile(path)
	if err != nil {
		return fmt.Errorf("read vault key file: %w", err)
	}
	logging.Diagnostic(manager.logger, "vault key file loaded", "operation", "vault-load", "key_file_version", key.Version)
	dek, err := key.unwrap(password)
	if err != nil {
		return err
	}
	return manager.installUnlocked(ctx, dek, key.VaultID, false)
}

func (manager *Manager) AutoUnlock(ctx context.Context) error {
	logging.Diagnostic(manager.logger, "loading vault", "operation", "vault-load", "file_state", manager.FileState(), "schema_status", "not_loaded")
	if directoryHasEntries(manager.paths.CreateStaging) {
		return ErrIncomplete
	}
	key, keyErr := readKeyFile(manager.paths.VaultKey)
	if keyErr == nil {
		logging.Diagnostic(manager.logger, "vault key file loaded", "operation", "vault-load", "key_file_version", key.Version)
	}
	if !fileExists(manager.paths.Vault) || !fileExists(manager.paths.AutoUnlockKey) {
		if manager.FileState() == FileStateEmpty {
			return ErrNotInitialized
		}
		return ErrLocked
	}
	if errors.Is(keyErr, ErrUnsupportedVersion) {
		return keyErr
	}
	usedNext := false
	err := manager.unlockAutoCandidate(ctx, manager.paths.AutoUnlockKey)
	if err != nil && fileExists(manager.paths.AutoUnlockKey+".next") {
		if nextErr := manager.unlockAutoCandidate(ctx, manager.paths.AutoUnlockKey+".next"); nextErr == nil {
			usedNext = true
			err = nil
		}
	}
	if err != nil {
		return err
	}
	if !usedNext && fileExists(manager.paths.VaultKey+".next") {
		if !fileExists(manager.paths.AutoUnlockKey + ".next") {
			usedNext = true
		} else if data, readErr := readBounded(manager.paths.AutoUnlockKey+".next", autoFileLimit); readErr == nil {
			_, nextDEK, decodeErr := decodeAutoUnlock(data, manager.protector)
			manager.mu.Lock()
			matches := decodeErr == nil && len(nextDEK) == len(manager.dek) && subtle.ConstantTimeCompare(nextDEK, manager.dek) == 1
			manager.mu.Unlock()
			wipe(nextDEK)
			usedNext = matches
		}
	}
	if err := manager.completeUnlock(ctx, usedNext, ""); err != nil {
		return err
	}
	manager.logger.Info("vault unlocked with Windows protection")
	return nil
}

func (manager *Manager) unlockAutoCandidate(ctx context.Context, path string) error {
	data, err := readBounded(path, autoFileLimit)
	if err != nil {
		return err
	}
	identifier, dek, err := decodeAutoUnlock(data, manager.protector)
	if err != nil {
		return err
	}
	logging.Diagnostic(manager.logger, "automatic unlock file loaded", "operation", "vault-load", "auto_unlock_version", autoFormatVersion)
	return manager.installUnlocked(ctx, dek, identifier, true)
}

func (manager *Manager) installUnlocked(ctx context.Context, dek []byte, identifier [vaultIDBytes]byte, automatic bool) error {
	if len(dek) != dekBytes {
		wipe(dek)
		return fmt.Errorf("invalid database key length")
	}
	db, err := openAndVerify(ctx, manager.paths.Vault, dek, identifier, manager.logger)
	if err != nil {
		wipe(dek)
		return err
	}
	manager.mu.Lock()
	if manager.db != nil {
		manager.mu.Unlock()
		_ = db.Close()
		wipe(dek)
		return nil
	}
	manager.db = db
	manager.dek = dek
	manager.vaultID = identifier
	manager.unlockedWithDPAPI = automatic
	manager.closing = false
	manager.mu.Unlock()
	return nil
}

func (manager *Manager) install(db *sql.DB, dek []byte, identifier [vaultIDBytes]byte, automatic bool) {
	manager.mu.Lock()
	manager.db = db
	manager.dek = dek
	manager.vaultID = identifier
	manager.unlockedWithDPAPI = automatic
	manager.closing = false
	manager.mu.Unlock()
}

func (manager *Manager) Lease(_ context.Context) (*sql.DB, func(), error) {
	manager.mu.Lock()
	if manager.db == nil || manager.closing {
		manager.mu.Unlock()
		return nil, nil, ErrLocked
	}
	manager.active++
	db := manager.db
	manager.mu.Unlock()
	var once sync.Once
	release := func() {
		once.Do(func() {
			manager.mu.Lock()
			manager.active--
			if manager.closing && manager.active == 0 && manager.drained != nil {
				close(manager.drained)
				manager.drained = nil
			}
			manager.mu.Unlock()
		})
	}
	return db, release, nil
}

func (manager *Manager) Lock() error { return manager.close(true) }

func (manager *Manager) Close() error { return manager.close(false) }

func (manager *Manager) close(manual bool) error {
	manager.mu.Lock()
	if manager.db == nil {
		manager.mu.Unlock()
		return nil
	}
	manager.closing = true
	var drained <-chan struct{}
	if manager.active > 0 {
		manager.drained = make(chan struct{})
		drained = manager.drained
	}
	manager.mu.Unlock()
	if drained != nil {
		select {
		case <-drained:
		case <-time.After(lockTimeout):
			manager.mu.Lock()
			manager.closing = false
			manager.drained = nil
			manager.mu.Unlock()
			return ErrLockTimeout
		}
	}
	manager.mu.Lock()
	db := manager.db
	dek := manager.dek
	manager.db = nil
	manager.dek = nil
	manager.vaultID = [vaultIDBytes]byte{}
	manager.unlockedWithDPAPI = false
	manager.closing = false
	manager.mu.Unlock()
	if err := db.Close(); err != nil {
		wipe(dek)
		return fmt.Errorf("close vault: %w", err)
	}
	wipe(dek)
	if manual {
		if err := os.Remove(manager.paths.AutoUnlockKey); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove automatic unlock file: %w", err)
		}
	}
	manager.logger.Info("vault locked", "automatic_unlock_revoked", manual)
	return nil
}

func (manager *Manager) TestPassword(ctx context.Context, password string) error {
	key, err := readKeyFile(manager.paths.VaultKey)
	if err != nil {
		return err
	}
	dek, err := key.unwrap(password)
	if err != nil {
		return err
	}
	defer wipe(dek)
	manager.mu.Lock()
	valid := manager.db != nil && key.VaultID == manager.vaultID && len(manager.dek) == len(dek) && subtle.ConstantTimeCompare(manager.dek, dek) == 1
	manager.mu.Unlock()
	if !valid {
		return ErrInvalidPassword
	}
	return nil
}

func (manager *Manager) ResetVault() error {
	if manager.Unlocked() {
		return fmt.Errorf("lock the vault before resetting it")
	}
	for _, path := range []string{manager.paths.Vault, manager.paths.VaultKey, manager.paths.AutoUnlockKey, manager.paths.VaultKey + ".next", manager.paths.AutoUnlockKey + ".next"} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove %q: %w", fileName(path), err)
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

func openAndVerify(ctx context.Context, path string, dek []byte, identifier [vaultIDBytes]byte, logger *slog.Logger) (*sql.DB, error) {
	db, err := openDatabase(ctx, path, dek, logger)
	if err != nil {
		return nil, err
	}
	if err := verifyOpenDatabase(ctx, db, identifier, logger); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func verifyOpenDatabase(ctx context.Context, db *sql.DB, identifier [vaultIDBytes]byte, logger *slog.Logger) error {
	if err := verifySchema(ctx, db, identifier, logger); err != nil {
		return err
	}
	var integrity string
	if err := db.QueryRowContext(ctx, `PRAGMA quick_check`).Scan(&integrity); err != nil || integrity != "ok" {
		return fmt.Errorf("vault integrity check failed")
	}
	return nil
}

func openDatabase(ctx context.Context, path string, dek []byte, logger *slog.Logger) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", databaseDSN(path, dek))
	if err != nil {
		return nil, fmt.Errorf("open encrypted vault: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	closeOnError := func(err error) (*sql.DB, error) {
		_ = db.Close()
		return nil, err
	}
	openCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var cipherVersion string
	if err := db.QueryRowContext(openCtx, `PRAGMA cipher_version`).Scan(&cipherVersion); err != nil || cipherVersion == "" {
		return closeOnError(fmt.Errorf("SQLCipher support is unavailable"))
	}
	var sqliteVersion string
	if err := db.QueryRowContext(openCtx, `SELECT sqlite_version()`).Scan(&sqliteVersion); err != nil {
		return closeOnError(fmt.Errorf("read SQLite version: %w", err))
	}
	logging.Diagnostic(logger, "database engine loaded", "operation", "vault-load", "sqlite_version", sqliteVersion, "sqlcipher_version", cipherVersion)
	if err := db.QueryRowContext(openCtx, `SELECT count(*) FROM sqlite_master`).Scan(new(int)); err != nil {
		return closeOnError(fmt.Errorf("verify encrypted vault key: %w", err))
	}
	var foreignKeys, pageSize, secureDelete int
	var journalMode string
	if err := db.QueryRowContext(openCtx, `PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		return closeOnError(fmt.Errorf("foreign key enforcement is unavailable"))
	}
	if err := db.QueryRowContext(openCtx, `PRAGMA cipher_page_size`).Scan(&pageSize); err != nil || pageSize != 4096 {
		return closeOnError(fmt.Errorf("SQLCipher page size is unsupported"))
	}
	if err := db.QueryRowContext(openCtx, `PRAGMA secure_delete`).Scan(&secureDelete); err != nil || secureDelete != 1 {
		return closeOnError(fmt.Errorf("secure deletion is unavailable"))
	}
	if err := db.QueryRowContext(openCtx, `PRAGMA journal_mode`).Scan(&journalMode); err != nil || !strings.EqualFold(journalMode, "delete") {
		return closeOnError(fmt.Errorf("portable journal mode is unavailable"))
	}
	return db, nil
}

func databaseDSN(path string, dek []byte) string {
	uriPath := filepath.ToSlash(path)
	if filepath.VolumeName(path) != "" && !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	u := &url.URL{Scheme: "file", Path: uriPath}
	query := u.Query()
	query.Set("_pragma_key", "x'"+hex.EncodeToString(dek)+"'")
	query.Set("_pragma_cipher_compatibility", "4")
	query.Set("_pragma_cipher_page_size", "4096")
	query.Set("_pragma_cipher_memory_security", "ON")
	query.Set("_busy_timeout", "5000")
	query.Set("_foreign_keys", "on")
	query.Set("_journal_mode", "DELETE")
	query.Set("_secure_delete", "on")
	u.RawQuery = query.Encode()
	return u.String()
}
