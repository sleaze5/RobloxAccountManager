package appservice

import (
	"context"

	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// GetAvatarPortrait reads the full-body avatar image of the user and the
// Marketplace background that the user equipped. A part that Roblox cannot
// give leaves the rest of the portrait, so that the profile still shows it.
func (service *Service) GetAvatarPortrait(ctx context.Context, accountID, userID int64) (robloxservices.AvatarPortrait, error) {
	const operation = "avatar-portrait"
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, operation)
	if err != nil {
		return robloxservices.AvatarPortrait{}, err
	}
	defer done()
	logger := service.logs.Module("application.avatar").With("operation", operation, "account_id", accountID)
	portrait := robloxservices.AvatarPortrait{}
	background, backgroundError := service.users.EquippedAvatarBackground(ctx, accountID, version, userID)
	if backgroundError != nil {
		logger.WarnContext(ctx, "avatar background unavailable", "error", backgroundError)
		portrait.IsBackgroundUnavailable = true
	}
	fullBodyTarget := robloxservices.AvatarImageTarget(userID)
	fullBodyImages, imageError := service.thumbnails.Images(ctx, []robloxservices.ThumbnailTarget{fullBodyTarget}, robloxservices.LargeThumbnailSize)
	if imageError != nil {
		logger.WarnContext(ctx, "avatar full-body image unavailable", "error", imageError)
	}
	portrait.FullBodyImageURL = fullBodyImages[fullBodyTarget]
	if background != nil {
		backgroundImages, backgroundImageError := service.thumbnails.Images(ctx, []robloxservices.ThumbnailTarget{background.ThumbnailTarget()}, robloxservices.WideThumbnailSize)
		if backgroundImageError != nil {
			logger.WarnContext(ctx, "avatar background image unavailable", "asset_id", background.AssetID, "error", backgroundImageError)
		}
		background.ImageURL = backgroundImages[background.ThumbnailTarget()]
		portrait.Background = background
	}
	return portrait, nil
}
