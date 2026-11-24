package appservice

import (
	"context"

	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// GetAvatarBackground reads the Marketplace background that the user equipped,
// with its image, or nil when the user has none. A background whose image
// Roblox cannot render keeps an empty ImageURL.
func (service *Service) GetAvatarBackground(ctx context.Context, accountID, userID int64) (*robloxservices.AvatarBackground, error) {
	const operation = "avatar-background"
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, operation)
	if err != nil {
		return nil, err
	}
	defer done()
	logger := service.logs.Module("application.avatar").With("operation", operation, "account_id", accountID)
	background, err := service.users.EquippedAvatarBackground(ctx, accountID, version, userID)
	if err != nil {
		logger.WarnContext(ctx, "avatar background unavailable", "error", err)
		return nil, err
	}
	if background == nil {
		return nil, nil
	}
	images, imageError := service.thumbnails.Images(ctx, []robloxservices.ThumbnailTarget{background.ThumbnailTarget()}, robloxservices.WideThumbnailSize)
	if imageError != nil {
		logger.WarnContext(ctx, "avatar background image unavailable", "asset_id", background.AssetID, "error", imageError)
	}
	background.ImageURL = images[background.ThumbnailTarget()]
	return background, nil
}
