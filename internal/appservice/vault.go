package appservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/vault"
)

type VaultState struct {
	Initialized              bool            `json:"initialized"`
	Unlocked                 bool            `json:"unlocked"`
	FileState                vault.FileState `json:"fileState"`
	PasswordHint             string          `json:"passwordHint,omitempty"`
	PasswordReminderDue      bool            `json:"passwordReminderDue"`
	PasswordTestIntervalDays int             `json:"passwordTestIntervalDays"`
	LastPasswordTestedAt     string          `json:"lastPasswordTestedAt,omitempty"`
	AutomaticUnlockRecovery  bool            `json:"automaticUnlockRecovery"`
	AutomaticUnlock          bool            `json:"automaticUnlock"`
	ValidAutomaticUnlock     bool            `json:"validAutomaticUnlock"`
	StartupError             string          `json:"startupError,omitempty"`
}

func (service *Service) GetVaultState() VaultState {
	service.mu.Lock()
	defer service.mu.Unlock()
	settings := service.settings.Vault()
	return VaultState{
		Initialized:              service.vault.Initialized(),
		Unlocked:                 service.vault.Unlocked(),
		FileState:                service.vault.FileState(),
		PasswordHint:             service.vault.PasswordHint(),
		PasswordReminderDue:      service.reminderDue,
		PasswordTestIntervalDays: settings.PasswordTestIntervalDays,
		LastPasswordTestedAt:     settings.LastPasswordTestedAt,
		AutomaticUnlockRecovery:  service.vault.UnlockedAutomatically(),
		AutomaticUnlock:          service.vault.AutomaticUnlockEnabled(),
		ValidAutomaticUnlock:     service.vault.ValidAutomaticUnlock(),
		StartupError:             service.startupError,
	}
}

func (service *Service) CreateVault(ctx context.Context, password, confirmation, hint string, automaticUnlock bool) error {
	if password != confirmation {
		return errors.New("master passwords do not match")
	}
	if err := service.vault.Create(ctx, password, hint, automaticUnlock); err != nil {
		return safeVaultError(err)
	}
	if err := service.settings.RecordPasswordTested(time.Now()); err != nil {
		service.logger.Error("record master password setup failed", "error", err)
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
	if err := service.settings.RecordPasswordTested(time.Now()); err != nil {
		service.logger.Error("record master password verification failed", "error", err)
	}
	service.finishUnlock(false)
	return nil
}

func (service *Service) finishUnlock(automatic bool) {
	service.sessions.Unlock()
	if err := service.vault.RemoveMigration(); err != nil {
		service.logger.Warn("stale vault migration file could not be removed", "operation", "vault-migration", "error", err)
	}
	if err := service.applyPendingSettings(service.lifecycle); err != nil {
		service.logger.Warn("imported settings could not be saved after unlock", "operation", "vault-import", "error", err)
	}
	var cursor *accounts.AccountCursor
	for {
		page, err := service.ListAccounts(service.lifecycle, accounts.AccountQuery{Cursor: cursor})
		if err != nil {
			service.logger.Warn("account browser IDs could not be initialized", "error", err)
			break
		}
		if page.NextCursor == nil {
			break
		}
		cursor = page.NextCursor
	}
	due := service.settings.PasswordReminderDue(time.Now())
	service.mu.Lock()
	service.startupError = ""
	if service.gamesCancel != nil {
		service.gamesCancel()
	}
	service.gamesContext, service.gamesCancel = context.WithCancel(service.lifecycle)
	service.reminderDue = automatic && due
	service.mu.Unlock()
	service.startBackground()
	if service.browser != nil {
		service.browser.AllowLaunches()
	}
	service.events.send("vault:unlocked", struct{}{})
	go func() {
		if err := service.vault.MaybeCreateBackup(context.WithoutCancel(service.lifecycle)); err != nil {
			service.logger.Warn("automatic vault backup failed", "error", err)
		}
	}()
}

func (service *Service) LockVault() error {
	service.cancelLaunchConfirmation()
	service.cancelPendingValidations()
	if service.browser != nil {
		service.browser.ForceShutdown()
	}
	service.clearPendingImports()
	service.stopBackground()
	if err := service.vault.Lock(); err != nil {
		if service.vault.Unlocked() {
			service.startBackground()
			if service.browser != nil {
				service.browser.AllowLaunches()
			}
		} else {
			service.finishLocked()
		}
		service.logger.Error("vault lock workflow failed", "error", err)
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
	service.mu.Lock()
	service.reminderDue = false
	service.mu.Unlock()
	service.events.send("vault:locked", struct{}{})
}

func (service *Service) TestVaultPassword(ctx context.Context, password string) error {
	if err := service.vault.TestPassword(ctx, password); err != nil {
		return safeVaultError(err)
	}
	if err := service.settings.RecordPasswordTested(time.Now()); err != nil {
		return fmt.Errorf("save password test time: %w", err)
	}
	service.mu.Lock()
	service.reminderDue = false
	service.mu.Unlock()
	return nil
}

func (service *Service) DismissPasswordReminder() error {
	if err := service.settings.DismissPasswordReminder(time.Now()); err != nil {
		return fmt.Errorf("save password reminder dismissal: %w", err)
	}
	service.mu.Lock()
	service.reminderDue = false
	service.mu.Unlock()
	return nil
}

func (service *Service) ChangeVaultPassword(ctx context.Context, currentPassword, newPassword, confirmation, hint string, automaticUnlock bool) error {
	recovering := service.vault.FileState() == vault.FileStateIncomplete
	if err := service.vault.ChangePassword(ctx, currentPassword, newPassword, confirmation, hint, automaticUnlock); err != nil {
		return err
	}
	if err := service.settings.RecordPasswordTested(time.Now()); err != nil {
		service.logger.Error("record changed master password failed", "error", err)
	}
	service.mu.Lock()
	service.reminderDue = false
	service.mu.Unlock()
	if recovering {
		service.finishUnlock(false)
	}
	return nil
}

func (service *Service) SetPasswordTestIntervalDays(days int) error {
	return service.settings.SetPasswordTestIntervalDays(days)
}

func (service *Service) SetAutomaticUnlock(enabled bool) error {
	if err := service.vault.SetAutomaticUnlock(enabled); err != nil {
		service.logger.Error("automatic unlock update failed", "enabled", enabled, "error", err)
		if enabled {
			return errors.New("automatic unlock could not be turned on")
		}
		return errors.New("automatic unlock could not be turned off")
	}
	service.logger.Info("automatic unlock updated", "enabled", enabled)
	return nil
}

func (service *Service) ListBackups() ([]vault.BackupInfo, error) { return service.vault.ListBackups() }

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
		service.logger.Error("legacy vault could not be set aside", "operation", "vault-legacy", "error", err)
		return errors.New("the old vault could not be set aside")
	}
	return nil
}
