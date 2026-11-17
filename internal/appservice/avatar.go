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

// SavedAvatar is the avatar after a save or an outfit change, or a nil Avatar
// when the change succeeded but the avatar could not be read again. SkippedAssetIDs lists the
// requested assets that Roblox did not put on the avatar, such as items that
// the account does not own. UnkeptAdjustmentAssetIDs lists the worn
// accessories whose saved placement differs from the requested one.
type SavedAvatar struct {
	Avatar                   *robloxservices.Avatar `json:"avatar"`
	SkippedAssetIDs          []int64                `json:"skippedAssetIds"`
	UnkeptAdjustmentAssetIDs []int64                `json:"unkeptAdjustmentAssetIds"`
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
	var rejectedAssetIDs []int64
	err = service.editAccount(ctx, accountID, "avatar-save", "", func(ctx context.Context, version int64) error {
		var saveError error
		rejectedAssetIDs, saveError = service.users.SaveAvatar(ctx, accountID, version, change)
		return saveError
	})
	if err != nil {
		return SavedAvatar{}, err
	}
	saved := SavedAvatar{SkippedAssetIDs: rejectedAssetIDs, UnkeptAdjustmentAssetIDs: []int64{}}
	// GetAvatar logs a failed read.
	if avatar, readError := service.GetAvatar(ctx, accountID, account.RobloxUserID); readError == nil {
		saved.Avatar = &avatar
		saved.addUnwornAndUnkeptAssets(change.Assets)
	}
	return saved, nil
}

// addUnwornAndUnkeptAssets compares the change with the saved avatar. It adds
// each requested asset that the avatar does not wear to SkippedAssetIDs, and
// each worn accessory whose placement differs from the requested one to
// UnkeptAdjustmentAssetIDs.
func (saved *SavedAvatar) addUnwornAndUnkeptAssets(changeAssets []robloxservices.AvatarChangeAsset) {
	for _, changeAsset := range changeAssets {
		wornIndex := slices.IndexFunc(saved.Avatar.Assets, func(asset robloxservices.AvatarAsset) bool { return asset.Item.ItemID == changeAsset.AssetID })
		switch {
		case wornIndex < 0:
			if !slices.Contains(saved.SkippedAssetIDs, changeAsset.AssetID) {
				saved.SkippedAssetIDs = append(saved.SkippedAssetIDs, changeAsset.AssetID)
			}
		case changeAsset.Adjustment != nil && !robloxservices.IsSameAdjustment(*changeAsset.Adjustment, saved.Avatar.Assets[wornIndex].Adjustment):
			saved.UnkeptAdjustmentAssetIDs = append(saved.UnkeptAdjustmentAssetIDs, changeAsset.AssetID)
		}
	}
}

// WaitForAvatarImage reads the full-body image of the account after a change.
// Roblox renders a changed avatar a few seconds after the change. Until then
// it answers Pending, or keeps serving the previous image, so the read repeats
// a bounded number of times until the image differs from previousImageURL. A
// change that keeps the look keeps the image, so the last image is returned
// when every read returns the previous one.
func (service *Service) WaitForAvatarImage(ctx context.Context, accountID int64, previousImageURL string) (string, error) {
	const (
		maxAttempts = 10
		retryDelay  = 2 * time.Second
	)
	account, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return "", mapRepositoryError(err)
	}
	target := robloxservices.AvatarImageTarget(account.RobloxUserID)
	latestImageURL := ""
	_, err = pollUntilReady(ctx, "avatar-image", "The avatar image was not loaded.", maxAttempts, retryDelay, func() (bool, error) {
		images, err := service.thumbnails.Images(ctx, []robloxservices.ThumbnailTarget{target}, robloxservices.LargeThumbnailSize)
		if err != nil {
			service.logs.Module("application.avatar").WarnContext(ctx, "avatar image unavailable", "operation", "avatar-image", "account_id", accountID, "error", err)
			return false, err
		}
		latestImageURL = images[target]
		return latestImageURL != "" && latestImageURL != previousImageURL, nil
	})
	if err != nil {
		return "", err
	}
	if latestImageURL != "" {
		return latestImageURL, nil
	}
	return "", &roblox.Error{Kind: roblox.KindTimeout, Endpoint: "avatar-image", Message: "Roblox is still rendering the avatar. Refresh it in a minute."}
}

// RenderAvatarPreview renders the change as a full-body image without saving
// it, so that the page can show unsaved changes.
func (service *Service) RenderAvatarPreview(ctx context.Context, accountID int64, change robloxservices.AvatarChange) (string, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "avatar-preview")
	if err != nil {
		return "", err
	}
	defer done()
	return service.renderAvatarPreview(ctx, accountID, version, "avatar-preview", change)
}

