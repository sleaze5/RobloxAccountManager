package appservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/appsettings"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	transferstore "github.com/sleaze5/RobloxAccountManager/internal/storage/transfer"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/vault"
)

type VaultState struct {
	Initialized               bool            `json:"initialized"`
	Unlocked                  bool            `json:"unlocked"`
	FileState                 vault.FileState `json:"fileState"`
	MigrationAvailable        bool            `json:"migrationAvailable"`
	PasswordHint              string          `json:"passwordHint,omitempty"`
	PasswordReminderDue       bool            `json:"passwordReminderDue"`
	PasswordCheckIntervalDays int             `json:"passwordCheckIntervalDays"`
	LastPasswordCheckedAt     string          `json:"lastPasswordCheckedAt,omitempty"`
	AutomaticUnlockRecovery   bool            `json:"automaticUnlockRecovery"`
	AutomaticUnlock           bool            `json:"automaticUnlock"`
	ValidAutomaticUnlock      bool            `json:"validAutomaticUnlock"`
	StartupError              string          `json:"startupError,omitempty"`
}

func (service *Service) GetVaultState() VaultState {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	settings := service.settings.Vault()
	return VaultState{
		Initialized:               service.vault.Initialized(),
		Unlocked:                  service.vault.Unlocked(),
		FileState:                 service.vault.FileState(),
		MigrationAvailable:        service.vault.MigrationAvailable(),
		PasswordHint:              service.vault.PasswordHint(),
		PasswordReminderDue:       service.reminderDue,
		PasswordCheckIntervalDays: settings.PasswordCheckIntervalDays,
		LastPasswordCheckedAt:     settings.LastPasswordCheckedAt,
		AutomaticUnlockRecovery:   service.vault.UnlockedAutomatically(),
		AutomaticUnlock:           service.vault.AutomaticUnlockEnabled(),
		ValidAutomaticUnlock:      service.vault.ValidAutomaticUnlock(),
		StartupError:              service.startupError,
	}
}

func (service *Service) CreateVault(ctx context.Context, password, confirmation, hint string, automaticUnlock bool) error {
	if password != confirmation {
		return errors.New("master passwords do not match")
	}
	if err := service.vault.Create(ctx, password, hint, automaticUnlock); err != nil {
		service.logger.Error("could not create the vault", "operation", "vault-create", "error", err)
		message := vaultFailureMessage(err, "Could not create the vault. Try again.")
		if errors.Is(err, vault.ErrAutomaticUnlockUnavailable) {
			message = keyringUnavailableMessage + " Turn off automatic unlock, then create the vault."
		}
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "vault-create", Message: message, Cause: err}
	}
	if err := service.settings.RecordPasswordChecked(time.Now()); err != nil {
		service.logger.Error("could not record the master password setup", "error", err)
	}
	service.finishUnlock(false)
	return nil
}

func (service *Service) UnlockVault(ctx context.Context, password string) error {
	if service.vault.Unlocked() {
		return nil
	}
	if err := service.vault.Unlock(ctx, password); err != nil {
		return safeVaultError(err)
	}
	if err := service.settings.RecordPasswordChecked(time.Now()); err != nil {
		service.logger.Error("could not record the master password check", "error", err)
	}
	service.finishUnlock(false)
	return nil
}

func (service *Service) finishUnlock(automatic bool) {
	service.sessions.Unlock()
	if err := service.vault.RemoveMigration(); err != nil {
		service.logger.Warn("could not remove the old vault migration file", "operation", "vault-migration", "error", err)
	}
	if err := service.applyPendingSettings(service.lifecycle); err != nil {
		service.logger.Warn("could not save the imported settings after unlock", "operation", "vault-import", "error", err)
	}
	views, err := service.savedAccounts(service.lifecycle)
	if err != nil {
		service.logger.Warn("could not set up account browser IDs", "operation", "vault-unlock", "error", err)
	}
	for _, view := range views {
		service.sessions.BrowserID(view.ID)
	}
	due := service.settings.PasswordReminderDue(time.Now())
	service.mutex.Lock()
	service.startupError = ""
	if service.gamesCancel != nil {
		service.gamesCancel()
	}
	service.gamesContext, service.gamesCancel = context.WithCancel(service.lifecycle)
	logsContext := service.gamesContext
	service.reminderDue = automatic && due
	service.mutex.Unlock()
	go service.readRobloxLogsAhead(logsContext)
	service.startBackground()
	service.browser.AllowLaunches()
	go func() {
		if err := service.vault.MaybeCreateBackup(context.WithoutCancel(service.lifecycle)); err != nil {
			service.logger.Warn("could not back up the vault automatically", "error", err)
		}
	}()
}

func (service *Service) LockVault() error {
	service.cancelLaunchConfirmation()
	service.cancelPendingValidations()
	service.browser.ForceShutdown()
	service.clearPendingImports()
	service.stopBackground()
	if err := service.vault.Lock(); err != nil {
		if service.vault.Unlocked() {
			service.startBackground()
			service.browser.AllowLaunches()
		} else {
			service.finishLocked()
		}
		service.logger.Error("could not lock the vault", "error", err)
		return err
	}
	service.finishLocked()
	return nil
}

