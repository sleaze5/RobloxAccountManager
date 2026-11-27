package appservice

import (
	"context"
	"slices"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

const (
	// maxOwnershipCheckItemCount is one inventory page, the most items that a
	// page asks about at once.
	maxOwnershipCheckItemCount = 100
	itemOwnershipWorkers       = 6
)

// InventoryItemReference names one item by the inventory category that lists
// it, because a bundle, a badge, or a game pass can share the ID of an asset.
type InventoryItemReference struct {
	Category robloxservices.InventoryCategory `json:"category"`
	ItemID   int64                            `json:"itemId"`
}

type ItemOwnership struct {
	Category robloxservices.InventoryCategory `json:"category"`
	ItemID   int64                            `json:"itemId"`
	IsOwned  bool                             `json:"isOwned"`
}

// CheckInventoryItemsOwned reads which of the items the account owns. Roblox
// answers one item for each request, so the checks run a few at a time. An
// item whose check fails is left out, unless every check fails.
func (service *Service) CheckInventoryItemsOwned(ctx context.Context, accountID int64, references []InventoryItemReference) ([]ItemOwnership, error) {
	const operation = "inventory-items-ownership"
	if len(references) > maxOwnershipCheckItemCount {
		return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: operation, Message: "Check fewer items at once."}
	}
	account, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return nil, mapRepositoryError(err)
	}
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, operation)
	if err != nil {
		return nil, err
	}
	defer done()
	ownerships := make([]ItemOwnership, len(references))
	failures := make([]error, len(references))
	forEachIndexConcurrently(len(references), itemOwnershipWorkers, func(index int) {
		reference := references[index]
		ownerships[index] = ItemOwnership{Category: reference.Category, ItemID: reference.ItemID}
		ownerships[index].IsOwned, failures[index] = service.users.IsInventoryItemOwned(ctx, accountID, version, account.RobloxUserID, reference.Category, reference.ItemID)
	})
	checkedOwnerships := make([]ItemOwnership, 0, len(references))
	var firstFailure error
	for index, failure := range failures {
		if failure == nil {
			checkedOwnerships = append(checkedOwnerships, ownerships[index])
		} else if firstFailure == nil {
			firstFailure = failure
		}
	}
	if firstFailure != nil {
		service.logs.Module("application.inventory").WarnContext(ctx, "item ownership incomplete", "operation", operation, "account_id", accountID, "items", len(references), "checked", len(checkedOwnerships), "error", firstFailure)
		if len(checkedOwnerships) == 0 {
			return nil, firstFailure
		}
	}
	return checkedOwnerships, nil
}

// SavedAccountItemRelation is how another saved account relates to an item.
// Roblox has favorites only for catalog items.
type SavedAccountItemRelation struct {
	AccountID  int64 `json:"accountId"`
	IsOwned    bool  `json:"isOwned"`
	IsWorn     bool  `json:"isWorn"`
	IsFavorite bool  `json:"isFavorite"`
}

func (relation SavedAccountItemRelation) isRelated() bool {
	return relation.IsOwned || relation.IsFavorite
}

// SavedAccountsItemRelations lists the other saved accounts that own the item
// or marked it as a favorite.
type SavedAccountsItemRelations struct {
	Relations []SavedAccountItemRelation `json:"relations"`
	// FailedAccountCount counts the accounts with a check that failed. Their
	// relations show what the other checks found.
	FailedAccountCount int `json:"failedAccountCount"`
}

// GetSavedAccountsItemRelations reads which of the other saved accounts own
// the item, wear it, or marked it as a favorite. An empty catalog item type
// skips the favorite check, because Roblox has favorites only for catalog
// items.
func (service *Service) GetSavedAccountsItemRelations(ctx context.Context, viewerAccountID int64, category robloxservices.InventoryCategory, catalogItemType robloxservices.CatalogItemType, itemID int64) (SavedAccountsItemRelations, error) {
	relations, failedAccountCount, err := otherSavedAccountRelations(ctx, service, viewerAccountID, "saved-accounts-item-relations",
		func(ctx context.Context, view accounts.AccountView) (SavedAccountItemRelation, error) {
			return service.savedAccountItemRelation(ctx, view, category, catalogItemType, itemID)
		}, SavedAccountItemRelation.isRelated)
	if err != nil {
		return SavedAccountsItemRelations{}, err
	}
	return SavedAccountsItemRelations{Relations: relations, FailedAccountCount: failedAccountCount}, nil
}

// savedAccountItemRelation reads how the account relates to the item. It
// returns what the checks found before a check failed. A version of 0
// accepts the account's current session.
func (service *Service) savedAccountItemRelation(ctx context.Context, view accounts.AccountView, category robloxservices.InventoryCategory, catalogItemType robloxservices.CatalogItemType, itemID int64) (SavedAccountItemRelation, error) {
	relation := SavedAccountItemRelation{AccountID: view.ID}
	var err error
	if robloxservices.ValidCatalogItemType(catalogItemType) {
		if relation.IsFavorite, err = service.users.IsItemFavorite(ctx, view.ID, 0, view.RobloxUserID, catalogItemType, itemID); err != nil {
			return relation, err
		}
	}
	if relation.IsOwned, err = service.users.IsInventoryItemOwned(ctx, view.ID, 0, view.RobloxUserID, category, itemID); err != nil || !relation.IsOwned {
		return relation, err
	}
	if robloxservices.IsAssetCategory(category) {
		wornAssetIDs, err := service.users.WornAssetIDs(ctx, view.ID, 0, view.RobloxUserID)
		if err != nil {
			return relation, err
		}
		relation.IsWorn = slices.Contains(wornAssetIDs, itemID)
	}
	return relation, nil
}
