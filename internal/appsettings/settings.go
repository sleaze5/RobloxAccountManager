package appsettings

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
)

const (
	FormatVersion                          = 1
	DefaultPasswordTestIntervalDays        = 30
	DefaultTimestampFormat                 = "MMM D YYYY, h:mm:ss A"
	DefaultTimestampHoverFormat            = "dddd, MMMM DD YYYY, hh:mm:ss A ([relative])"
	DefaultAccountsPresenceIntervalSeconds = 120
	DefaultProfilePresenceIntervalSeconds  = 60
	maxTimestampFormatLength               = 256
)

var loggingLevels = []string{"trace", "debug", "info", "warn", "error"}

type object = map[string]json.RawMessage

var migrations = [...]func(object) (object, error){}

var _ = [1]struct{}{}[len(migrations)-(FormatVersion-1)]

type VaultSettings struct {
	PasswordTestIntervalDays      int    `json:"passwordTestIntervalDays"`
	LastPasswordTestedAtMs        int64  `json:"lastPasswordTestedAtMs,omitempty"`
	PasswordReminderDismissedAtMs int64  `json:"passwordReminderDismissedAtMs,omitempty"`
	LastPasswordTestedAt          string `json:"-"`
}

type Settings struct {
	Version       int                   `json:"version"`
	Presence      PresenceSettings      `json:"presence"`
	Roblox        RobloxSettings        `json:"roblox"`
	Integrations  IntegrationSettings   `json:"integrations"`
	Vault         VaultSettings         `json:"vault"`
	Logging       LoggingSettings       `json:"logging"`
	UserInterface UserInterfaceSettings `json:"userInterface"`
}

type PresenceScope string

const (
	PresenceAccounts PresenceScope = "accounts"
	PresenceProfile  PresenceScope = "profile"
)

type PresenceSettings struct {
	Accounts PresenceUpdateSettings `json:"accounts"`
	Profile  PresenceUpdateSettings `json:"profile"`
}

type PresenceUpdateSettings struct {
	Enabled         bool `json:"enabled"`
	IntervalSeconds int  `json:"intervalSeconds"`
}

type RobloxSettings struct {
	MultiInstance bool   `json:"multiInstance"`
	LinuxClient   string `json:"linuxClient"`
}

type IntegrationSettings struct {
	RoValra       bool   `json:"rovalra"`
	RoValraRegion string `json:"rovalraRegion"`
}

type LoggingSettings struct {
	EnabledLevels map[string]bool `json:"enabledLevels"`
}

type MotionPreference string

const (
	MotionSystem  MotionPreference = "system"
	MotionReduced MotionPreference = "reduced"
	MotionFull    MotionPreference = "full"
)

type UserInterfaceSettings struct {
	TimestampFormat      string           `json:"timestampFormat"`
	TimestampHoverFormat string           `json:"timestampHoverFormat"`
	Motion               MotionPreference `json:"motion"`
}

type Store struct {
	mu             sync.Mutex
	path           string
	settings       Settings
	origin         string
	sourceVersion  int
	replacedFields []string
	recoveryError  error
}

func ReadLogging(path string) (LoggingSettings, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return defaults().Logging, nil
	}
	if err != nil {
		return defaults().Logging, fmt.Errorf("read logging settings: %w", err)
	}
	loaded, err := decode(data)
	if err != nil {
		return defaults().Logging, err
	}
	return loaded.settings.Logging, nil
}

func Open(path string) (*Store, error) {
	loaded, loadErr := load(path)
	store := &Store{
		path:           path,
		settings:       loaded.settings,
		origin:         "loaded",
		sourceVersion:  loaded.sourceVersion,
		replacedFields: loaded.replacedFields,
	}
	if loadErr != nil {
		store.origin = "created"
		if !errors.Is(loadErr, os.ErrNotExist) {
			store.origin = "recovered"
			if err := preserveUnreadableSettings(path); err != nil {
				return nil, errors.Join(loadErr, err)
			}
			store.recoveryError = loadErr
		}
		store.settings = defaults()
	}
	if err := store.saveLocked(); err != nil {
		return nil, err
	}
	return store, nil
}

