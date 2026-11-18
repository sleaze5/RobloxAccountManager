package services

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

// InventoryItemDetails is what the item dialog shows beyond the inventory
// page. A timestamp is 0 and an ID is empty when Roblox does not list it for
// the kind of item.
type InventoryItemDetails struct {
	// LargeImageURL is empty for an item that Roblox renders only small.
	LargeImageURL        string `json:"largeImageUrl"`
	CreatedAtMs          int64  `json:"createdAtMs"`
	UpdatedAtMs          int64  `json:"updatedAtMs"`
	CollectibleItemID    string `json:"collectibleItemId"`
	CollectibleProductID string `json:"collectibleProductId"`
	// AwardingGameName names the game that awards a badge.
	AwardingGameName string `json:"awardingGameName"`
	HasModelView     bool   `json:"hasModelView"`
	HasRenderedModel bool   `json:"hasRenderedModel"`
	// CanTryOn reports whether an avatar can wear the item, so that a preview
	// can show it on the avatar of the account.
	CanTryOn bool `json:"canTryOn"`
	// CanHaveRelatedItems reports whether the item can include or use other
	// assets, as a bundle or a model file does.
	CanHaveRelatedItems bool `json:"canHaveRelatedItems"`
	// CanDownload reports whether the account can save the file of the item.
	CanDownload bool `json:"canDownload"`
	// Availability tells whether Roblox moderated or deleted the item.
	Availability ItemAvailability `json:"availability"`
}

// ItemAvailability tells whether Roblox still has an item.
type ItemAvailability string

const (
	ItemAvailable ItemAvailability = "available"
	// ItemModerated is an item that Roblox moderated and deleted the content
	// of, whose name and dates remain.
	ItemModerated ItemAvailability = "moderated"
	// ItemDeleted is an asset that Roblox lists nowhere anymore.
	ItemDeleted ItemAvailability = "deleted"
)

// MarkUnavailable removes the actions that need an item's content after Roblox
// moderates or deletes it.
func (details *InventoryItemDetails) MarkUnavailable(availability ItemAvailability) {
	details.Availability = availability
	details.HasModelView = false
	details.HasRenderedModel = false
	details.CanTryOn = false
	details.CanHaveRelatedItems = false
	details.CanDownload = false
}

// inventoryItemType is the item type that the inventory ownership API takes.
type inventoryItemType string

const (
	inventoryItemAsset    inventoryItemType = "Asset"
	inventoryItemBundle   inventoryItemType = "Bundle"
	inventoryItemBadge    inventoryItemType = "Badge"
	inventoryItemGamePass inventoryItemType = "GamePass"
)

func inventoryItemTypeOf(category InventoryCategory) (inventoryItemType, bool) {
	switch category {
	case InventoryBadges:
		return inventoryItemBadge, true
	case InventoryGamePasses:
		return inventoryItemGamePass, true
	case InventoryBundles:
		return inventoryItemBundle, true
	case InventoryCollectibles, InventoryOtherAssets:
		return inventoryItemAsset, true
	default:
		_, ok := inventoryAssetTypesByCategory[category]
		return inventoryItemAsset, ok
	}
}

// LargeImageTarget names the large image of an asset or a bundle. Roblox
// renders badges and game passes only small. Verified on 2026-11-14.
func LargeImageTarget(category InventoryCategory, itemID int64) (ThumbnailTarget, bool) {
	itemType, ok := inventoryItemTypeOf(category)
	switch {
	case !ok:
		return ThumbnailTarget{}, false
	case itemType == inventoryItemAsset:
		return ThumbnailTarget{Type: "Asset", TargetID: itemID}, true
	case itemType == inventoryItemBundle:
		return ThumbnailTarget{Type: "BundleThumbnail", TargetID: itemID}, true
	}
	return ThumbnailTarget{}, false
}

// IconTarget names the icon of a badge or a game pass, which Roblox renders
// only small.
func IconTarget(category InventoryCategory, itemID int64) (ThumbnailTarget, bool) {
	switch category {
	case InventoryBadges:
		return ThumbnailTarget{Type: "BadgeIcon", TargetID: itemID}, true
	case InventoryGamePasses:
		return ThumbnailTarget{Type: "GamePass", TargetID: itemID}, true
	default:
		return ThumbnailTarget{}, false
	}
}

