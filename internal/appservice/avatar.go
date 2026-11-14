package appservice

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// SavedAvatar is the avatar after a save, or a nil Avatar when the save
// succeeded but the avatar could not be read again. SkippedAssetIDs lists the
// requested assets that Roblox did not put on the avatar, such as items that
// the account does not own.
type SavedAvatar struct {
	Avatar          *robloxservices.Avatar `json:"avatar"`
	SkippedAssetIDs []int64                `json:"skippedAssetIds"`
}

// GetAvatar reads how the user looks now, as the account sees it, with the
// images and catalog details of the worn items. A missing image or catalog
// detail leaves the avatar without it.
func (service *Service) GetAvatar(ctx context.Context, accountID, userID int64) (robloxservices.Avatar, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "avatar")
	if err != nil {
		return robloxservices.Avatar{}, err
	}
	defer done()
	logger := service.logs.Module("application.avatar").With("operation", "avatar", "account_id", accountID)
	avatar, err := service.users.Avatar(ctx, accountID, version, userID)
	if err != nil {
		logger.WarnContext(ctx, "avatar unavailable", "error", err)
		return robloxservices.Avatar{}, err
	}
	service.addAvatarDetails(ctx, accountID, version, &avatar, robloxservices.AvatarImageTarget(userID), logger)
	return avatar, nil
}

// GetAvatarOutfitDetails reads how the outfit looks, with the images and
// catalog details of its items. A missing image or catalog detail leaves the
// outfit without it.
func (service *Service) GetAvatarOutfitDetails(ctx context.Context, accountID, outfitID int64) (robloxservices.Avatar, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "avatar-outfit-details")
	if err != nil {
		return robloxservices.Avatar{}, err
	}
	defer done()
	logger := service.logs.Module("application.avatar").With("operation", "avatar-outfit-details", "account_id", accountID)
	outfit, err := service.users.AvatarOutfitDetails(ctx, accountID, version, outfitID)
	if err != nil {
		logger.WarnContext(ctx, "avatar outfit details unavailable", "error", err)
		return robloxservices.Avatar{}, err
	}
	service.addAvatarDetails(ctx, accountID, version, &outfit, robloxservices.AvatarOutfitImageTarget(outfitID), logger)
	return outfit, nil
}

// addAvatarDetails adds the catalog details and images of the worn items, and
// the large image of the target, such as the avatar or the outfit.
func (service *Service) addAvatarDetails(ctx context.Context, accountID, version int64, avatar *robloxservices.Avatar, imageTarget robloxservices.ThumbnailTarget, logger *slog.Logger) {
	// The catalog details replace the worn items while the images load, so
	// the image targets are read before either starts.
	itemImageTargets := avatar.ThumbnailTargets()
	var group sync.WaitGroup
	group.Go(func() {
		if err := service.users.AddAvatarCatalogDetails(ctx, accountID, version, avatar); err != nil {
			logger.WarnContext(ctx, "avatar catalog details unavailable", "error", err)
		}
	})
	var itemImages, avatarImages map[robloxservices.ThumbnailTarget]string
	group.Go(func() {
		var imageError error
		if itemImages, imageError = service.thumbnails.Images(ctx, itemImageTargets, robloxservices.SmallThumbnailSize); imageError != nil {
			logger.WarnContext(ctx, "avatar item images unavailable", "error", imageError)
		}
	})
	group.Go(func() {
		var imageError error
		if avatarImages, imageError = service.thumbnails.Images(ctx, []robloxservices.ThumbnailTarget{imageTarget}, robloxservices.LargeThumbnailSize); imageError != nil {
			logger.WarnContext(ctx, "avatar image unavailable", "error", imageError)
		}
	})
	group.Wait()
	avatar.AddImages(itemImages)
	avatar.ImageURL = avatarImages[imageTarget]
}

// GetAvatarRules reads what the account may change on its avatar.
func (service *Service) GetAvatarRules(ctx context.Context, accountID int64) (robloxservices.AvatarRules, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "avatar-rules")
	if err != nil {
		return robloxservices.AvatarRules{}, err
	}
	defer done()
	rules, err := service.users.AvatarRules(ctx, accountID, version)
	if err != nil {
		service.logs.Module("application.avatar").WarnContext(ctx, "avatar rules unavailable", "operation", "avatar-rules", "account_id", accountID, "error", err)
		return robloxservices.AvatarRules{}, err
	}
	return rules, nil
}

// SaveAvatar replaces the avatar of the account and reads it again, so that
// the result shows what Roblox kept. A failed read returns no avatar instead
// of an error, because the save itself succeeded.
func (service *Service) SaveAvatar(ctx context.Context, accountID int64, change robloxservices.AvatarChange) (SavedAvatar, error) {
	account, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return SavedAvatar{}, mapRepositoryError(err)
	}
	err = service.editAccount(ctx, accountID, "avatar-save", "", func(ctx context.Context, version int64) error {
		return service.users.SaveAvatar(ctx, accountID, version, change)
	})
	if err != nil {
		return SavedAvatar{}, err
	}
	saved := SavedAvatar{SkippedAssetIDs: []int64{}}
	// GetAvatar logs a failed read.
	avatar, readError := service.GetAvatar(ctx, accountID, account.RobloxUserID)
	if readError == nil {
		saved.Avatar = &avatar
		for _, assetID := range change.AssetIDs {
			isWorn := slices.ContainsFunc(avatar.Assets, func(asset robloxservices.AvatarAsset) bool { return asset.Item.ItemID == assetID })
			if !isWorn {
				saved.SkippedAssetIDs = append(saved.SkippedAssetIDs, assetID)
			}
		}
	}
	return saved, nil
}

