package appservice

import (
	"context"
	"log/slog"

	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// GetInventoryPage reads one page of what the user owns in the category, as
// the account sees it, with item images and catalog details. A missing image
// or catalog detail leaves the item without it.
func (service *Service) GetInventoryPage(ctx context.Context, accountID, userID int64, category robloxservices.InventoryCategory, cursor string) (robloxservices.InventoryPage, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "inventory")
	if err != nil {
		return robloxservices.InventoryPage{}, err
	}
	defer done()
	logger := service.logs.Module("application.inventory").With("operation", "inventory", "account_id", accountID, "category", category)
	page, err := service.users.InventoryPage(ctx, accountID, version, userID, category, cursor)
	if err != nil {
		logger.WarnContext(ctx, "inventory page unavailable", "error", err)
		return robloxservices.InventoryPage{}, err
	}
	service.addCatalogDetailsAndImages(ctx, accountID, version, page.Items, logger)
	return page, nil
}

// addCatalogDetailsAndImages adds the catalog details and the small image of
// each item. A missing detail or image leaves the item without it.
func (service *Service) addCatalogDetailsAndImages(ctx context.Context, accountID, version int64, items []robloxservices.InventoryItem, logger *slog.Logger) {
	if err := service.users.AddCatalogDetails(ctx, accountID, version, items); err != nil {
		logger.WarnContext(ctx, "inventory catalog details unavailable", "error", err)
	}
	service.addItemImages(ctx, items, logger)
}

// addItemImages adds the small image of each item. A missing image leaves the
// item without it.
func (service *Service) addItemImages(ctx context.Context, items []robloxservices.InventoryItem, logger *slog.Logger) {
	targets := make([]robloxservices.ThumbnailTarget, len(items))
	for index, item := range items {
		targets[index] = item.ThumbnailTarget()
	}
	images, err := service.thumbnails.Images(ctx, targets, robloxservices.SmallThumbnailSize)
	if err != nil {
		logger.WarnContext(ctx, "item images unavailable", "error", err)
	}
	for index := range items {
		items[index].ImageURL = images[targets[index]]
	}
}

// GetInventoryRelatedItems lists the assets that the item includes or uses,
// such as the parts of a bundle or the mesh and texture of an accessory.
func (service *Service) GetInventoryRelatedItems(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) ([]robloxservices.CategorizedInventoryItem, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "inventory-related-items")
	if err != nil {
		return nil, err
	}
	defer done()
	logger := service.logs.Module("application.inventory").With("operation", "inventory-related-items", "account_id", accountID, "category", category)
	relatedItems, err := service.users.RelatedInventoryItems(ctx, accountID, version, category, itemID)
	if err != nil {
		logger.WarnContext(ctx, "related items unavailable", "error", err)
		return nil, err
	}
	items := make([]robloxservices.InventoryItem, len(relatedItems))
	for index, relatedItem := range relatedItems {
		items[index] = relatedItem.Item
	}
	service.addCatalogDetailsAndImages(ctx, accountID, version, items, logger)
	for index := range relatedItems {
		relatedItems[index].Item = items[index]
	}
	return relatedItems, nil
}

// GetInventoryAssetItem reads one asset as an item, such as a mesh that the
// explorer shows, so that its details can open.
func (service *Service) GetInventoryAssetItem(ctx context.Context, accountID, assetID int64) (robloxservices.CategorizedInventoryItem, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "inventory-asset-item")
	if err != nil {
		return robloxservices.CategorizedInventoryItem{}, err
	}
	defer done()
	logger := service.logs.Module("application.inventory").With("operation", "inventory-asset-item", "account_id", accountID, "asset_id", assetID)
	assetItem, err := service.users.AssetItem(ctx, accountID, version, assetID)
	if err != nil {
		logger.WarnContext(ctx, "asset item unavailable", "error", err)
		return robloxservices.CategorizedInventoryItem{}, err
	}
	items := []robloxservices.InventoryItem{assetItem.Item}
	service.addCatalogDetailsAndImages(ctx, accountID, version, items, logger)
	assetItem.Item = items[0]
	return *assetItem, nil
}

// GetInventoryItemModel reads the 3D model of the item, as the account sees
// it.
func (service *Service) GetInventoryItemModel(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) (robloxservices.RenderedModel, error) {
	return service.readRenderedModel(ctx, accountID, "inventory-item-model", func(ctx context.Context, version int64) (string, error) {
		return service.users.InventoryItemModelManifestURL(ctx, accountID, version, category, itemID)
	})
}

// GetInventoryItemLook reads how the 3D viewer shows the item.
func (service *Service) GetInventoryItemLook(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) (robloxservices.InventoryItemLook, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "inventory-item-look")
	if err != nil {
		return robloxservices.InventoryItemLook{}, err
	}
	defer done()
	look, err := service.users.InventoryItemLook(ctx, accountID, version, category, itemID)
	if err != nil {
		// The viewer shows the model that Roblox renders instead.
		service.logs.Module("application.inventory").WarnContext(ctx, "inventory item look unavailable", "operation", "inventory-item-look", "account_id", accountID, "category", category, "error", err)
		return robloxservices.InventoryItemLook{}, err
	}
	return look, nil
}

