package bindings

import (
	"context"

	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

func (service *Service) GetItemInstanceTree(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) (robloxservices.ItemInstanceTree, error) {
	if err := validateAccountAndTargetIDs(accountID, itemID, "item-instance-tree"); err != nil {
		return robloxservices.ItemInstanceTree{}, err
	}
	return service.core.GetItemInstanceTree(ctx, accountID, category, itemID)
}
