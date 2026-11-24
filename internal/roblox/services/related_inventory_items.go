package services

import (
	"context"
	"slices"
	"strconv"
	"sync"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox/modelfile"
)

const (
	maxRelatedItemCount    = 50
	maxDependencyFileBytes = 8 << 20
	relatedItemsEndpoint   = "inventory-related-items"
	assetItemEndpoint      = "inventory-asset-item"
)

// hasDependencies reports whether the assets of the category are model files
// that name other assets, such as the mesh and the texture of an accessory.
// Images, meshes, and decals are the assets that others use.
func hasDependencies(category InventoryCategory) bool {
	switch category {
	case InventoryCollectibles, InventoryAccessories, InventoryClassicClothing, InventoryLayeredClothing,
		InventoryHeadsAndBodies, InventoryFaces, InventoryGear, InventoryModels:
		return true
	default:
		return false
	}
}

// RelatedInventoryItems lists the assets of a bundle, or the assets that an
// asset uses, with their names, types, creators, and descriptions. Other kinds
// of items relate to none.
func (service *Users) RelatedInventoryItems(ctx context.Context, accountID, version int64, category InventoryCategory, itemID int64) ([]CategorizedInventoryItem, error) {
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
// assets that it names. An asset that the account may not download has no
// dependencies that it can see.
func (service *Users) assetDependencyIDs(ctx context.Context, accountID, version, assetID int64) ([]int64, error) {
	modelFile, _, err := service.downloadAssetFile(ctx, accountID, version, relatedItemsEndpoint, assetID, AssetRepresentation{}, maxDependencyFileBytes)
	if _, isRefused := assetFileRefusal(err); isRefused {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	assetIDs, err := modelfile.ReferencedAssetIDs(modelFile)
	if err != nil {
		return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: relatedItemsEndpoint, Message: "Roblox returned a file that the app cannot read.", Cause: err}
	}
	return slices.DeleteFunc(assetIDs, func(referencedAssetID int64) bool { return referencedAssetID == assetID }), nil
}

// relatedItemsOf reads the economy details of each asset at once. An asset
// whose details fail is left out, unless every one fails.
func (service *Users) relatedItemsOf(ctx context.Context, accountID, version int64, assetIDs []int64) ([]CategorizedInventoryItem, error) {
	relatedItems := make([]*CategorizedInventoryItem, len(assetIDs))
	failures := make([]error, len(assetIDs))
	var group sync.WaitGroup
	for index, assetID := range assetIDs {
		group.Go(func() {
			relatedItems[index], failures[index] = service.AssetItem(ctx, accountID, version, assetID)
		})
	}
	group.Wait()
	readItems := make([]CategorizedInventoryItem, 0, len(assetIDs))
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

// AssetItem reads the economy details of one asset as an item, such as a mesh
// that a model file names.
func (service *Users) AssetItem(ctx context.Context, accountID, version, assetID int64) (*CategorizedInventoryItem, error) {
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
	if err := service.readAccountDetails(ctx, accountID, version, assetItemEndpoint, "https://economy.roblox.com/v2/assets/"+strconv.FormatInt(assetID, 10)+"/details", &payload); err != nil {
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
	return &CategorizedInventoryItem{Category: inventoryCategoryOfAssetTypeOrOtherAssets(typeName), Item: item}, nil
}