// InventoryItemDetails reads the dates and collectible IDs of one item. Each
// kind of item has its own endpoint. Verified against the API on 2026-11-14.
func (service *Users) InventoryItemDetails(ctx context.Context, accountID, version int64, category InventoryCategory, itemID int64) (InventoryItemDetails, error) {
	itemType, ok := inventoryItemTypeOf(category)
	if !ok {
		return InventoryItemDetails{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "inventory-item-details", Message: "Select an inventory category."}
	}
	itemIDText := strconv.FormatInt(itemID, 10)
	var (
		details InventoryItemDetails
		err     error
	)
	switch itemType {
	case inventoryItemBadge:
		details, err = service.badgeItemDetails(ctx, accountID, version, itemIDText)
	case inventoryItemGamePass:
		details, err = service.gamePassItemDetails(ctx, accountID, version, itemIDText)
	case inventoryItemBundle:
		details, err = service.bundleItemDetails(ctx, accountID, version, itemIDText)
	default:
		details, err = service.assetItemDetails(ctx, accountID, version, itemIDText)
	}
	if err != nil {
		return InventoryItemDetails{}, err
	}
	// Only the details of an asset tell that Roblox moderated or deleted it.
	if details.Availability == "" {
		details.Availability = ItemAvailable
	}
	details.HasModelView = hasModelView(category)
	details.HasRenderedModel = hasRenderedModel(category)
	details.CanTryOn = canTryOn(category)
	details.CanHaveRelatedItems = category == InventoryBundles || hasDependencies(category)
	details.CanDownload = canDownload(category)
	if details.Availability != ItemAvailable {
		details.MarkUnavailable(details.Availability)
	}
	return details, nil
}

// canTryOn reports whether the items of the category are assets that an
// avatar wears, other than animations, which a still image cannot show.
func canTryOn(category InventoryCategory) bool {
	switch category {
	case InventoryCollectibles, InventoryAccessories, InventoryClassicClothing, InventoryLayeredClothing,
		InventoryHeadsAndBodies, InventoryFaces, InventoryGear:
		return true
	default:
		return false
	}
}

// hasModelView reports whether the 3D viewer shows the items of the category:
// every item that Roblox renders as a model, classic clothing on a plain
// avatar, and animations and emotes played on the avatar of the account.
func hasModelView(category InventoryCategory) bool {
	return hasRenderedModel(category) || category == InventoryClassicClothing ||
		category == InventoryAnimations || category == InventoryEmotes
}

// hasRenderedModel reports whether the model can come from Roblox. For
// bundles, the model comes from the outfit that the bundle lists.
func hasRenderedModel(category InventoryCategory) bool {
	switch category {
	case InventoryCollectibles, InventoryAccessories, InventoryLayeredClothing,
		InventoryHeadsAndBodies, InventoryFaces, InventoryBundles, InventoryGear, InventoryModels:
		return true
	default:
		return false
	}
}

// InventoryItemModelManifestURL reads the address of the 3D model manifest of
// the item, or an empty address while Roblox renders the model.
func (service *Users) InventoryItemModelManifestURL(ctx context.Context, accountID, version int64, category InventoryCategory, itemID int64) (string, error) {
	const endpoint = "inventory-item-model"
	if !hasRenderedModel(category) {
		return "", &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Roblox cannot show this item in 3D."}
	}
	if category != InventoryBundles {
		return service.AssetModelManifestURL(ctx, accountID, version, itemID)
	}
	payload, err := service.readBundleDetails(ctx, accountID, version, endpoint, strconv.FormatInt(itemID, 10))
	if err != nil {
		return "", err
	}
	outfitID, hasOutfit := payload.outfitID()
	if !hasOutfit {
		return "", &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Roblox cannot show this bundle in 3D."}
	}
	return service.renderedModelManifestURL(ctx, accountID, version, endpoint, "https://thumbnails.roblox.com/v1/users/outfit-3d?outfitId="+strconv.FormatInt(outfitID, 10))
}

// assetItemDetails reads the economy details, which Roblox lists for every
// asset, including places, models, and decals.
func (service *Users) assetItemDetails(ctx context.Context, accountID, version int64, assetID string) (InventoryItemDetails, error) {
	var payload struct {
		Created              string `json:"Created"`
		Updated              string `json:"Updated"`
		CollectibleItemID    string `json:"CollectibleItemId"`
		CollectibleProductID string `json:"CollectibleProductId"`
	}
	err := service.readAccountDetails(ctx, accountID, version, "inventory-asset-details", "https://economy.roblox.com/v2/assets/"+assetID+"/details", &payload)
	if remote, isRemote := errors.AsType[*roblox.Error](err); isRemote && (remote.Status == http.StatusBadRequest || remote.Status == http.StatusNotFound) {
		return service.developAssetItemDetails(ctx, accountID, version, assetID, err)
	}
	if err != nil {
		return InventoryItemDetails{}, err
	}
	return InventoryItemDetails{
		CreatedAtMs: robloxTimeMs(payload.Created), UpdatedAtMs: robloxTimeMs(payload.Updated),
		CollectibleItemID:    boundedText(payload.CollectibleItemID, maxInventoryItemNameLength),
		CollectibleProductID: boundedText(payload.CollectibleProductID, maxInventoryItemNameLength),
	}, nil
}

