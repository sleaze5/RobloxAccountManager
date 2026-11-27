package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/appservice"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

func (service *Service) CheckInventoryItemsOwned(ctx context.Context, accountID int64, references []appservice.InventoryItemReference) ([]appservice.ItemOwnership, error) {
	const operation = "inventory-items-ownership"
	if err := positiveID(accountID, operation); err != nil {
		return nil, err
	}
	for _, reference := range references {
		if err := positiveID(reference.ItemID, operation); err != nil {
			return nil, err
		}
	}
	return service.core.CheckInventoryItemsOwned(ctx, accountID, references)
}

func (service *Service) GetSavedAccountsItemRelations(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, catalogItemType robloxservices.CatalogItemType, itemID int64) (appservice.SavedAccountsItemRelations, error) {
	if err := validateAccountAndTargetIDs(accountID, itemID, "saved-accounts-item-relations"); err != nil {
		return appservice.SavedAccountsItemRelations{}, err
	}
	return service.core.GetSavedAccountsItemRelations(ctx, accountID, category, catalogItemType, itemID)
}
