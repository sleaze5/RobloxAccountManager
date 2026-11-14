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
	images, err := service.thumbnails.Images(ctx, targets)
	if err != nil {
		logger.WarnContext(ctx, "inventory images unavailable", "error", err)
	}
	for index := range page.Items {
		page.Items[index].ImageURL = images[targets[index]]
	}
	return page, nil
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

func (service *Service) DeleteInventoryAsset(ctx context.Context, accountID, assetID int64) error {
	return service.editAccount(ctx, accountID, "inventory-asset-delete", "Roblox does not allow deleting this item.", func(ctx context.Context, version int64) error {
		return service.users.DeleteInventoryAsset(ctx, accountID, version, assetID)
	})
}