// developAssetItemDetails reads the details of an asset that the economy API
// no longer lists from the develop API, which still lists a hidden or
// moderated asset, as RoSeal reads it. An asset that neither lists is
// deleted. economyError is returned when the develop API fails too.
func (service *Users) developAssetItemDetails(ctx context.Context, accountID, version int64, assetID string, economyError error) (InventoryItemDetails, error) {
	var payload struct {
		Data []struct {
			ID          int64  `json:"id"`
			Created     string `json:"created"`
			Updated     string `json:"updated"`
			IsModerated bool   `json:"isModerated"`
		} `json:"data"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "inventory-asset-develop-details", "https://develop.roblox.com/v1/assets?assetIds="+assetID, &payload); err != nil {
		return InventoryItemDetails{}, economyError
	}
	for _, listedAsset := range payload.Data {
		if strconv.FormatInt(listedAsset.ID, 10) != assetID {
			continue
		}
		availability := ItemAvailable
		if listedAsset.IsModerated {
			availability = ItemModerated
		}
		return InventoryItemDetails{
			CreatedAtMs: robloxTimeMs(listedAsset.Created), UpdatedAtMs: robloxTimeMs(listedAsset.Updated),
			Availability: availability,
		}, nil
	}
	return InventoryItemDetails{Availability: ItemDeleted}, nil
}

// bundleDetailsPayload is a bundle as the catalog details API lists it. The
// bundled items are assets, and the UserOutfit that shows the bundle.
type bundleDetailsPayload struct {
	ItemCreatedUtc    string `json:"itemCreatedUtc"`
	CollectibleItemID string `json:"collectibleItemId"`
	BundledItems      []struct {
		ID   int64  `json:"id"`
		Type string `json:"type"`
	} `json:"bundledItems"`
}

// outfitID finds the UserOutfit that Roblox lists to show the bundle.
func (payload bundleDetailsPayload) outfitID() (int64, bool) {
	for _, bundledItem := range payload.BundledItems {
		if bundledItem.Type == "UserOutfit" && bundledItem.ID > 0 {
			return bundledItem.ID, true
		}
	}
	return 0, false
}

func (service *Users) readBundleDetails(ctx context.Context, accountID, version int64, endpoint, bundleID string) (bundleDetailsPayload, error) {
	var payload bundleDetailsPayload
	err := service.readAccountDetails(ctx, accountID, version, endpoint, "https://catalog.roblox.com/v1/catalog/items/"+bundleID+"/details?itemType=Bundle", &payload)
	return payload, err
}

// bundleItemDetails reads the catalog details of the bundle, which list no
// update time.
func (service *Users) bundleItemDetails(ctx context.Context, accountID, version int64, bundleID string) (InventoryItemDetails, error) {
	payload, err := service.readBundleDetails(ctx, accountID, version, "inventory-bundle-details", bundleID)
	if err != nil {
		return InventoryItemDetails{}, err
	}
	return InventoryItemDetails{
		CreatedAtMs:       robloxTimeMs(payload.ItemCreatedUtc),
		CollectibleItemID: boundedText(payload.CollectibleItemID, maxInventoryItemNameLength),
	}, nil
}

func (service *Users) badgeItemDetails(ctx context.Context, accountID, version int64, badgeID string) (InventoryItemDetails, error) {
	var payload struct {
		Created          string `json:"created"`
		Updated          string `json:"updated"`
		AwardingUniverse struct {
			Name string `json:"name"`
		} `json:"awardingUniverse"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "inventory-badge-details", "https://badges.roblox.com/v1/badges/"+badgeID, &payload); err != nil {
		return InventoryItemDetails{}, err
	}
	return InventoryItemDetails{
		CreatedAtMs: robloxTimeMs(payload.Created), UpdatedAtMs: robloxTimeMs(payload.Updated),
		AwardingGameName: boundedText(payload.AwardingUniverse.Name, maxInventoryItemNameLength),
	}, nil
}

func (service *Users) gamePassItemDetails(ctx context.Context, accountID, version int64, gamePassID string) (InventoryItemDetails, error) {
	var payload struct {
		Created string `json:"Created"`
		Updated string `json:"Updated"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "inventory-game-pass-details", "https://apis.roblox.com/game-passes/v1/game-passes/"+gamePassID+"/product-info", &payload); err != nil {
		return InventoryItemDetails{}, err
	}
	return InventoryItemDetails{CreatedAtMs: robloxTimeMs(payload.Created), UpdatedAtMs: robloxTimeMs(payload.Updated)}, nil
}

// IsInventoryItemOwned reads whether the user owns the item. Roblox answers
// a bare true or false. Verified on 2026-11-14.
func (service *Users) IsInventoryItemOwned(ctx context.Context, accountID, version, userID int64, category InventoryCategory, itemID int64) (bool, error) {
	itemType, ok := inventoryItemTypeOf(category)
	if !ok {
		return false, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "inventory-item-ownership", Message: "Select an inventory category."}
	}
	var isOwned bool
	address := "https://inventory.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/items/" + string(itemType) + "/" + strconv.FormatInt(itemID, 10) + "/is-owned"
	if err := service.readAccountDetails(ctx, accountID, version, "inventory-item-ownership", address, &isOwned); err != nil {
		return false, err
	}
	return isOwned, nil
}
