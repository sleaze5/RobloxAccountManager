package appservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

var accountSettingsOperationID atomic.Uint64

type accountSettingsReads struct {
	settings, options, voice          robloxservices.SettingsDocument
	settingsErr, optionsErr, voiceErr error
	optionsV2                         robloxservices.SettingsDocument
	optionsV2Err                      error
}

type accountSettingsLock struct {
	gate  chan struct{}
	users int
}

func (service *Service) GetAccountSettings(ctx context.Context, accountID int64) (AccountSettingsSnapshot, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "account-settings")
	if err != nil {
		return AccountSettingsSnapshot{}, err
	}
	defer done()
	logger := service.logger.With("operation_id", fmt.Sprintf("settings-%d", accountSettingsOperationID.Add(1)), "operation", "read")
	return service.readAccountSettings(ctx, accountID, version, logger)
}

// beginAccountSession pins a multi-request account workflow to the current session.
func (service *Service) beginAccountSession(parent context.Context, accountID int64, endpoint string) (context.Context, int64, func(), error) {
	if accountID <= 0 {
		return nil, 0, nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Select an account first."}
	}
	if !service.vault.Unlocked() {
		return nil, 0, nil, &roblox.Error{Kind: roblox.KindVaultLocked, Endpoint: endpoint, Message: "The account vault is locked."}
	}
	snapshot, err := service.sessions.Snapshot(accountID)
	if err != nil {
		return nil, 0, nil, err
	}
	ctx, cancel := context.WithTimeout(parent, 75*time.Second)
	stop := context.AfterFunc(snapshot.Context, cancel)
	if snapshot.Context.Err() != nil {
		cancel()
	}
	return ctx, snapshot.SecretVersion, func() { stop(); cancel() }, nil
}

func (service *Service) readAccountSettings(ctx context.Context, accountID, version int64, logger *slog.Logger) (AccountSettingsSnapshot, error) {
	started := time.Now()
	var reads accountSettingsReads
	var group sync.WaitGroup
	group.Add(4)
	go func() {
		defer group.Done()
		reads.settings, reads.settingsErr = service.accountSettings.Read(ctx, accountID, version)
	}()
	go func() {
		defer group.Done()
		reads.options, reads.optionsErr = service.accountSettings.ReadOptions(ctx, accountID, version)
	}()
	go func() {
		defer group.Done()
		reads.voice, reads.voiceErr = service.accountSettings.ReadVoice(ctx, accountID, version)
	}()
	go func() {
		defer group.Done()
		reads.optionsV2, reads.optionsV2Err = service.accountSettings.ReadOptionsV2(ctx, accountID, version)
	}()
	group.Wait()
	if ctx.Err() != nil || !service.sessions.VersionCurrent(accountID, version) {
		return AccountSettingsSnapshot{}, accountSettingsError(roblox.KindCancelled, "The settings request ended or the account session changed. Refresh settings to continue.")
	}
	snapshot := AccountSettingsSnapshot{
		AccountID: accountID, FetchedAtMs: time.Now().UnixMilli(),
		Settings: []AccountSettingView{}, SectionErrors: []AccountSettingsSectionError{},
	}
	for _, section := range []struct {
		source string
		err    error
	}{
		{"settings", reads.settingsErr}, {"options", reads.optionsErr}, {"communication", reads.optionsV2Err}, {"voice", reads.voiceErr},
	} {
		if section.err != nil {
			failure := accountSettingsSectionError(section.source, section.err)
			snapshot.SectionErrors = append(snapshot.SectionErrors, failure)
			status := 0
			code := 0
			var remote *roblox.Error
			if errors.As(section.err, &remote) {
				status = remote.Status
				code = remote.RobloxCode
			}
			logger.Warn("account settings section unavailable", "source", section.source, "error_kind", failure.Kind, "status", status, "roblox_code", code)
		}
	}
	for _, definition := range accountSettingDefinitions {
		if view, present := normalizeAccountSetting(definition, reads); present {
			snapshot.Settings = append(snapshot.Settings, view)
		}
	}
	applyAccountSettingDependencies(&snapshot)
	logger.Debug("account settings read finished", "duration_ms", time.Since(started).Milliseconds(), "failed_sections", len(snapshot.SectionErrors))
	return snapshot, nil
}

