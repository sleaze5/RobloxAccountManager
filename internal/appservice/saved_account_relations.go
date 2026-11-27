package appservice

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const savedAccountRelationWorkers = 4

// otherSavedAccountRelations reads how each saved account other than the
// viewer relates to a game or an item, and keeps the relations that isRelated
// accepts. Each read uses that account's own session, because Roblox shows
// favorites and private inventories only to the account itself. A failed read
// keeps what it found and counts as failed. The call fails only when every
// read failed.
func otherSavedAccountRelations[Relation any](ctx context.Context, service *Service, viewerAccountID int64, operation string, readRelation func(context.Context, accounts.AccountView) (Relation, error), isRelated func(Relation) bool) ([]Relation, int, error) {
	ctx, _, done, err := service.beginAccountSession(ctx, viewerAccountID, operation)
	if err != nil {
		return nil, 0, err
	}
	defer done()
	views, err := service.savedAccounts(ctx)
	if err != nil {
		return nil, 0, err
	}
	candidates := make([]accounts.AccountView, 0, len(views))
	for _, view := range views {
		if view.ID != viewerAccountID && canReadWithSession(view) {
			candidates = append(candidates, view)
		}
	}
	readRelations := make([]Relation, len(candidates))
	failures := make([]error, len(candidates))
	forEachIndexConcurrently(len(candidates), savedAccountRelationWorkers, func(index int) {
		readRelations[index], failures[index] = readRelation(ctx, candidates[index])
	})
	relations := []Relation{}
	failedAccountCount := 0
	var firstFailure error
	for index, relation := range readRelations {
		if failures[index] != nil {
			failedAccountCount++
			if firstFailure == nil {
				firstFailure = failures[index]
			}
		}
		if isRelated(relation) {
			relations = append(relations, relation)
		}
	}
	if firstFailure != nil {
		service.logs.Module("application.saved-accounts").WarnContext(ctx, "saved account relations incomplete", "operation", operation, "account_id", viewerAccountID, "accounts", len(candidates), "failed", failedAccountCount, "error", firstFailure)
		if failedAccountCount == len(candidates) {
			return nil, 0, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: operation, Message: "Could not check your other accounts.", Cause: firstFailure}
		}
	}
	return relations, failedAccountCount, nil
}

// SavedAccountsGameFavorites lists the other saved accounts that marked a game
// as a favorite.
type SavedAccountsGameFavorites struct {
	AccountIDs         []int64 `json:"accountIds"`
	FailedAccountCount int     `json:"failedAccountCount"`
}

// GetSavedAccountsGameFavorites reads which of the other saved accounts marked
// the game as a favorite.
func (service *Service) GetSavedAccountsGameFavorites(ctx context.Context, viewerAccountID, universeID int64) (SavedAccountsGameFavorites, error) {
	accountIDs, failedAccountCount, err := otherSavedAccountRelations(ctx, service, viewerAccountID, "saved-accounts-game-favorites",
		func(ctx context.Context, view accounts.AccountView) (int64, error) {
			isFavorite, err := service.users.IsGameFavorite(ctx, view.ID, 0, universeID)
			if err != nil || !isFavorite {
				return 0, err
			}
			return view.ID, nil
		}, func(accountID int64) bool { return accountID != 0 })
	if err != nil {
		return SavedAccountsGameFavorites{}, err
	}
	return SavedAccountsGameFavorites{AccountIDs: accountIDs, FailedAccountCount: failedAccountCount}, nil
}
