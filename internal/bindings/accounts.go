package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/appservice"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

func (service *Service) GetAccountProfile(ctx context.Context, accountID int64, section appservice.AccountProfileSection) (appservice.AccountProfileSnapshot, error) {
	if err := positiveID(accountID, "account-profile"); err != nil {
		return appservice.AccountProfileSnapshot{}, err
	}
	return service.core.GetAccountProfile(ctx, accountID, section)
}

func (service *Service) CheckCookies(ctx context.Context, input string) (appservice.ImportPreview, error) {
	return service.core.CheckCookies(ctx, input)
}

func (service *Service) SaveValidatedCookies(ctx context.Context, batchID string, selectedIndexes []int) (appservice.ImportBatchResult, error) {
	return service.core.SaveValidatedCookies(ctx, batchID, selectedIndexes)
}

func (service *Service) DiscardValidatedCookies(batchID string) {
	service.core.DiscardValidatedCookies(batchID)
}

func (service *Service) RenewCookie(ctx context.Context, accountID int64) (appservice.CookieRenewalResult, error) {
	if err := positiveID(accountID, "cookie-renewal"); err != nil {
		return appservice.CookieRenewalResult{}, err
	}
	return service.core.RenewCookie(ctx, accountID)
}

func (service *Service) ReactivateAccount(ctx context.Context, accountID int64) (accounts.AccountView, error) {
	if err := positiveID(accountID, "account-reactivate"); err != nil {
		return accounts.AccountView{}, err
	}
	return service.core.ReactivateAccount(ctx, accountID)
}

func (service *Service) RemoveAccount(ctx context.Context, accountID int64) error {
	if err := positiveID(accountID, "account-remove"); err != nil {
		return err
	}
	return service.core.RemoveAccount(ctx, accountID)
}

func (service *Service) ListAccounts(ctx context.Context, query accounts.AccountQuery) (accounts.AccountPage, error) {
	return service.core.ListAccounts(ctx, query)
}

func (service *Service) MoveAccount(ctx context.Context, accountID int64, beforeID, afterID *int64) error {
	if err := positiveID(accountID, "account-order"); err != nil {
		return err
	}
	return service.core.MoveAccount(ctx, accountID, beforeID, afterID)
}

func (service *Service) ListTags(ctx context.Context) ([]accounts.TagView, error) {
	return service.core.ListTags(ctx)
}

func (service *Service) CreateTag(ctx context.Context, name string) (accounts.TagView, error) {
	return service.core.CreateTag(ctx, name)
}

func (service *Service) RemoveTag(ctx context.Context, tagID int64) error {
	if err := positiveID(tagID, "account-tag"); err != nil {
		return err
	}
	return service.core.RemoveTag(ctx, tagID)
}

func (service *Service) SetAccountsTag(ctx context.Context, accountIDs []int64, tagID int64, selected bool) ([]accounts.AccountView, error) {
	if err := accountSelection(accountIDs, "account-tag"); err != nil {
		return nil, err
	}
	if err := positiveID(tagID, "account-tag"); err != nil {
		return nil, err
	}
	return service.core.SetAccountsTag(ctx, accountIDs, tagID, selected)
}

func (service *Service) GetAvatarHeadshots(ctx context.Context, userIDs []int64) ([]robloxservices.AvatarHeadshotView, error) {
	for _, userID := range userIDs {
		if err := positiveID(userID, "avatar-headshots"); err != nil {
			return nil, err
		}
	}
	return service.core.GetAvatarHeadshots(ctx, userIDs)
}

func (service *Service) GetAccountPresences(ctx context.Context, accountID *int64) ([]robloxservices.UserPresence, error) {
	if accountID != nil {
		if err := positiveID(*accountID, "user-presences"); err != nil {
			return nil, err
		}
	}
	return service.core.GetAccountPresences(ctx, accountID)
}

func (service *Service) FindAccountByRobloxUserID(ctx context.Context, robloxUserID int64) (*accounts.AccountView, error) {
	if err := positiveID(robloxUserID, "account-lookup"); err != nil {
		return nil, err
	}
	return service.core.FindAccountByRobloxUserID(ctx, robloxUserID)
}

func (service *Service) GetRobloxUser(ctx context.Context, robloxUserID int64) (robloxservices.UserView, error) {
	if err := positiveID(robloxUserID, "user-profile"); err != nil {
		return robloxservices.UserView{}, err
	}
	return service.core.GetRobloxUser(ctx, robloxUserID)
}

func (service *Service) CheckAccount(ctx context.Context, accountID int64) (accounts.AccountView, error) {
	if err := positiveID(accountID, "account-validation"); err != nil {
		return accounts.AccountView{}, err
	}
	return service.core.CheckAccount(ctx, accountID)
}