type loadResult struct {
	settings       Settings
	sourceVersion  int
	replacedFields []string
}

func load(path string) (loadResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return loadResult{}, fmt.Errorf("read settings file: %w", err)
	}
	return decode(data)
}

func decode(data []byte) (loadResult, error) {
	var document object
	if err := decodeJSON(data, &document); err != nil {
		return loadResult{}, fmt.Errorf("decode settings file: %w", err)
	}
	if document == nil {
		return loadResult{}, errors.New("settings file must contain a JSON object")
	}
	raw, ok := document["version"]
	if !ok {
		return loadResult{}, errors.New("settings file has no version")
	}
	var version int
	if err := json.Unmarshal(raw, &version); err != nil || version < 1 || version > FormatVersion {
		return loadResult{}, fmt.Errorf("unsupported settings version %s; supported versions are 1 to %d", raw, FormatVersion)
	}
	for from := version; from < FormatVersion; from++ {
		var err error
		if document, err = migrations[from-1](document); err != nil {
			return loadResult{}, fmt.Errorf("migrate settings from version %d: %w", from, err)
		}
	}
	settings, replaced := decodeCurrent(document)
	return loadResult{settings: settings, sourceVersion: version, replacedFields: replaced}, nil
}

func decodeCurrent(document object) (Settings, []string) {
	fields := &fieldDecoder{}
	settings := defaults()
	presence := fields.object(document, "presence")
	fields.presence(fields.object(presence, "presence.accounts"), "presence.accounts", &settings.Presence.Accounts)
	fields.presence(fields.object(presence, "presence.profile"), "presence.profile", &settings.Presence.Profile)
	roblox := fields.object(document, "roblox")
	decodeField(fields, roblox, "roblox.multiInstance", &settings.Roblox.MultiInstance, nil)
	decodeField(fields, roblox, "roblox.linuxClient", &settings.Roblox.LinuxClient, validLinuxClient)
	integrations := fields.object(document, "integrations")
	decodeField(fields, integrations, "integrations.rovalra", &settings.Integrations.RoValra, nil)
	decodeField(fields, integrations, "integrations.rovalraRegion", &settings.Integrations.RoValraRegion, nil)
	vault := fields.object(document, "vault")
	decodeField(fields, vault, "vault.passwordTestIntervalDays", &settings.Vault.PasswordTestIntervalDays, validPasswordTestInterval)
	decodeField(fields, vault, "vault.lastPasswordTestedAtMs", &settings.Vault.LastPasswordTestedAtMs, validTimestamp)
	decodeField(fields, vault, "vault.passwordReminderDismissedAtMs", &settings.Vault.PasswordReminderDismissedAtMs, validTimestamp)
	levels := fields.object(fields.object(document, "logging"), "logging.enabledLevels")
	for _, level := range loggingLevels {
		enabled := settings.Logging.EnabledLevels[level]
		decodeField(fields, levels, "logging.enabledLevels."+level, &enabled, nil)
		settings.Logging.EnabledLevels[level] = enabled
	}
	userInterface := fields.object(document, "userInterface")
	decodeField(fields, userInterface, "userInterface.timestampFormat", &settings.UserInterface.TimestampFormat, validTimestampFormat)
	decodeField(fields, userInterface, "userInterface.timestampHoverFormat", &settings.UserInterface.TimestampHoverFormat, validTimestampFormat)
	decodeField(fields, userInterface, "userInterface.motion", &settings.UserInterface.Motion, validMotion)
	return settings, fields.replaced
}

type fieldDecoder struct {
	replaced []string
}

func (fields *fieldDecoder) object(document object, path string) object {
	raw, ok := document[fieldName(path)]
	if !ok {
		return nil
	}
	var nested object
	if json.Unmarshal(raw, &nested) != nil || nested == nil {
		fields.replaced = append(fields.replaced, path)
		return nil
	}
	return nested
}

func (fields *fieldDecoder) presence(document object, path string, target *PresenceUpdateSettings) {
	decodeField(fields, document, path+".enabled", &target.Enabled, nil)
	decodeField(fields, document, path+".intervalSeconds", &target.IntervalSeconds, validPresenceInterval)
}

