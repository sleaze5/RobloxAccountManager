package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/appservice"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/vault"
)

func (service *Service) GetVaultState() appservice.VaultState { return service.core.GetVaultState() }

func (service *Service) CreateVault(ctx context.Context, password, confirmation, hint string, automaticUnlock bool) error {
	return service.core.CreateVault(ctx, password, confirmation, hint, automaticUnlock)
}

func (service *Service) UnlockVault(ctx context.Context, password string) error {
	return service.core.UnlockVault(ctx, password)
}

func (service *Service) LockVault() error { return service.core.LockVault() }

func (service *Service) CheckVaultPassword(ctx context.Context, password string) error {
	return service.core.CheckVaultPassword(ctx, password)
}

func (service *Service) DismissPasswordReminder() error {
	return service.core.DismissPasswordReminder()
}

func (service *Service) SetPasswordCheckIntervalDays(days int) error {
	return service.core.SetPasswordCheckIntervalDays(days)
}

func (service *Service) ChangeVaultPassword(ctx context.Context, currentPassword, newPassword, confirmation, hint string, automaticUnlock bool) error {
	return service.core.ChangeVaultPassword(ctx, currentPassword, newPassword, confirmation, hint, automaticUnlock)
}

func (service *Service) SetAutomaticUnlock(enabled bool) error {
	return service.core.SetAutomaticUnlock(enabled)
}

func (service *Service) ListBackups() ([]vault.Backup, error) { return service.core.ListBackups() }

func (service *Service) RestoreBackup(ctx context.Context, name, password string) error {
	return service.core.RestoreBackup(ctx, name, password)
}

func (service *Service) ResetVault(confirmation string) error {
	if confirmation != "RESET" {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "vault-reset", Message: "Type RESET to confirm vault deletion."}
	}
	return service.core.ResetVault()
}

func (service *Service) MigrateLegacyVault(ctx context.Context, password string) error {
	return service.core.MigrateLegacyVault(ctx, password)
}

func (service *Service) SetAsideLegacyVault(confirmation string) error {
	if confirmation != "CONFIRM" {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "vault-legacy", Message: "Type CONFIRM to set the old vault aside."}
	}
	return service.core.SetAsideLegacyVault()
}