func accountSettingsSectionError(source string, err error) AccountSettingsSectionError {
	failure := AccountSettingsSectionError{Source: source, Kind: string(roblox.KindProtocol), Message: "Could not load these settings. Refresh settings to retry."}
	var remote *roblox.Error
	if errors.As(err, &remote) {
		failure.Kind = string(remote.Kind)
		if remote.Message != "" {
			failure.Message = remote.Message
		}
	}
	return failure
}

func accountSettingsError(kind roblox.ErrorKind, message string) error {
	return &roblox.Error{Kind: kind, Endpoint: "account-settings", Message: message}
}

func (service *Service) lockAccountSettings(ctx context.Context, accountID int64) (func(), error) {
	service.accountSettingsMu.Lock()
	lock := service.accountSettingsLocks[accountID]
	if lock == nil {
		lock = &accountSettingsLock{gate: make(chan struct{}, 1)}
		service.accountSettingsLocks[accountID] = lock
	}
	lock.users++
	service.accountSettingsMu.Unlock()
	drop := func() {
		service.accountSettingsMu.Lock()
		lock.users--
		if lock.users == 0 {
			delete(service.accountSettingsLocks, accountID)
		}
		service.accountSettingsMu.Unlock()
	}
	select {
	case lock.gate <- struct{}{}:
		return func() { <-lock.gate; drop() }, nil
	case <-ctx.Done():
		drop()
		return nil, accountSettingsError(roblox.KindCancelled, "The settings request ended before saving.")
	}
}

func validateAccountSettingChange(change AccountSettingChange) error {
	definition, found := accountSettingDefinitionFor(change.Key)
	if !found {
		return accountSettingsError(roblox.KindProtocol, "This account setting is not supported.")
	}
	if definition.boolean {
		if change.BoolValue == nil || change.ExpectedBoolValue == nil || change.StringValue != nil || change.ExpectedStringValue != nil {
			return accountSettingsError(roblox.KindProtocol, "This setting requires a boolean value and its last confirmed value.")
		}
	} else if change.StringValue == nil || change.ExpectedStringValue == nil || change.BoolValue != nil || change.ExpectedBoolValue != nil {
		return accountSettingsError(roblox.KindProtocol, "This setting requires a string value and its last confirmed value.")
	}
	if !definition.verifiedWrite {
		return accountSettingsError(roblox.KindForbidden, unverifiedSettingReason)
	}
	if change.StringValue != nil && !supportedSettingToken(change.Key, *change.StringValue) {
		return accountSettingsError(roblox.KindProtocol, "This value is not supported for this setting.")
	}

	return nil
}

func (service *Service) UpdateAccountSetting(ctx context.Context, accountID int64, change AccountSettingChange) (AccountSettingUpdateResult, error) {
	if err := validateAccountSettingChange(change); err != nil {
		return AccountSettingUpdateResult{}, err
	}
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "account-settings")
	if err != nil {
		return AccountSettingUpdateResult{}, err
	}
	defer done()
	unlock, err := service.lockAccountSettings(ctx, accountID)
	if err != nil {
		return AccountSettingUpdateResult{}, err
	}
	defer unlock()
	logger := service.logger.With("operation_id", fmt.Sprintf("settings-%d", accountSettingsOperationID.Add(1)), "operation", "update", "setting_key", change.Key)
	before, err := service.readAccountSettings(ctx, accountID, version, logger)
	if err != nil {
		return AccountSettingUpdateResult{}, err
	}
	current := findAccountSetting(before, change.Key)
	expected := AccountSettingView{ValueState: "known", StringValue: change.ExpectedStringValue, BoolValue: change.ExpectedBoolValue}
	if current.UnavailableReason == conflictingSettingReason || !sameSettingValue(current, expected) {
		logger.Info("account setting update finished", "outcome", "conflict")
		return AccountSettingUpdateResult{Snapshot: before, Outcome: "conflict", Message: "This setting changed on Roblox. Review the current value before saving again."}, nil
	}
	desired := AccountSettingView{ValueState: "known", StringValue: change.StringValue, BoolValue: change.BoolValue}
	if sameSettingValue(current, desired) {
		logger.Info("account setting update finished", "outcome", "unchanged")
		return AccountSettingUpdateResult{Snapshot: before, Outcome: "unchanged", Message: "This value is already applied."}, nil
	}
	if !current.Editable || !accountSettingAllows(current, desired) {
		return AccountSettingUpdateResult{}, accountSettingsError(roblox.KindForbidden, "Roblox does not allow this change. Refresh settings to review available choices.")
	}
	if ctx.Err() != nil || !service.sessions.VersionCurrent(accountID, version) {
		return AccountSettingUpdateResult{}, accountSettingsError(roblox.KindCancelled, "The account session changed before saving. Refresh settings to continue.")
	}
	var writeErr error
	if change.BoolValue != nil {
		writeErr = service.accountSettings.UpdateBool(ctx, accountID, version, string(change.Key), *change.BoolValue)
	} else {
		writeErr = service.accountSettings.Update(ctx, accountID, version, string(change.Key), *change.StringValue)
	}
	return service.confirmAccountSetting(ctx, accountID, version, before, change.Key, desired, writeErr, logger)
}

