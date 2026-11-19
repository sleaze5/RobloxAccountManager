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
	maxOwnershipCheckItemCount   = 100
	itemOwnershipWorkers         = 6
	savedAccountOwnershipWorkers = 4
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

type SavedAccountItemOwner struct {
	AccountID int64 `json:"accountId"`
	IsWorn    bool  `json:"isWorn"`
}

// SavedAccountsItemOwnership lists the other saved accounts that own an item.
type SavedAccountsItemOwnership struct {
	Owners []SavedAccountItemOwner `json:"owners"`
	// CheckedAccountCount counts the accounts that answered, owners included.
	CheckedAccountCount int `json:"checkedAccountCount"`
	FailedAccountCount  int `json:"failedAccountCount"`
}

// GetSavedAccountsItemOwnership reads which of the other saved accounts own
// the item, and whether they wear it. Each read uses that account's own
// session, because Roblox shows a private inventory only to its owner.
func (service *Service) GetSavedAccountsItemOwnership(ctx context.Context, viewerAccountID int64, category robloxservices.InventoryCategory, itemID int64) (SavedAccountsItemOwnership, error) {
	const operation = "saved-accounts-item-ownership"
	ctx, _, done, err := service.beginAccountSession(ctx, viewerAccountID, operation)
	if err != nil {
		return SavedAccountsItemOwnership{}, err
	}
	defer done()
	views, err := service.savedAccounts(ctx)
	if err != nil {
		return SavedAccountsItemOwnership{}, err
	}
	candidates := slices.DeleteFunc(views, func(view accounts.AccountView) bool {
		return view.ID == viewerAccountID || !canReadWithSession(view)
	})
	owners := make([]*SavedAccountItemOwner, len(candidates))
	failures := make([]error, len(candidates))
	forEachIndexConcurrently(len(candidates), savedAccountOwnershipWorkers, func(index int) {
		owners[index], failures[index] = service.savedAccountItemOwner(ctx, candidates[index], category, itemID)
	})
	ownership := SavedAccountsItemOwnership{Owners: []SavedAccountItemOwner{}}
	var firstFailure error
	for index, failure := range failures {
		switch {
		case failure != nil:
			ownership.FailedAccountCount++
			if firstFailure == nil {
				firstFailure = failure
			}
		case owners[index] != nil:
			ownership.Owners = append(ownership.Owners, *owners[index])
			ownership.CheckedAccountCount++
		default:
			ownership.CheckedAccountCount++
		}
	}
	if firstFailure != nil {
		service.logs.Module("application.inventory").WarnContext(ctx, "saved account item ownership incomplete", "operation", operation, "account_id", viewerAccountID, "category", category, "accounts", len(candidates), "failed", ownership.FailedAccountCount, "error", firstFailure)
		if ownership.CheckedAccountCount == 0 {
			return SavedAccountsItemOwnership{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: operation, Message: "Could not check your other accounts.", Cause: firstFailure}
		}
	}
	return ownership, nil
}

// savedAccountItemOwner returns the account as an owner of the item, or nil
// when it does not own the item. A version of 0 accepts the account's current
// session.
func (service *Service) savedAccountItemOwner(ctx context.Context, view accounts.AccountView, category robloxservices.InventoryCategory, itemID int64) (*SavedAccountItemOwner, error) {
	isOwned, err := service.users.IsInventoryItemOwned(ctx, view.ID, 0, view.RobloxUserID, category, itemID)
	if err != nil || !isOwned {
		return nil, err
	}
	owner := &SavedAccountItemOwner{AccountID: view.ID}
	if robloxservices.IsAssetCategory(category) {
		wornAssetIDs, err := service.users.WornAssetIDs(ctx, view.ID, 0, view.RobloxUserID)
		if err != nil {
			return nil, err
		}
		owner.IsWorn = slices.Contains(wornAssetIDs, itemID)
	}
	return owner, nil
}
