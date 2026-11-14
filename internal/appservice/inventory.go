package appservice

import (
	"context"

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
	if err := service.users.AddCatalogDetails(ctx, accountID, version, page.Items); err != nil {
		logger.WarnContext(ctx, "inventory catalog details unavailable", "error", err)
	}
	targets := make([]robloxservices.ThumbnailTarget, len(page.Items))
	for index, item := range page.Items {
		targets[index] = item.ThumbnailTarget()
	}
	images, err := service.thumbnails.Images(ctx, targets, robloxservices.SmallThumbnailSize)
	if err != nil {
		logger.WarnContext(ctx, "inventory images unavailable", "error", err)
	}
	for index := range page.Items {
		page.Items[index].ImageURL = images[targets[index]]
	}
	return page, nil
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
	if target, ok := robloxservices.LargeImageTarget(category, itemID); ok {
		images, err := service.thumbnails.Images(ctx, []robloxservices.ThumbnailTarget{target}, robloxservices.LargeThumbnailSize)
		if err != nil {
			logger.WarnContext(ctx, "item large image unavailable", "error", err)
		}
		details.LargeImageURL = images[target]
	}
	return details, nil
}

// IsInventoryItemOwned reads whether the account owns the item.
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

// GetWornAssetIDs lists the assets that the user wears now.
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