func findAccountSetting(snapshot AccountSettingsSnapshot, key AccountSettingKey) AccountSettingView {
	for _, view := range snapshot.Settings {
		if view.Key == key {
			return view
		}
	}
	return AccountSettingView{Key: key, ValueState: "unavailable"}
}

func accountSettingAllows(current, desired AccountSettingView) bool {
	for _, option := range current.Options {
		if option.Enabled && sameSettingValue(AccountSettingView{ValueState: "known", StringValue: option.StringValue, BoolValue: option.BoolValue}, desired) {
			return true
		}
	}
	return false
}

func (service *Service) confirmAccountSetting(ctx context.Context, accountID, version int64, before AccountSettingsSnapshot, key AccountSettingKey, desired AccountSettingView, writeErr error, logger *slog.Logger) (AccountSettingUpdateResult, error) {
	after, readErr := service.readAccountSettings(ctx, accountID, version, logger)
	if readErr == nil && settingsReadBackConfirmed(after, key, desired) {
		logger.Info("account setting update finished", "outcome", "applied")
		return AccountSettingUpdateResult{Snapshot: after, Outcome: "applied", Message: "Saved"}, nil
	}
	if writeErr != nil {
		failure := accountSettingsSectionError("settings", writeErr)
		logger.Warn("account setting write not confirmed", "error_kind", failure.Kind)
		if definiteSettingRejection(writeErr) {
			return AccountSettingUpdateResult{}, writeErr
		}
	}
	logger.Warn("account setting update finished", "outcome", "unconfirmed")
	if !service.vault.Unlocked() || !service.sessions.VersionCurrent(accountID, version) {
		before = AccountSettingsSnapshot{AccountID: accountID, Settings: []AccountSettingView{}, SectionErrors: []AccountSettingsSectionError{}}
	}
	// Preserve the last confirmed baseline; never present a draft as applied or
	// automatically repeat a POST whose result might have been lost.
	return AccountSettingUpdateResult{Snapshot: before, Outcome: "unconfirmed", Message: "Could not confirm the change. Refresh settings to check."}, nil
}

func settingsReadBackConfirmed(snapshot AccountSettingsSnapshot, key AccountSettingKey, desired AccountSettingView) bool {
	current := findAccountSetting(snapshot, key)
	return settingKnownValue(current) && current.UnavailableReason != conflictingSettingReason && sameSettingValue(current, desired)
}

func definiteSettingRejection(err error) bool {
	var remote *roblox.Error
	if !errors.As(err, &remote) {
		return false
	}
	switch remote.Kind {
	case roblox.KindForbidden, roblox.KindCSRFRejected, roblox.KindChallengeRequired, roblox.KindInvalidSession, roblox.KindReauthRequired, roblox.KindRateLimited:
		return true
	default:
		return false
	}
}
