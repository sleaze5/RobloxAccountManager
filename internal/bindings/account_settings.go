package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/appservice"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

func (service *Service) GetRobuxTransactions(ctx context.Context, accountID int64, transactionType appservice.RobuxTransactionType, cursor string) (appservice.RobuxTransactionPage, error) {
	if err := positiveID(accountID, "robux-transactions"); err != nil {
		return appservice.RobuxTransactionPage{}, err
	}
	return service.core.GetRobuxTransactions(ctx, accountID, transactionType, cursor)
}

func (service *Service) GetAccountInfo(ctx context.Context, accountID int64, section appservice.AccountDetailsSection) (appservice.AccountDetailsSnapshot, error) {
	if err := positiveID(accountID, "account-info"); err != nil {
		return appservice.AccountDetailsSnapshot{}, err
	}
	return service.core.GetAccountInfo(ctx, accountID, section)
}

func (service *Service) GetAccountSettings(ctx context.Context, accountID int64) (appservice.AccountSettingsSnapshot, error) {
	if err := positiveID(accountID, "account-settings"); err != nil {
		return appservice.AccountSettingsSnapshot{}, err
	}
	return service.core.GetAccountSettings(ctx, accountID)
}

func (service *Service) UpdateAccountSetting(ctx context.Context, accountID int64, change appservice.AccountSettingChange) (appservice.AccountSettingUpdateResult, error) {
	if err := positiveID(accountID, "account-settings-update"); err != nil {
		return appservice.AccountSettingUpdateResult{}, err
	}
	return service.core.UpdateAccountSetting(ctx, accountID, change)
}

func (service *Service) ChangeDisplayName(ctx context.Context, accountID int64, displayName string) error {
	if err := positiveID(accountID, "display-name-update"); err != nil {
		return err
	}
	return service.core.ChangeDisplayName(ctx, accountID, displayName)
}

func (service *Service) CheckAccountDescription(ctx context.Context, accountID int64, description string) (robloxservices.TextFilter, error) {
	if err := positiveID(accountID, "description-validate"); err != nil {
		return robloxservices.TextFilter{}, err
	}
	return service.core.CheckAccountDescription(ctx, accountID, description)
}

func (service *Service) ChangeAccountDescription(ctx context.Context, accountID int64, description string) (string, error) {
	if err := positiveID(accountID, "description-update"); err != nil {
		return "", err
	}
	return service.core.ChangeAccountDescription(ctx, accountID, description)
}

func (service *Service) ChangeAccountEmail(ctx context.Context, accountID int64, password, email string) error {
	if err := positiveID(accountID, "email-update"); err != nil {
		return err
	}
	return service.core.ChangeAccountEmail(ctx, accountID, password, email)
}

func (service *Service) ChangeAccountPassword(ctx context.Context, accountID int64, currentPassword, newPassword string) error {
	if err := positiveID(accountID, "password-change"); err != nil {
		return err
	}
	return service.core.ChangeAccountPassword(ctx, accountID, currentPassword, newPassword)
}

func (service *Service) SaveAccountPassword(ctx context.Context, accountID int64, password string) error {
	if err := positiveID(accountID, "password-save"); err != nil {
		return err
	}
	return service.core.SaveAccountPassword(ctx, accountID, password)
}

func (service *Service) GetAccountPassword(ctx context.Context, accountID int64) (string, error) {
	if err := positiveID(accountID, "password-read"); err != nil {
		return "", err
	}
	return service.core.GetAccountPassword(ctx, accountID)
}

func (service *Service) CheckAccountPassword(ctx context.Context, accountID int64, password string) (bool, error) {
	if err := positiveID(accountID, "password-validate"); err != nil {
		return false, err
	}
	return service.core.CheckAccountPassword(ctx, accountID, password)
}

func (service *Service) UnlockAccountPin(ctx context.Context, accountID int64, pin string) error {
	if err := positiveID(accountID, "account-pin-unlock"); err != nil {
		return err
	}
	return service.core.UnlockAccountPin(ctx, accountID, pin)
}

func (service *Service) GetBlockedUsers(ctx context.Context, accountID int64) ([]appservice.BlockedUser, error) {
	if err := positiveID(accountID, "blocked-users"); err != nil {
		return nil, err
	}
	return service.core.GetBlockedUsers(ctx, accountID)
}

func (service *Service) UnblockUser(ctx context.Context, accountID, userID int64) error {
	if err := positiveID(accountID, "user-unblock"); err != nil {
		return err
	}
	if err := positiveID(userID, "user-unblock"); err != nil {
		return err
	}
	return service.core.UnblockUser(ctx, accountID, userID)
}

func (service *Service) GetAccountSessions(ctx context.Context, accountID int64) ([]appservice.AccountSession, error) {
	if err := positiveID(accountID, "account-sessions"); err != nil {
		return nil, err
	}
	return service.core.GetAccountSessions(ctx, accountID)
}

func (service *Service) SignOutOtherSessions(ctx context.Context, accountID int64) (appservice.CookieRenewalResult, error) {
	if err := positiveID(accountID, "sessions-sign-out"); err != nil {
		return appservice.CookieRenewalResult{}, err
	}
	return service.core.SignOutOtherSessions(ctx, accountID)
}
