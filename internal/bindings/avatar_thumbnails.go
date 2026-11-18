package bindings

import (
	"context"

	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

func (service *Service) GetAvatarThumbnails(ctx context.Context, accountID, userID int64) (robloxservices.AvatarThumbnails, error) {
	if err := validateAccountAndTargetIDs(accountID, userID, "avatar-thumbnails"); err != nil {
		return robloxservices.AvatarThumbnails{}, err
	}
	return service.core.GetAvatarThumbnails(ctx, accountID, userID)
}

func (service *Service) SaveAvatarThumbnail(ctx context.Context, accountID int64, change robloxservices.AvatarThumbnailChange) error {
	if err := positiveID(accountID, "avatar-thumbnail-save"); err != nil {
		return err
	}
	return service.core.SaveAvatarThumbnail(ctx, accountID, change)
}
