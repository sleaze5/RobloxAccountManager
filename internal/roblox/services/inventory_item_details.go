package services

import (
	"context"
	"strconv"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const maxIncludedItemCount = 50

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
	// IncludedItemNames lists the assets of a bundle.
	IncludedItemNames []string `json:"includedItemNames"`
	// AwardingGameName names the game that awards a badge.
	AwardingGameName string `json:"awardingGameName"`
	// HasModel reports whether Roblox renders the item as a 3D model.
	HasModel bool `json:"hasModel"`
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
	case InventoryCollectibles:
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

// InventoryItemDetails reads the dates and collectible IDs of one item. Each
// kind of item has its own endpoint. Verified against the API on 2026-11-14.
func (service *Users) InventoryItemDetails(ctx context.Context, accountID, version int64, category InventoryCategory, itemID int64) (InventoryItemDetails, error) {
	itemType, ok := inventoryItemTypeOf(category)
	if !ok {
		return InventoryItemDetails{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "inventory-item-details", Message: "Choose an inventory category."}
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
	details.HasModel = hasRenderedModel(category)
	return details, nil
}

// hasRenderedModel reports whether Roblox renders the items of the category as
// 3D models: the items that a character wears or holds, and models. Roblox
// renders no model of a bundle, so a bundle shows the outfit that it lists.
func hasRenderedModel(category InventoryCategory) bool {
	switch category {
	case InventoryCollectibles, InventoryAccessories, InventoryClassicClothing, InventoryLayeredClothing,
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
		return "", &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Roblox can't show this item in 3D."}
	}
	if category != InventoryBundles {
		return service.renderedModelManifestURL(ctx, accountID, version, endpoint, "https://thumbnails.roblox.com/v1/assets-thumbnail-3d?assetId="+strconv.FormatInt(itemID, 10))
	}
	payload, err := service.readBundleDetails(ctx, accountID, version, endpoint, strconv.FormatInt(itemID, 10))
	if err != nil {
		return "", err
	}
	for _, bundledItem := range payload.BundledItems {
		if bundledItem.Type == "UserOutfit" && bundledItem.ID > 0 {
			return service.renderedModelManifestURL(ctx, accountID, version, endpoint, "https://thumbnails.roblox.com/v1/users/outfit-3d?outfitId="+strconv.FormatInt(bundledItem.ID, 10))
		}
	}
	return "", &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Roblox can't show this bundle in 3D."}
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
	if err := service.readAccountDetails(ctx, accountID, version, "inventory-asset-details", "https://economy.roblox.com/v2/assets/"+assetID+"/details", &payload); err != nil {
		return InventoryItemDetails{}, err
	}
	return InventoryItemDetails{
		CreatedAtMs: robloxTimeMs(payload.Created), UpdatedAtMs: robloxTimeMs(payload.Updated),
		CollectibleItemID:    boundedText(payload.CollectibleItemID, maxInventoryItemNameLength),
		CollectibleProductID: boundedText(payload.CollectibleProductID, maxInventoryItemNameLength),
	}, nil
}

// bundleDetailsPayload is a bundle as the catalog details API lists it. The
// bundled items are assets, and the UserOutfit that shows the bundle.
type bundleDetailsPayload struct {
	ItemCreatedUtc    string `json:"itemCreatedUtc"`
	CollectibleItemID string `json:"collectibleItemId"`
	BundledItems      []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"bundledItems"`
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
	details := InventoryItemDetails{
		CreatedAtMs:       robloxTimeMs(payload.ItemCreatedUtc),
		CollectibleItemID: boundedText(payload.CollectibleItemID, maxInventoryItemNameLength),
		IncludedItemNames: []string{},
	}
	// A bundle also lists the outfit that shows it, which is not an item.
	for _, bundledItem := range payload.BundledItems {
		if bundledItem.Type == "Asset" && bundledItem.Name != "" && len(details.IncludedItemNames) < maxIncludedItemCount {
			details.IncludedItemNames = append(details.IncludedItemNames, boundedText(bundledItem.Name, maxInventoryItemNameLength))
		}
	}
	return details, nil
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
		return false, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "inventory-item-ownership", Message: "Choose an inventory category."}
	}
	var isOwned bool
	address := "https://inventory.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/items/" + string(itemType) + "/" + strconv.FormatInt(itemID, 10) + "/is-owned"
	if err := service.readAccountDetails(ctx, accountID, version, "inventory-item-ownership", address, &isOwned); err != nil {
		return false, err
	}
	return isOwned, nil
}
