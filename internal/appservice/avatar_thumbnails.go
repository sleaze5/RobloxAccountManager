package appservice

import (
	"context"
	"sync"

	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// GetAvatarThumbnails reads how Roblox poses and frames the avatar images of
// the user, with the names and images of their emotes and the headshot and
// bust images. A missing name or image leaves it out.
func (service *Service) GetAvatarThumbnails(ctx context.Context, accountID, userID int64) (robloxservices.AvatarThumbnails, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "avatar-thumbnails")
	if err != nil {
		return robloxservices.AvatarThumbnails{}, err
	}
	defer done()
	logger := service.logs.Module("application.avatar").With("operation", "avatar-thumbnails", "account_id", accountID)
	thumbnails, err := service.users.AvatarThumbnailCustomizations(ctx, accountID, version, userID)
	if err != nil {
		logger.WarnContext(ctx, "avatar thumbnail customizations unavailable", "error", err)
		return robloxservices.AvatarThumbnails{}, err
	}
	headshotTarget, bustTarget := robloxservices.AvatarHeadshotImageTarget(userID), robloxservices.AvatarBustImageTarget(userID)
	emoteTargets := make([]robloxservices.ThumbnailTarget, 0, 2)
	for _, emote := range []*robloxservices.InventoryItem{thumbnails.Closeup.Emote, thumbnails.FullBody.Emote} {
		if emote != nil {
			emoteTargets = append(emoteTargets, emote.ThumbnailTarget())
		}
	}
	var group sync.WaitGroup
	group.Go(func() {
		if detailError := service.users.AddThumbnailEmoteDetails(ctx, accountID, version, &thumbnails); detailError != nil {
			logger.WarnContext(ctx, "avatar thumbnail emote details unavailable", "error", detailError)
		}
	})
	var avatarImages, emoteImages map[robloxservices.ThumbnailTarget]string
	group.Go(func() {
		var imageError error
		if avatarImages, imageError = service.thumbnails.Images(ctx, []robloxservices.ThumbnailTarget{headshotTarget, bustTarget}, robloxservices.LargeThumbnailSize); imageError != nil {
			logger.WarnContext(ctx, "avatar close-up images unavailable", "error", imageError)
		}
	})
	group.Go(func() {
		var imageError error
		if emoteImages, imageError = service.thumbnails.Images(ctx, emoteTargets, robloxservices.SmallThumbnailSize); imageError != nil {
			logger.WarnContext(ctx, "avatar thumbnail emote images unavailable", "error", imageError)
		}
	})
	group.Wait()
	thumbnails.HeadshotImageURL = avatarImages[headshotTarget]
	thumbnails.BustImageURL = avatarImages[bustTarget]
	for _, emote := range []*robloxservices.InventoryItem{thumbnails.Closeup.Emote, thumbnails.FullBody.Emote} {
		if emote != nil {
			emote.ImageURL = emoteImages[emote.ThumbnailTarget()]
		}
	}
	return thumbnails, nil
}

func (service *Service) SaveAvatarThumbnail(ctx context.Context, accountID int64, change robloxservices.AvatarThumbnailChange) error {
	return service.editAccount(ctx, accountID, "avatar-thumbnail-save", "", func(ctx context.Context, version int64) error {
		return service.users.SaveAvatarThumbnail(ctx, accountID, version, change)
	})
}

// GetAvatarAnimation reads the emote or animation that the reference names,
// such as a pasted ID or catalog link, with its image. A missing image leaves
// the item without it.
func (service *Service) GetAvatarAnimation(ctx context.Context, accountID int64, reference string) (robloxservices.InventoryItem, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "avatar-animation")
	if err != nil {
		return robloxservices.InventoryItem{}, err
	}
	defer done()
	logger := service.logs.Module("application.avatar").With("operation", "avatar-animation", "account_id", accountID)
	animation, err := service.users.AvatarAnimation(ctx, accountID, version, reference)
	if err != nil {
		// A pasted ID that names no animation is an input mistake.
		logger.InfoContext(ctx, "avatar animation unavailable", "error", err)
		return robloxservices.InventoryItem{}, err
	}
	images, err := service.thumbnails.Images(ctx, []robloxservices.ThumbnailTarget{animation.ThumbnailTarget()}, robloxservices.SmallThumbnailSize)
	if err != nil {
		logger.WarnContext(ctx, "avatar animation image unavailable", "error", err)
	}
	animation.ImageURL = images[animation.ThumbnailTarget()]
	return animation, nil
}