// WaitForAvatarImage reads the full-body image of the account after a change.
// Roblox renders a changed avatar a few seconds after the save and answers
// Pending until then, so the read repeats a bounded number of times.
func (service *Service) WaitForAvatarImage(ctx context.Context, accountID int64) (string, error) {
	const (
		maxAttempts = 10
		retryDelay  = 2 * time.Second
	)
	account, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return "", mapRepositoryError(err)
	}
	target := robloxservices.AvatarImageTarget(account.RobloxUserID)
	for attempt := range maxAttempts {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", &roblox.Error{Kind: roblox.KindCancelled, Endpoint: "avatar-image", Message: "The avatar image was not loaded.", Cause: ctx.Err()}
			case <-time.After(retryDelay):
			}
		}
		images, err := service.thumbnails.Images(ctx, []robloxservices.ThumbnailTarget{target}, robloxservices.LargeThumbnailSize)
		if err != nil {
			service.logs.Module("application.avatar").WarnContext(ctx, "avatar image unavailable", "operation", "avatar-image", "account_id", accountID, "error", err)
			return "", err
		}
		if imageURL := images[target]; imageURL != "" {
			return imageURL, nil
		}
	}
	return "", &roblox.Error{Kind: roblox.KindTimeout, Endpoint: "avatar-image", Message: "Roblox is still rendering the avatar. Refresh it in a minute."}
}

// GetAvatarOutfits reads one page of the outfits that the user created, with
// their images. A missing image leaves the outfit without it.
func (service *Service) GetAvatarOutfits(ctx context.Context, accountID, userID int64, cursor string) (robloxservices.AvatarOutfitPage, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "avatar-outfits")
	if err != nil {
		return robloxservices.AvatarOutfitPage{}, err
	}
	defer done()
	logger := service.logs.Module("application.avatar").With("operation", "avatar-outfits", "account_id", accountID)
	page, err := service.users.AvatarOutfits(ctx, accountID, version, userID, cursor)
	if err != nil {
		logger.WarnContext(ctx, "avatar outfits unavailable", "error", err)
		return robloxservices.AvatarOutfitPage{}, err
	}
	targets := make([]robloxservices.ThumbnailTarget, len(page.Outfits))
	for index, outfit := range page.Outfits {
		targets[index] = outfit.ThumbnailTarget()
	}
	images, err := service.thumbnails.Images(ctx, targets, robloxservices.SmallThumbnailSize)
	if err != nil {
		logger.WarnContext(ctx, "avatar outfit images unavailable", "error", err)
	}
	for index := range page.Outfits {
		page.Outfits[index].ImageURL = images[targets[index]]
	}
	return page, nil
}

// WearAvatarOutfit puts the outfit on the avatar of the account and reads the
// avatar again.
func (service *Service) WearAvatarOutfit(ctx context.Context, accountID, outfitID int64) (robloxservices.Avatar, error) {
	account, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return robloxservices.Avatar{}, mapRepositoryError(err)
	}
	err = service.editAccount(ctx, accountID, "avatar-outfit-wear", "", func(ctx context.Context, version int64) error {
		return service.users.WearAvatarOutfit(ctx, accountID, version, outfitID)
	})
	if err != nil {
		return robloxservices.Avatar{}, err
	}
	avatar, err := service.GetAvatar(ctx, accountID, account.RobloxUserID)
	if err != nil {
		return robloxservices.Avatar{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "avatar-outfit-wear", Message: "The outfit was put on but the avatar could not be reloaded. Refresh the avatar.", Cause: err}
	}
	return avatar, nil
}

func (service *Service) CreateAvatarOutfit(ctx context.Context, accountID int64, name string) error {
	return service.editAccount(ctx, accountID, "avatar-outfit-create", "", func(ctx context.Context, version int64) error {
		return service.users.CreateAvatarOutfit(ctx, accountID, version, name)
	})
}

func (service *Service) RenameAvatarOutfit(ctx context.Context, accountID, outfitID int64, name string) error {
	return service.editAccount(ctx, accountID, "avatar-outfit-rename", "You can only rename this account's own outfits.", func(ctx context.Context, version int64) error {
		return service.users.RenameAvatarOutfit(ctx, accountID, version, outfitID, name)
	})
}

func (service *Service) DeleteAvatarOutfit(ctx context.Context, accountID, outfitID int64) error {
	return service.editAccount(ctx, accountID, "avatar-outfit-delete", "You can only delete this account's own outfits.", func(ctx context.Context, version int64) error {
		return service.users.DeleteAvatarOutfit(ctx, accountID, version, outfitID)
	})
}
