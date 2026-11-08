package appsettings

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
	"github.com/sleaze5/RobloxAccountManager/internal/timestampformat"
)

const (
	FormatVersion                          = 2
	DefaultPasswordTestIntervalDays        = 30
	DefaultTimestampFormat                 = "$MMM $D $YYYY, $h:$mm:$ss $A"
	DefaultTimestampHoverFormat            = "$dddd, $MMMM $DD $YYYY, $hh:$mm:$ss $A ($relative)"
	DefaultAccountsPresenceIntervalSeconds = 300
	DefaultProfilePresenceIntervalSeconds  = 60
	DefaultSocialPresenceIntervalSeconds   = 60
)

var loggingLevels = []string{"trace", "debug", "info", "warn", "error"}

type object = map[string]json.RawMessage

var migrations = [...]func(object) (object, error){resetTimestampFormats}

var _ = [1]struct{}{}[len(migrations)-(FormatVersion-1)]

// resetTimestampFormats removes the version 1 timestamp formats, whose syntax
// version 2 no longer reads, so decoding applies the version 2 defaults.
func resetTimestampFormats(document object) (object, error) {
	document["version"] = json.RawMessage("2")
	userInterface := (&fieldDecoder{}).object(document, "userInterface")
	if userInterface == nil {
		return document, nil
	}
	delete(userInterface, "timestampFormat")
	delete(userInterface, "timestampHoverFormat")
	encoded, err := json.Marshal(userInterface)
	if err != nil {
		return nil, fmt.Errorf("encode user interface settings: %w", err)
	}
	document["userInterface"] = encoded
	return document, nil
}

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
	PresenceSocial   PresenceScope = "social"
)

type PresenceSettings struct {
	Accounts PresenceUpdateSettings `json:"accounts"`
	Profile  PresenceUpdateSettings `json:"profile"`
	Social   PresenceUpdateSettings `json:"social"`
}

type PresenceUpdateSettings struct {
	Enabled         bool `json:"enabled"`
	IntervalSeconds int  `json:"intervalSeconds"`
}

type RobloxSettings struct {
	MultiInstance bool   `json:"multiInstance"`
	LinuxClient   string `json:"linuxClient"`
	PlayerPath    string `json:"playerPath"`
}

type IntegrationSettings struct {
	RoValra       bool   `json:"rovalra"`
	RoValraRegion string `json:"rovalraRegion"`
	RBXServers    bool   `json:"rbxservers"`
}

type LoggingSettings struct {
	EnabledLevels map[string]bool `json:"enabledLevels"`
}

type Section string

const (
	SectionPresence      Section = "presence"
	SectionRoblox        Section = "roblox"
	SectionIntegrations  Section = "integrations"
	SectionUserInterface Section = "userInterface"
	SectionLogging       Section = "logging"
	SectionVault         Section = "vault"
)

var Sections = []Section{SectionPresence, SectionRoblox, SectionIntegrations, SectionUserInterface, SectionLogging, SectionVault}

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
	mutex          sync.Mutex
	path           string
	settings       Settings
	origin         string
	sourceVersion  int
	replacedFields []string
	recoveryError  error
}

func ReadLogging(path string) (LoggingSettings, error) {
	settingsJSON, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return defaults().Logging, nil
	}
	if err != nil {
		return defaults().Logging, fmt.Errorf("read logging settings: %w", err)
	}
	loaded, err := decode(settingsJSON)
	if err != nil {
		return defaults().Logging, err
	}
	return loaded.settings.Logging, nil
}

// New returns default settings without writing them. The first change saves the file.
func New(path string) *Store {
	return &Store{path: path, settings: defaults(), origin: "created"}
}

