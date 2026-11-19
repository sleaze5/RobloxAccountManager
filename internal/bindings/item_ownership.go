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

func (service *Service) GetSavedAccountsItemOwnership(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) (appservice.SavedAccountsItemOwnership, error) {
	if err := validateAccountAndTargetIDs(accountID, itemID, "saved-accounts-item-ownership"); err != nil {
		return appservice.SavedAccountsItemOwnership{}, err
	}
	return service.core.GetSavedAccountsItemOwnership(ctx, accountID, category, itemID)
}