// TryOnInventoryItem renders the avatar of the account wearing the item,
// without saving it.
func (service *Service) TryOnInventoryItem(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) (string, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "avatar-try-on")
	if err != nil {
		return "", err
	}
	defer done()
	change, err := service.users.TryOnChange(ctx, accountID, version, category, itemID)
	if err != nil {
		service.logs.Module("application.avatar").WarnContext(ctx, "avatar try-on unavailable", "operation", "avatar-try-on", "account_id", accountID, "category", category, "error", err)
		return "", err
	}
	return service.renderAvatarPreview(ctx, accountID, version, "avatar-try-on", change)
}

// GetInventoryItemTryOnModel renders the avatar of the account wearing the
// item as a 3D model, without saving it.
func (service *Service) GetInventoryItemTryOnModel(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) (robloxservices.RenderedModel, error) {
	var change *robloxservices.AvatarChange
	return service.readRenderedModel(ctx, accountID, "avatar-try-on-model", func(ctx context.Context, version int64) (string, error) {
		// The avatar to render is read once, and each repeated request renders
		// the same one.
		if change == nil {
			tryOnChange, err := service.users.TryOnChange(ctx, accountID, version, category, itemID)
			if err != nil {
				return "", err
			}
			change = &tryOnChange
		}
		return service.users.RenderAvatarPreview(ctx, accountID, version, *change, robloxservices.AvatarPreviewModel)
	})
}

// renderAvatarPreview renders the change as a full-body image. Roblox answers
// Pending until the image is ready and renders it only when asked again, so
// the request repeats a bounded number of times.
func (service *Service) renderAvatarPreview(ctx context.Context, accountID, version int64, operation string, change robloxservices.AvatarChange) (string, error) {
	const (
		maxAttempts = 8
		retryDelay  = 1500 * time.Millisecond
	)
	imageURL := ""
	isRendered, err := pollUntilReady(ctx, operation, "The avatar preview was not loaded.", maxAttempts, retryDelay, func() (bool, error) {
		var err error
		imageURL, err = service.users.RenderAvatarPreview(ctx, accountID, version, change, robloxservices.AvatarPreviewImage)
		if err != nil {
			service.logs.Module("application.avatar").WarnContext(ctx, "avatar preview unavailable", "operation", operation, "account_id", accountID, "error", err)
			return false, err
		}
		return imageURL != "", nil
	})
	if err != nil {
		return "", err
	}
	if !isRendered {
		return "", &roblox.Error{Kind: roblox.KindTimeout, Endpoint: operation, Message: "Roblox is still rendering the preview. Try again in a moment."}
	}
	return imageURL, nil
}

// RedrawAvatarImage asks Roblox to render the avatar images of the account
// again, such as when an image shows an outdated look.
func (service *Service) RedrawAvatarImage(ctx context.Context, accountID int64) error {
	return service.editAccount(ctx, accountID, "avatar-redraw", "", func(ctx context.Context, version int64) error {
		return service.users.RedrawAvatarImage(ctx, accountID, version)
	})
}

// GetAvatarRenderFile downloads the file of an asset that the 3D avatar on
// the avatar page reads, in the representation that it asks for.
func (service *Service) GetAvatarRenderFile(ctx context.Context, accountID, assetID int64, representation robloxservices.AssetRepresentation) ([]byte, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "avatar-render-file")
	if err != nil {
		return nil, err
	}
	defer done()
	contents, err := service.users.AvatarRenderFile(ctx, accountID, version, assetID, representation)
	if err != nil {
		// The 3D avatar shows the rest of the avatar without the asset, so a
		// missing asset is not a warning.
		service.logs.Module("application.avatar").DebugContext(ctx, "avatar render file unavailable", "operation", "avatar-render-file", "account_id", accountID, "asset_id", assetID, "error", err)
		return nil, err
	}
	return contents, nil
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
// avatar again, so that the result shows what Roblox kept. A failed read
// returns no avatar instead of an error, because the outfit was put on.
func (service *Service) WearAvatarOutfit(ctx context.Context, accountID, outfitID int64) (SavedAvatar, error) {
	account, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return SavedAvatar{}, mapRepositoryError(err)
	}
	var outfitAssets []robloxservices.AvatarChangeAsset
	var rejectedAssetIDs []int64
	err = service.editAccount(ctx, accountID, "avatar-outfit-wear", "", func(ctx context.Context, version int64) error {
		var wearError error
		outfitAssets, rejectedAssetIDs, wearError = service.users.WearAvatarOutfit(ctx, accountID, version, outfitID)
		return wearError
	})
	if err != nil {
		return SavedAvatar{}, err
	}
	worn := SavedAvatar{SkippedAssetIDs: rejectedAssetIDs, UnkeptAdjustmentAssetIDs: []int64{}}
	// GetAvatar logs a failed read.
	if avatar, readError := service.GetAvatar(ctx, accountID, account.RobloxUserID); readError == nil {
		worn.Avatar = &avatar
		worn.addUnwornAndUnkeptAssets(outfitAssets)
	}
	return worn, nil
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
