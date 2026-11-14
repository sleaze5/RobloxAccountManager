package bindings

import (
	"context"

	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

func (service *Service) GetInventoryPage(ctx context.Context, accountID, userID int64, category robloxservices.InventoryCategory, cursor string) (robloxservices.InventoryPage, error) {
	if err := validateAccountAndTargetIDs(accountID, userID, "inventory"); err != nil {
		return robloxservices.InventoryPage{}, err
	}
	return service.core.GetInventoryPage(ctx, accountID, userID, category, cursor)
}

func (service *Service) GetInventoryItemDetails(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) (robloxservices.InventoryItemDetails, error) {
	if err := validateAccountAndTargetIDs(accountID, itemID, "inventory-item-details"); err != nil {
		return robloxservices.InventoryItemDetails{}, err
	}
	return service.core.GetInventoryItemDetails(ctx, accountID, category, itemID)
}

func (service *Service) IsInventoryItemOwned(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) (bool, error) {
	if err := validateAccountAndTargetIDs(accountID, itemID, "inventory-item-ownership"); err != nil {
		return false, err
	}
	return service.core.IsInventoryItemOwned(ctx, accountID, category, itemID)
}

func (service *Service) IsInventoryItemFavorite(ctx context.Context, accountID int64, itemType robloxservices.CatalogItemType, itemID int64) (bool, error) {
	if err := validateAccountAndTargetIDs(accountID, itemID, "item-favorite"); err != nil {
		return false, err
	}
	return service.core.IsInventoryItemFavorite(ctx, accountID, itemType, itemID)
}

func (service *Service) SetInventoryItemFavorite(ctx context.Context, accountID int64, itemType robloxservices.CatalogItemType, itemID int64, isFavorite bool) error {
	if err := validateAccountAndTargetIDs(accountID, itemID, "item-favorite-update"); err != nil {
		return err
	}
	return service.core.SetInventoryItemFavorite(ctx, accountID, itemType, itemID, isFavorite)
}

func (service *Service) BuyInventoryItem(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID, expectedPriceInRobux int64) (bool, error) {
	if err := validateAccountAndTargetIDs(accountID, itemID, "inventory-item-purchase"); err != nil {
		return false, err
	}
	return service.core.BuyInventoryItem(ctx, accountID, category, itemID, expectedPriceInRobux)
}

func (service *Service) DeleteInventoryAsset(ctx context.Context, accountID, assetID int64) error {
	if err := validateAccountAndTargetIDs(accountID, assetID, "inventory-asset-delete"); err != nil {
		return err
	}
	return service.core.DeleteInventoryAsset(ctx, accountID, assetID)
}

func (service *Service) GetWornAssetIDs(ctx context.Context, accountID, userID int64) ([]int64, error) {
	if err := validateAccountAndTargetIDs(accountID, userID, "inventory-worn-assets"); err != nil {
		return nil, err
	}
	return service.core.GetWornAssetIDs(ctx, accountID, userID)
}