func Open(path string) (*Store, error) {
	loaded, loadError := load(path)
	store := &Store{
		path:           path,
		settings:       loaded.settings,
		origin:         "loaded",
		sourceVersion:  loaded.sourceVersion,
		replacedFields: loaded.replacedFields,
	}
	if loadError != nil {
		store.origin = "created"
		if !errors.Is(loadError, os.ErrNotExist) {
			store.origin = "recovered"
			if err := preserveUnreadableSettings(path); err != nil {
				return nil, errors.Join(loadError, err)
			}
			store.recoveryError = loadError
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
	settingsJSON, err := os.ReadFile(path)
	if err != nil {
		return loadResult{}, fmt.Errorf("read settings file: %w", err)
	}
	return decode(settingsJSON)
}

func decode(settingsJSON []byte) (loadResult, error) {
	var document object
	if err := decodeJSON(settingsJSON, &document); err != nil {
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

func Decode(settingsJSON []byte) (Settings, error) {
	loaded, err := decode(settingsJSON)
	if err != nil {
		return Settings{}, err
	}
	return loaded.settings, nil
}

// Export leaves out the Roblox Player path and the vault timestamps, because they belong to this computer and vault.
// Options for other operating systems stay in the export. They are saved on import and have no effect on other systems.
func (store *Store) Export() ([]byte, error) {
	store.mutex.Lock()
	settings := store.settings
	settings.Logging.EnabledLevels = cloneLoggingLevels(settings.Logging.EnabledLevels)
	store.mutex.Unlock()
	settings.Roblox.PlayerPath = ""
	settings.Vault = VaultSettings{PasswordTestIntervalDays: settings.Vault.PasswordTestIntervalDays}
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("encode exported settings: %w", err)
	}
	return settingsJSON, nil
}

func (store *Store) Import(imported Settings, sections []Section) error {
	return store.update(func(settings *Settings) error {
		for _, section := range sections {
			switch section {
			case SectionPresence:
				settings.Presence = imported.Presence
			case SectionRoblox:
				settings.Roblox.MultiInstance = imported.Roblox.MultiInstance
				settings.Roblox.LinuxClient = imported.Roblox.LinuxClient
			case SectionIntegrations:
				settings.Integrations = imported.Integrations
			case SectionUserInterface:
				settings.UserInterface = imported.UserInterface
			case SectionLogging:
				settings.Logging.EnabledLevels = cloneLoggingLevels(imported.Logging.EnabledLevels)
			case SectionVault:
				settings.Vault.PasswordTestIntervalDays = imported.Vault.PasswordTestIntervalDays
			default:
				return fmt.Errorf("unknown settings section %q", section)
			}
		}
		return nil
	})
}

func decodeCurrent(document object) (Settings, []string) {
	fields := &fieldDecoder{}
	settings := defaults()
	presence := fields.object(document, "presence")
	fields.presence(fields.object(presence, "presence.accounts"), "presence.accounts", &settings.Presence.Accounts)
	fields.presence(fields.object(presence, "presence.profile"), "presence.profile", &settings.Presence.Profile)
	fields.presence(fields.object(presence, "presence.social"), "presence.social", &settings.Presence.Social)
	roblox := fields.object(document, "roblox")
	decodeField(fields, roblox, "roblox.multiInstance", &settings.Roblox.MultiInstance, nil)
	decodeField(fields, roblox, "roblox.linuxClient", &settings.Roblox.LinuxClient, validLinuxClient)
	decodeField(fields, roblox, "roblox.playerPath", &settings.Roblox.PlayerPath, validPlayerPath)
	integrations := fields.object(document, "integrations")
	decodeField(fields, integrations, "integrations.rovalra", &settings.Integrations.RoValra, nil)
	decodeField(fields, integrations, "integrations.rovalraRegion", &settings.Integrations.RoValraRegion, nil)
	decodeField(fields, integrations, "integrations.rbxservers", &settings.Integrations.RBXServers, nil)
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

func decodeField[FieldValue any](fields *fieldDecoder, document object, path string, target *FieldValue, valid func(FieldValue) bool) {
	raw, ok := document[fieldName(path)]
	if !ok {
		return
	}
	var value FieldValue
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
			Social:   PresenceUpdateSettings{Enabled: true, IntervalSeconds: DefaultSocialPresenceIntervalSeconds},
		},
		Integrations: IntegrationSettings{RoValra: true, RBXServers: true},
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
	store.mutex.Lock()
	defer store.mutex.Unlock()
	settings := store.settings.Vault
	if settings.LastPasswordTestedAtMs != 0 {
		settings.LastPasswordTestedAt = time.UnixMilli(settings.LastPasswordTestedAtMs).UTC().Format(time.RFC3339Nano)
	}
	return settings
}

func (store *Store) Logging() LoggingSettings {
	store.mutex.Lock()
	defer store.mutex.Unlock()
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
	store.mutex.Lock()
	defer store.mutex.Unlock()
	return store.settings.UserInterface
}

func (store *Store) Roblox() RobloxSettings {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	return store.settings.Roblox
}

func (store *Store) Integrations() IntegrationSettings {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	return store.settings.Integrations
}

func (store *Store) Presence() PresenceSettings {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	return store.settings.Presence
}

func (store *Store) SetPresenceUpdates(scope PresenceScope, enabled bool, intervalSeconds int) error {
	presence := PresenceUpdateSettings{Enabled: enabled, IntervalSeconds: intervalSeconds}
	if err := validatePresenceSettings(presence); err != nil {
		return err
	}
	return store.update(func(settings *Settings) error {
		switch scope {
		case PresenceAccounts:
			settings.Presence.Accounts = presence
		case PresenceProfile:
			settings.Presence.Profile = presence
		case PresenceSocial:
			settings.Presence.Social = presence
		default:
			return fmt.Errorf("unknown presence update scope %q", scope)
		}
		return nil
	})
}

func (store *Store) SetMultiInstance(enabled bool) error {
	return store.set(func(settings *Settings) { settings.Roblox.MultiInstance = enabled })
}

func (store *Store) SetLinuxClient(client string) error {
	if !validLinuxClient(client) {
		return fmt.Errorf("unknown linux client %q", client)
	}
	return store.set(func(settings *Settings) { settings.Roblox.LinuxClient = client })
}

func (store *Store) SetPlayerPath(path string) error {
	if !validPlayerPath(path) {
		return errors.New("the Roblox Player path is not valid UTF-8 or is longer than 4096 bytes")
	}
	return store.set(func(settings *Settings) { settings.Roblox.PlayerPath = path })
}

func (store *Store) SetRoValraEnabled(enabled bool) error {
	return store.set(func(settings *Settings) { settings.Integrations.RoValra = enabled })
}

func (store *Store) SetRBXServersEnabled(enabled bool) error {
	return store.set(func(settings *Settings) { settings.Integrations.RBXServers = enabled })
}

func (store *Store) SetRoValraRegion(region string) error {
	return store.set(func(settings *Settings) { settings.Integrations.RoValraRegion = region })
}

func (store *Store) SetLoggingLevelEnabled(level string, enabled bool) error {
	if !validLoggingLevel(level) {
		return fmt.Errorf("unknown logging level %q", level)
	}
	return store.set(func(settings *Settings) { settings.Logging.EnabledLevels[level] = enabled })
}

func (store *Store) SetAllLoggingLevelsEnabled(enabled bool) error {
	return store.set(func(settings *Settings) {
		for _, level := range loggingLevels {
			settings.Logging.EnabledLevels[level] = enabled
		}
	})
}

func (store *Store) SetTimestampFormats(timestampFormat, timestampHoverFormat string) error {
	if err := validateUserInterfaceSettings(UserInterfaceSettings{TimestampFormat: timestampFormat, TimestampHoverFormat: timestampHoverFormat}); err != nil {
		return err
	}
	return store.set(func(settings *Settings) {
		settings.UserInterface.TimestampFormat = timestampFormat
		settings.UserInterface.TimestampHoverFormat = timestampHoverFormat
	})
}

func (store *Store) SetMotion(motion MotionPreference) error {
	if !validMotion(motion) {
		return fmt.Errorf("unknown motion preference %q", motion)
	}
	return store.set(func(settings *Settings) { settings.UserInterface.Motion = motion })
}

func (store *Store) SetPasswordTestIntervalDays(days int) error {
	if !validPasswordTestInterval(days) {
		return errors.New("password test interval must be 0, 7, 30, or 90 days")
	}
	return store.set(func(settings *Settings) { settings.Vault.PasswordTestIntervalDays = days })
}

func (store *Store) RecordPasswordTested(at time.Time) error {
	return store.set(func(settings *Settings) {
		settings.Vault.LastPasswordTestedAtMs = at.UTC().UnixMilli()
		settings.Vault.PasswordReminderDismissedAtMs = 0
	})
}

func (store *Store) DismissPasswordReminder(at time.Time) error {
	return store.set(func(settings *Settings) { settings.Vault.PasswordReminderDismissedAtMs = at.UTC().UnixMilli() })
}

func (store *Store) set(change func(*Settings)) error {
	return store.update(func(settings *Settings) error {
		change(settings)
		return nil
	})
}

// update saves the changed settings, or keeps the previous settings if the
// change or the save fails.
func (store *Store) update(change func(*Settings) error) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	previous := store.settings
	previous.Logging.EnabledLevels = cloneLoggingLevels(previous.Logging.EnabledLevels)
	if err := change(&store.settings); err != nil {
		store.settings = previous
		return err
	}
	if err := store.saveLocked(); err != nil {
		store.settings = previous
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
	settingsStatus, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect unreadable settings: %w", err)
	}
	if !settingsStatus.Mode().IsRegular() {
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
	backupStatus, err := os.Lstat(backup)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect previous settings backup: %w", err)
	}
	if !backupStatus.Mode().IsRegular() {
		return errors.New("cannot preserve previous settings backup: settings.json.bak is not a regular file")
	}
	pattern := fmt.Sprintf("%s.%d.*", filepath.Base(backup), time.Now().UnixMilli())
	archive, err := os.CreateTemp(filepath.Dir(backup), pattern)
	if err != nil {
		return fmt.Errorf("reserve previous settings backup: %w", err)
	}
	if err := archive.Close(); err != nil {
		_ = os.Remove(archive.Name())
		return fmt.Errorf("reserve previous settings backup: %w", err)
	}
	if err := appdata.ReplaceFile(backup, archive.Name()); err != nil {
		_ = os.Remove(archive.Name())
		return fmt.Errorf("archive previous settings backup: %w", err)
	}
	return nil
}

func (store *Store) saveLocked() error {
	settingsJSON, err := json.MarshalIndent(store.settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings file: %w", err)
	}
	if err := appdata.WritePrivateFile(store.path, append(settingsJSON, '\n')); err != nil {
		return fmt.Errorf("save settings file: %w", err)
	}
	return nil
}

var utf8BOM = []byte{0xef, 0xbb, 0xbf}

func decodeJSON(encoded []byte, value any) error {
	encoded = bytes.TrimPrefix(encoded, utf8BOM)
	if !utf8.Valid(encoded) {
		return errors.New("settings must use valid UTF-8 encoding")
	}
	if err := json.Unmarshal(encoded, value); err != nil {
		return fmt.Errorf("parse JSON: %w", err)
	}
	return nil
}

func (store *Store) SourceVersion() int { return store.sourceVersion }

func (store *Store) ReplacedFields() []string { return store.replacedFields }

func (store *Store) RecoveryError() error { return store.recoveryError }

func (store *Store) Origin() string { return store.origin }

func validatePresenceSettings(settings PresenceUpdateSettings) error {
	if !validPresenceInterval(settings.IntervalSeconds) {
		return errors.New("presence update interval must be between 30 and 900 seconds in steps of 5")
	}
	return nil
}

func validPresenceInterval(seconds int) bool {
	return seconds >= 30 && seconds <= 900 && seconds%5 == 0
}

func validateUserInterfaceSettings(settings UserInterfaceSettings) error {
	if _, err := timestampformat.Parse(settings.TimestampFormat); err != nil {
		return fmt.Errorf("timestampFormat: %w", err)
	}
	if _, err := timestampformat.Parse(settings.TimestampHoverFormat); err != nil {
		return fmt.Errorf("timestampHoverFormat: %w", err)
	}
	return nil
}

func validTimestampFormat(format string) bool {
	_, err := timestampformat.Parse(format)
	return err == nil
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

func validPlayerPath(path string) bool {
	return len(path) <= 4096 && utf8.ValidString(path)
}

func validMotion(motion MotionPreference) bool {
	return motion == MotionSystem || motion == MotionReduced || motion == MotionFull
}

func validPasswordTestInterval(days int) bool {
	// 0 turns the reminder off.
	return days == 0 || days == 7 || days == 30 || days == 90
}

func validLoggingLevel(level string) bool {
	return slices.Contains(loggingLevels, level)
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
	maps.Copy(cloned, levels)
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
