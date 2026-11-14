package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/appservice"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

func (service *Service) GetAvatar(ctx context.Context, accountID, userID int64) (robloxservices.Avatar, error) {
	if err := validateAccountAndTargetIDs(accountID, userID, "avatar"); err != nil {
		return robloxservices.Avatar{}, err
	}
	return service.core.GetAvatar(ctx, accountID, userID)
}

func (service *Service) GetAvatarRules(ctx context.Context, accountID int64) (robloxservices.AvatarRules, error) {
	if err := positiveID(accountID, "avatar-rules"); err != nil {
		return robloxservices.AvatarRules{}, err
	}
	return service.core.GetAvatarRules(ctx, accountID)
}

func (service *Service) SaveAvatar(ctx context.Context, accountID int64, change robloxservices.AvatarChange) (appservice.SavedAvatar, error) {
	if err := positiveID(accountID, "avatar-save"); err != nil {
		return appservice.SavedAvatar{}, err
	}
	return service.core.SaveAvatar(ctx, accountID, change)
}

func (service *Service) WaitForAvatarImage(ctx context.Context, accountID int64) (string, error) {
	if err := positiveID(accountID, "avatar-image"); err != nil {
		return "", err
	}
	return service.core.WaitForAvatarImage(ctx, accountID)
}

func (service *Service) GetAvatarOutfits(ctx context.Context, accountID, userID int64, cursor string) (robloxservices.AvatarOutfitPage, error) {
	if err := validateAccountAndTargetIDs(accountID, userID, "avatar-outfits"); err != nil {
		return robloxservices.AvatarOutfitPage{}, err
	}
	return service.core.GetAvatarOutfits(ctx, accountID, userID, cursor)
}

func (service *Service) GetAvatarOutfitDetails(ctx context.Context, accountID, outfitID int64) (robloxservices.Avatar, error) {
	if err := validateAccountAndTargetIDs(accountID, outfitID, "avatar-outfit-details"); err != nil {
		return robloxservices.Avatar{}, err
	}
	return service.core.GetAvatarOutfitDetails(ctx, accountID, outfitID)
}

func (service *Service) WearAvatarOutfit(ctx context.Context, accountID, outfitID int64) (*robloxservices.Avatar, error) {
	if err := validateAccountAndTargetIDs(accountID, outfitID, "avatar-outfit-wear"); err != nil {
		return nil, err
	}
	return service.core.WearAvatarOutfit(ctx, accountID, outfitID)
}

func (service *Service) CreateAvatarOutfit(ctx context.Context, accountID int64, name string) error {
	if err := positiveID(accountID, "avatar-outfit-create"); err != nil {
		return err
	}
	return service.core.CreateAvatarOutfit(ctx, accountID, name)
}

func (service *Service) RenameAvatarOutfit(ctx context.Context, accountID, outfitID int64, name string) error {
	if err := validateAccountAndTargetIDs(accountID, outfitID, "avatar-outfit-rename"); err != nil {
		return err
	}
	return service.core.RenameAvatarOutfit(ctx, accountID, outfitID, name)
}

func (service *Service) DeleteAvatarOutfit(ctx context.Context, accountID, outfitID int64) error {
	if err := validateAccountAndTargetIDs(accountID, outfitID, "avatar-outfit-delete"); err != nil {
		return err
	}
	return service.core.DeleteAvatarOutfit(ctx, accountID, outfitID)
}