func decodeField[T any](fields *fieldDecoder, document object, path string, target *T, valid func(T) bool) {
	raw, ok := document[fieldName(path)]
	if !ok {
		return
	}
	var value T
	if string(bytes.TrimSpace(raw)) == "null" || json.Unmarshal(raw, &value) != nil || valid != nil && !valid(value) {
		fields.replaced = append(fields.replaced, path)
		return
	}
	*target = value
}

func fieldName(path string) string { return path[strings.LastIndex(path, ".")+1:] }

func defaults() Settings {
	return Settings{
		Version: FormatVersion,
		Presence: PresenceSettings{
			Accounts: PresenceUpdateSettings{Enabled: true, IntervalSeconds: DefaultAccountsPresenceIntervalSeconds},
			Profile:  PresenceUpdateSettings{Enabled: true, IntervalSeconds: DefaultProfilePresenceIntervalSeconds},
		},
		Integrations: IntegrationSettings{RoValra: true},
		Vault:        VaultSettings{PasswordTestIntervalDays: DefaultPasswordTestIntervalDays},
		Logging:      LoggingSettings{EnabledLevels: defaultEnabledLoggingLevels()},
		UserInterface: UserInterfaceSettings{
			TimestampFormat:      DefaultTimestampFormat,
			TimestampHoverFormat: DefaultTimestampHoverFormat,
			Motion:               MotionSystem,
		},
	}
}

func (store *Store) Vault() VaultSettings {
	store.mu.Lock()
	defer store.mu.Unlock()
	settings := store.settings.Vault
	if settings.LastPasswordTestedAtMs != 0 {
		settings.LastPasswordTestedAt = time.UnixMilli(settings.LastPasswordTestedAtMs).UTC().Format(time.RFC3339Nano)
	}
	return settings
}

func (store *Store) Logging() LoggingSettings {
	store.mu.Lock()
	defer store.mu.Unlock()
	settings := store.settings.Logging
	settings.EnabledLevels = cloneLoggingLevels(settings.EnabledLevels)
	return settings
}

func (settings LoggingSettings) EnabledLevelNames() []string {
	levels := make([]string, 0, len(settings.EnabledLevels))
	for _, level := range loggingLevels {
		if settings.EnabledLevels[level] {
			levels = append(levels, level)
		}
	}
	return levels
}

func (store *Store) UserInterface() UserInterfaceSettings {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.settings.UserInterface
}

func (store *Store) Roblox() RobloxSettings {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.settings.Roblox
}

func (store *Store) Integrations() IntegrationSettings {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.settings.Integrations
}

func (store *Store) Presence() PresenceSettings {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.settings.Presence
}

func (store *Store) SetPresenceUpdates(scope PresenceScope, enabled bool, intervalSeconds int) error {
	settings := PresenceUpdateSettings{Enabled: enabled, IntervalSeconds: intervalSeconds}
	if err := validatePresenceSettings(settings); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	previous := store.settings.Presence
	switch scope {
	case PresenceAccounts:
		store.settings.Presence.Accounts = settings
	case PresenceProfile:
		store.settings.Presence.Profile = settings
	default:
		return fmt.Errorf("unknown presence update scope %q", scope)
	}
	if err := store.saveLocked(); err != nil {
		store.settings.Presence = previous
		return err
	}
	return nil
}

func (store *Store) SetMultiInstance(enabled bool) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	previous := store.settings.Roblox
	store.settings.Roblox.MultiInstance = enabled
	if err := store.saveLocked(); err != nil {
		store.settings.Roblox = previous
		return err
	}
	return nil
}

func (store *Store) SetLinuxClient(client string) error {
	if !validLinuxClient(client) {
		return fmt.Errorf("unknown linux client %q", client)
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	previous := store.settings.Roblox
	store.settings.Roblox.LinuxClient = client
	if err := store.saveLocked(); err != nil {
		store.settings.Roblox = previous
		return err
	}
	return nil
}

func (store *Store) SetRoValraEnabled(enabled bool) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	previous := store.settings.Integrations
	store.settings.Integrations.RoValra = enabled
	if err := store.saveLocked(); err != nil {
		store.settings.Integrations = previous
		return err
	}
	return nil
}

