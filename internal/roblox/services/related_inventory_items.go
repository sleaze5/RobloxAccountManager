package services

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"sync"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	maxRelatedItemCount       = 50
	maxDependencyFileBytes    = 8 << 20
	relatedItemsEndpoint      = "inventory-related-items"
	relatedItemDetailEndpoint = "inventory-related-item"
)

// RelatedInventoryItem is an asset that another item includes or uses, such as
// a part of a bundle or the mesh of an accessory.
type RelatedInventoryItem struct {
	// Category is empty for an asset type that no inventory category lists,
	// such as a mesh or an image.
	Category InventoryCategory `json:"category"`
	Item     InventoryItem     `json:"item"`
}

// hasDependencies reports whether the assets of the category are model files
// that name other assets, such as the mesh and the texture of an accessory.
func hasDependencies(category InventoryCategory) bool {
	switch category {
	case InventoryCollectibles, InventoryAccessories, InventoryClassicClothing, InventoryLayeredClothing,
		InventoryHeadsAndBodies, InventoryFaces, InventoryGear, InventoryModels, InventoryDecals:
		return true
	default:
		return false
	}
}

// RelatedInventoryItems lists the assets of a bundle, or the assets that an
// asset uses, with their names, types, creators, and descriptions. Other kinds
// of items relate to none.
func (service *Users) RelatedInventoryItems(ctx context.Context, accountID, version int64, category InventoryCategory, itemID int64) ([]RelatedInventoryItem, error) {
	var (
		assetIDs []int64
		err      error
	)
	switch {
	case category == InventoryBundles:
		assetIDs, err = service.bundleAssetIDs(ctx, accountID, version, itemID)
	case hasDependencies(category):
		assetIDs, err = service.assetDependencyIDs(ctx, accountID, version, itemID)
	}
	if err != nil {
		return nil, err
	}
	return service.relatedItemsOf(ctx, accountID, version, assetIDs)
}

// bundleAssetIDs lists the assets of the bundle. A bundle also lists the
// outfit that shows it, which is not an asset.
func (service *Users) bundleAssetIDs(ctx context.Context, accountID, version, bundleID int64) ([]int64, error) {
	payload, err := service.readBundleDetails(ctx, accountID, version, relatedItemsEndpoint, strconv.FormatInt(bundleID, 10))
	if err != nil {
		return nil, err
	}
	assetIDs := make([]int64, 0, len(payload.BundledItems))
	for _, bundledItem := range payload.BundledItems {
		if bundledItem.Type == "Asset" && bundledItem.ID > 0 && len(assetIDs) < maxRelatedItemCount {
			assetIDs = append(assetIDs, bundledItem.ID)
		}
	}
	return assetIDs, nil
}

// assetDependencyIDs downloads the model file of the asset and lists the
// assets that it names. The asset delivery API answers with the address of the
// file on the CDN. An asset that the account may not download has no
// dependencies that it can see.
func (service *Users) assetDependencyIDs(ctx context.Context, accountID, version, assetID int64) ([]int64, error) {
	var payload struct {
		Locations []struct {
			Location string `json:"location"`
		} `json:"locations"`
	}
	err := service.readAccountDetails(ctx, accountID, version, relatedItemsEndpoint, "https://assetdelivery.roblox.com/v2/asset/?id="+strconv.FormatInt(assetID, 10), &payload)
	if remote, ok := errors.AsType[*roblox.Error](err); ok && (remote.Status == http.StatusForbidden || remote.Status == http.StatusNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(payload.Locations) == 0 {
		return nil, nil
	}
	modelFile, err := service.downloadCDNFile(ctx, payload.Locations[0].Location, maxDependencyFileBytes)
	if err != nil {
		return nil, err
	}
	assetIDs, err := assetIDsReferencedByModelFile(modelFile)
	if err != nil {
		return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: relatedItemsEndpoint, Message: "Roblox returned a file that couldn't be read.", Cause: err}
	}
	return slices.DeleteFunc(assetIDs, func(referencedAssetID int64) bool { return referencedAssetID == assetID }), nil
}

// relatedItemsOf reads the economy details of each asset at once. An asset
// whose details fail is left out, unless every one fails.
func (service *Users) relatedItemsOf(ctx context.Context, accountID, version int64, assetIDs []int64) ([]RelatedInventoryItem, error) {
	relatedItems := make([]*RelatedInventoryItem, len(assetIDs))
	failures := make([]error, len(assetIDs))
	var group sync.WaitGroup
	for index, assetID := range assetIDs {
		group.Go(func() {
			relatedItems[index], failures[index] = service.relatedItemOf(ctx, accountID, version, assetID)
		})
	}
	group.Wait()
	readItems := make([]RelatedInventoryItem, 0, len(assetIDs))
	for _, relatedItem := range relatedItems {
		if relatedItem != nil {
			readItems = append(readItems, *relatedItem)
		}
	}
	if len(readItems) == 0 && len(assetIDs) > 0 {
		// The first failure explains the others, which fail the same way.
		return nil, failures[0]
	}
	return readItems, nil
}

func (service *Users) relatedItemOf(ctx context.Context, accountID, version, assetID int64) (*RelatedInventoryItem, error) {
	var payload struct {
		Name        string `json:"Name"`
		Description string `json:"Description"`
		AssetTypeID int64  `json:"AssetTypeId"`
		Creator     struct {
			Name             string `json:"Name"`
			CreatorType      string `json:"CreatorType"`
			CreatorTargetID  int64  `json:"CreatorTargetId"`
			HasVerifiedBadge bool   `json:"HasVerifiedBadge"`
		} `json:"Creator"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, relatedItemDetailEndpoint, "https://economy.roblox.com/v2/assets/"+strconv.FormatInt(assetID, 10)+"/details", &payload); err != nil {
		return nil, err
	}
	typeName := assetTypeNameByID[payload.AssetTypeID]
	item := InventoryItem{
		ItemID: assetID, Name: boundedText(payload.Name, maxInventoryItemNameLength), TypeName: typeName,
		Description:             boundedText(payload.Description, maxInventoryDescriptionLength),
		CreatorName:             boundedText(payload.Creator.Name, maxInventoryItemNameLength),
		CreatorHasVerifiedBadge: payload.Creator.HasVerifiedBadge,
		thumbnailType:           "Asset", CatalogItemType: CatalogAsset,
	}
	item.setCreator(payload.Creator.CreatorTargetID, payload.Creator.CreatorType)
	return &RelatedInventoryItem{Category: inventoryCategoryOfAssetType(typeName), Item: item}, nil
}