func (service *Service) finishLocked() {
	service.stopGames()
	service.clearPendingImports()
	service.discardPendingTransfer()
	service.sessions.Lock()
	service.mutex.Lock()
	service.reminderDue = false
	service.mutex.Unlock()
	service.events.send("vault:locked", struct{}{})
}

func (service *Service) CheckVaultPassword(ctx context.Context, password string) error {
	if err := service.vault.CheckPassword(ctx, password); err != nil {
		return safeVaultError(err)
	}
	if err := service.settings.RecordPasswordChecked(time.Now()); err != nil {
		return fmt.Errorf("save password check time: %w", err)
	}
	service.mutex.Lock()
	service.reminderDue = false
	service.mutex.Unlock()
	return nil
}

func (service *Service) DismissPasswordReminder() error {
	if err := service.settings.DismissPasswordReminder(time.Now()); err != nil {
		return fmt.Errorf("save password reminder dismissal: %w", err)
	}
	service.mutex.Lock()
	service.reminderDue = false
	service.mutex.Unlock()
	return nil
}

func (service *Service) ChangeVaultPassword(ctx context.Context, currentPassword, newPassword, confirmation, hint string, automaticUnlock bool) error {
	recovering := service.vault.FileState() == vault.FileStateIncomplete
	if err := service.vault.ChangePassword(ctx, currentPassword, newPassword, confirmation, hint, automaticUnlock); err != nil {
		return err
	}
	if err := service.settings.RecordPasswordChecked(time.Now()); err != nil {
		service.logger.Error("could not record the changed master password", "error", err)
	}
	service.mutex.Lock()
	service.reminderDue = false
	service.mutex.Unlock()
	if recovering {
		service.finishUnlock(false)
	}
	return nil
}

func (service *Service) SetPasswordCheckIntervalDays(days int) error {
	return service.settings.SetPasswordCheckIntervalDays(days)
}

func (service *Service) SetAutomaticUnlock(enabled bool) error {
	if err := service.vault.SetAutomaticUnlock(enabled); err != nil {
		service.logger.Error("could not update automatic unlock", "operation", "automatic-unlock", "enabled", enabled, "error", err)
		message := "Could not turn off automatic unlock. Try again."
		switch {
		case enabled && errors.Is(err, vault.ErrAutomaticUnlockUnavailable):
			message = keyringUnavailableMessage
		case enabled:
			message = "Could not turn on automatic unlock. Try again."
		}
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "automatic-unlock", Message: message, Cause: err}
	}
	service.logger.Info("automatic unlock updated", "enabled", enabled)
	return nil
}

func (service *Service) ListBackups() ([]vault.Backup, error) { return service.vault.ListBackups() }

func (service *Service) RestoreBackup(ctx context.Context, name, password string) error {
	if err := service.vault.RestoreBackup(ctx, name, password); err != nil {
		if errors.Is(err, vault.ErrLegacyVault) {
			return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "vault-restore", Message: legacyVaultMessage, Cause: err}
		}
		return err
	}
	service.sessions.ResetBrowserIDs()
	return nil
}

func (service *Service) ResetVault() error {
	service.clearPendingImports()
	if err := service.vault.ResetVault(); err != nil {
		return err
	}
	service.sessions.ResetBrowserIDs()
	return nil
}

func (service *Service) SetAsideLegacyVault() error {
	if err := service.vault.SetAsideLegacyVault(); err != nil {
		service.logger.Error("could not set the legacy vault aside", "operation", "vault-legacy", "error", err)
		return errors.New("the old vault could not be set aside")
	}
	return nil
}

// MigrateLegacyVault imports the data that 0.9.2 saved before the update into
// a new vault, with the same master password as the old vault.
func (service *Service) MigrateLegacyVault(ctx context.Context, password string) error {
	var payload transferstore.Payload
	err := service.vault.MigrateLegacyVault(ctx, password, func(payloadJSON []byte) error {
		var err error
		if payload, err = transferstore.Decode(payloadJSON); err != nil {
			return err
		}
		_, err = appsettings.Decode(payload.Settings)
		return err
	})
	if err != nil {
		if errors.Is(err, vault.ErrInvalidPassword) {
			return safeVaultError(err)
		}
		service.logger.Error("could not move the legacy vault", "operation", "vault-migration", "error", err)
		return transferError("Could not move your data to the new vault format. Download version 0.9.2 to export it instead.", err)
	}
	if _, err := service.transfers.Replace(ctx, payload, pendingSettings(&pendingTransfer{payload: payload}, appsettings.Sections)); err != nil {
		service.logger.Error("could not import the moved data", "operation", "vault-migration", "error", err)
		return transferError("The app created the new vault but did not import your data. The old vault is in the recovery folder in storage/vault.", err)
	}
	if err := service.settings.RecordPasswordChecked(time.Now()); err != nil {
		service.logger.Error("could not record the master password after the move", "error", err)
	}
	service.finishUnlock(false)
	service.logger.Info("vault data moved", "operation", "vault-migration", "accounts", len(payload.Accounts))
	return nil
}