func (store *Store) SetRoValraRegion(region string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	previous := store.settings.Integrations
	store.settings.Integrations.RoValraRegion = region
	if err := store.saveLocked(); err != nil {
		store.settings.Integrations = previous
		return err
	}
	return nil
}

func (store *Store) SetLoggingLevelEnabled(level string, enabled bool) error {
	if !validLoggingLevel(level) {
		return fmt.Errorf("unknown logging level %q", level)
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	previous := cloneLoggingLevels(store.settings.Logging.EnabledLevels)
	store.settings.Logging.EnabledLevels[level] = enabled
	if err := store.saveLocked(); err != nil {
		store.settings.Logging.EnabledLevels = previous
		return err
	}
	return nil
}

func (store *Store) SetAllLoggingLevelsEnabled(enabled bool) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	previous := cloneLoggingLevels(store.settings.Logging.EnabledLevels)
	for _, level := range loggingLevels {
		store.settings.Logging.EnabledLevels[level] = enabled
	}
	if err := store.saveLocked(); err != nil {
		store.settings.Logging.EnabledLevels = previous
		return err
	}
	return nil
}

func (store *Store) SetTimestampFormats(timestampFormat, timestampHoverFormat string) error {
	settings := UserInterfaceSettings{
		TimestampFormat:      timestampFormat,
		TimestampHoverFormat: timestampHoverFormat,
	}
	if err := validateUserInterfaceSettings(settings); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	previous := store.settings.UserInterface
	store.settings.UserInterface.TimestampFormat = settings.TimestampFormat
	store.settings.UserInterface.TimestampHoverFormat = settings.TimestampHoverFormat
	if err := store.saveLocked(); err != nil {
		store.settings.UserInterface = previous
		return err
	}
	return nil
}

func (store *Store) SetMotion(motion MotionPreference) error {
	if !validMotion(motion) {
		return fmt.Errorf("unknown motion preference %q", motion)
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	previous := store.settings.UserInterface
	store.settings.UserInterface.Motion = motion
	if err := store.saveLocked(); err != nil {
		store.settings.UserInterface = previous
		return err
	}
	return nil
}

func (store *Store) SetPasswordTestIntervalDays(days int) error {
	if !validPasswordTestInterval(days) {
		return fmt.Errorf("password test interval must be 0, 7, 30, or 90 days")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	previous := store.settings.Vault.PasswordTestIntervalDays
	store.settings.Vault.PasswordTestIntervalDays = days
	if err := store.saveLocked(); err != nil {
		store.settings.Vault.PasswordTestIntervalDays = previous
		return err
	}
	return nil
}

func (store *Store) RecordPasswordTested(at time.Time) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	previous := store.settings.Vault
	store.settings.Vault.LastPasswordTestedAtMs = at.UTC().UnixMilli()
	store.settings.Vault.PasswordReminderDismissedAtMs = 0
	if err := store.saveLocked(); err != nil {
		store.settings.Vault = previous
		return err
	}
	return nil
}

func (store *Store) DismissPasswordReminder(at time.Time) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	previous := store.settings.Vault.PasswordReminderDismissedAtMs
	store.settings.Vault.PasswordReminderDismissedAtMs = at.UTC().UnixMilli()
	if err := store.saveLocked(); err != nil {
		store.settings.Vault.PasswordReminderDismissedAtMs = previous
		return err
	}
	return nil
}

func (store *Store) PasswordReminderDue(at time.Time) bool {
	settings := store.Vault()
	if settings.PasswordTestIntervalDays == 0 {
		return false
	}
	latest := latestTime(settings.LastPasswordTestedAtMs, settings.PasswordReminderDismissedAtMs)
	return latest.IsZero() || !at.Before(latest.Add(time.Duration(settings.PasswordTestIntervalDays)*24*time.Hour))
}