// GetInventoryItemDetails reads the large image, dates, and collectible IDs
// of one item. A missing large image leaves the details without it.
func (service *Service) GetInventoryItemDetails(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) (robloxservices.InventoryItemDetails, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "inventory-item-details")
	if err != nil {
		return robloxservices.InventoryItemDetails{}, err
	}
	defer done()
	logger := service.logs.Module("application.inventory").With("operation", "inventory-item-details", "account_id", accountID, "category", category)
	details, err := service.users.InventoryItemDetails(ctx, accountID, version, category, itemID)
	if err != nil {
		logger.WarnContext(ctx, "item details unavailable", "error", err)
		return robloxservices.InventoryItemDetails{}, err
	}
	if details.Availability == robloxservices.ItemDeleted {
		return details, nil
	}
	target, isLargeImage := robloxservices.LargeImageTarget(category, itemID)
	size := robloxservices.LargeThumbnailSize
	if !isLargeImage {
		var hasIcon bool
		if target, hasIcon = robloxservices.IconTarget(category, itemID); !hasIcon {
			return details, nil
		}
		size = robloxservices.SmallThumbnailSize
	}
	thumbnails, err := service.thumbnails.Thumbnails(ctx, []robloxservices.ThumbnailTarget{target}, size)
	if err != nil {
		logger.WarnContext(ctx, "item image unavailable", "error", err)
	}
	// Roblox blocks the image of an item that it moderated, even while its
	// details API still lists the item.
	if thumbnails[target].IsBlocked {
		details.MarkUnavailable(robloxservices.ItemModerated)
	}
	if isLargeImage {
		details.LargeImageURL = thumbnails[target].ImageURL
	}
	return details, nil
}

func (service *Service) IsInventoryItemOwned(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) (bool, error) {
	account, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return false, mapRepositoryError(err)
	}
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "inventory-item-ownership")
	if err != nil {
		return false, err
	}
	defer done()
	isOwned, err := service.users.IsInventoryItemOwned(ctx, accountID, version, account.RobloxUserID, category, itemID)
	if err != nil {
		service.logs.Module("application.inventory").WarnContext(ctx, "item ownership unavailable", "operation", "inventory-item-ownership", "account_id", accountID, "category", category, "error", err)
		return false, err
	}
	return isOwned, nil
}

// IsInventoryItemFavorite reads whether the account favorited the catalog item.
func (service *Service) IsInventoryItemFavorite(ctx context.Context, accountID int64, itemType robloxservices.CatalogItemType, itemID int64) (bool, error) {
	if !robloxservices.ValidCatalogItemType(itemType) {
		return false, accountEditError("item-favorite", "Only catalog items can be favorites.")
	}
	account, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return false, mapRepositoryError(err)
	}
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "item-favorite")
	if err != nil {
		return false, err
	}
	defer done()
	isFavorite, err := service.users.IsItemFavorite(ctx, accountID, version, account.RobloxUserID, itemType, itemID)
	if err != nil {
		service.logs.Module("application.inventory").WarnContext(ctx, "item favorite unavailable", "operation", "item-favorite", "account_id", accountID, "error", err)
		return false, err
	}
	return isFavorite, nil
}

func (service *Service) SetInventoryItemFavorite(ctx context.Context, accountID int64, itemType robloxservices.CatalogItemType, itemID int64, isFavorite bool) error {
	if !robloxservices.ValidCatalogItemType(itemType) {
		return accountEditError("item-favorite-update", "Only catalog items can be favorites.")
	}
	account, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return mapRepositoryError(err)
	}
	return service.editAccount(ctx, accountID, "item-favorite-update", "", func(ctx context.Context, version int64) error {
		return service.users.SetItemFavorite(ctx, accountID, version, account.RobloxUserID, itemType, itemID, isFavorite)
	})
}

// BuyInventoryItem buys the item for the account at the price that the user
// confirmed, and reports whether Roblox still delivers it.
func (service *Service) BuyInventoryItem(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID, expectedPriceInRobux int64) (bool, error) {
	if expectedPriceInRobux < 0 {
		return false, accountEditError("inventory-item-purchase", "The price of this item is invalid.")
	}
	account, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return false, mapRepositoryError(err)
	}
	var isPending bool
	err = service.editAccount(ctx, accountID, "inventory-item-purchase", "", func(ctx context.Context, version int64) error {
		var purchaseError error
		isPending, purchaseError = service.users.BuyInventoryItem(ctx, accountID, version, account.RobloxUserID, category, itemID, expectedPriceInRobux)
		return purchaseError
	})
	return isPending, err
}

func (service *Service) DeleteInventoryAsset(ctx context.Context, accountID, assetID int64) error {
	return service.editAccount(ctx, accountID, "inventory-asset-delete", "Roblox does not allow deleting this item.", func(ctx context.Context, version int64) error {
		return service.users.DeleteInventoryAsset(ctx, accountID, version, assetID)
	})
}

func (service *Service) GetWornAssetIDs(ctx context.Context, accountID, userID int64) ([]int64, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "inventory-worn-assets")
	if err != nil {
		return nil, err
	}
	defer done()
	assetIDs, err := service.users.WornAssetIDs(ctx, accountID, version, userID)
	if err != nil {
		service.logs.Module("application.inventory").WarnContext(ctx, "worn assets unavailable", "operation", "inventory-worn-assets", "account_id", accountID, "error", err)
		return nil, err
	}
	return assetIDs, nil
}