func preserveUnreadableSettings(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect unreadable settings: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("cannot back up settings: settings.json is not a regular file")
	}
	backup := path + ".bak"
	if err := archiveSettingsBackup(backup); err != nil {
		return err
	}
	if err := appdata.ReplaceFile(path, backup); err != nil {
		return fmt.Errorf("preserve unreadable settings as settings.json.bak: %w", err)
	}
	return appdata.RestrictFile(backup)
}

func archiveSettingsBackup(backup string) error {
	info, err := os.Lstat(backup)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect previous settings backup: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("cannot preserve previous settings backup: settings.json.bak is not a regular file")
	}
	pattern := fmt.Sprintf("%s.%d.*", filepath.Base(backup), time.Now().UnixMilli())
	archive, err := os.CreateTemp(filepath.Dir(backup), pattern)
	if err != nil {
		return fmt.Errorf("reserve previous settings backup: %w", err)
	}
	if err := archive.Close(); err != nil {
		_ = os.Remove(archive.Name())
		return err
	}
	if err := appdata.ReplaceFile(backup, archive.Name()); err != nil {
		_ = os.Remove(archive.Name())
		return fmt.Errorf("archive previous settings backup: %w", err)
	}
	return nil
}

func (store *Store) saveLocked() error {
	data, err := json.MarshalIndent(store.settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings file: %w", err)
	}
	if err := appdata.WritePrivateFile(store.path, append(data, '\n')); err != nil {
		return fmt.Errorf("save settings file: %w", err)
	}
	return nil
}

func decodeJSON(data []byte, value any) error {
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	if !utf8.Valid(data) {
		return errors.New("settings must use valid UTF-8 encoding")
	}
	return json.Unmarshal(data, value)
}

func (store *Store) SourceVersion() int { return store.sourceVersion }

func (store *Store) ReplacedFields() []string { return store.replacedFields }

func (store *Store) RecoveryError() error { return store.recoveryError }

func (store *Store) Origin() string { return store.origin }

func validatePresenceSettings(settings PresenceUpdateSettings) error {
	if !validPresenceInterval(settings.IntervalSeconds) {
		return fmt.Errorf("presence update interval must be between 30 and 900 seconds in steps of 5")
	}
	return nil
}

func validPresenceInterval(seconds int) bool {
	return seconds >= 30 && seconds <= 900 && seconds%5 == 0
}

func validateUserInterfaceSettings(settings UserInterfaceSettings) error {
	for name, value := range map[string]string{
		"timestampFormat":      settings.TimestampFormat,
		"timestampHoverFormat": settings.TimestampHoverFormat,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must not be empty", name)
		}
		if utf8.RuneCountInString(value) > maxTimestampFormatLength {
			return fmt.Errorf("%s must not exceed %d characters", name, maxTimestampFormatLength)
		}
	}
	return nil
}

func validTimestampFormat(format string) bool {
	return validateUserInterfaceSettings(UserInterfaceSettings{TimestampFormat: format, TimestampHoverFormat: format}) == nil
}

func validTimestamp(milliseconds int64) bool { return milliseconds >= 0 }

func validLinuxClient(client string) bool {
	switch client {
	case "", "sober", "mocktail":
		return true
	default:
		return false
	}
}

func validMotion(motion MotionPreference) bool {
	return motion == MotionSystem || motion == MotionReduced || motion == MotionFull
}

func validPasswordTestInterval(days int) bool {
	return days == 0 || days == 7 || days == 30 || days == 90
}

func validLoggingLevel(level string) bool {
	for _, value := range loggingLevels {
		if level == value {
			return true
		}
	}
	return false
}

func defaultEnabledLoggingLevels() map[string]bool {
	levels := make(map[string]bool, len(loggingLevels))
	for _, level := range loggingLevels {
		levels[level] = level == "warn" || level == "error"
	}
	return levels
}

func cloneLoggingLevels(levels map[string]bool) map[string]bool {
	cloned := make(map[string]bool, len(levels))
	for level, enabled := range levels {
		cloned[level] = enabled
	}
	return cloned
}

func latestTime(values ...int64) time.Time {
	var latest time.Time
	for _, value := range values {
		if value == 0 {
			continue
		}
		parsed := time.UnixMilli(value).UTC()
		if parsed.After(latest) {
			latest = parsed
		}
	}
	return latest
}
