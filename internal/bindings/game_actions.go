package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/appservice"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

func (service *Service) GetGameAccountStatus(ctx context.Context, accountID, universeID int64) (appservice.GameAccountStatus, error) {
	if err := validateAccountAndTargetIDs(accountID, universeID, "game-account-status"); err != nil {
		return appservice.GameAccountStatus{}, err
	}
	return service.core.GetGameAccountStatus(ctx, accountID, universeID)
}

func (service *Service) SetGameFavorite(ctx context.Context, accountID, universeID int64, isFavorite bool) error {
	if err := validateAccountAndTargetIDs(accountID, universeID, "game-favorite-update"); err != nil {
		return err
	}
	return service.core.SetGameFavorite(ctx, accountID, universeID, isFavorite)
}

func (service *Service) SetGameVote(ctx context.Context, accountID, universeID int64, vote robloxservices.GameVote) error {
	if err := validateAccountAndTargetIDs(accountID, universeID, "game-vote-update"); err != nil {
		return err
	}
	return service.core.SetGameVote(ctx, accountID, universeID, vote)
}

func (service *Service) GetGameSettings(ctx context.Context, accountID, universeID int64) (robloxservices.GameSettings, error) {
	if err := validateAccountAndTargetIDs(accountID, universeID, "game-settings"); err != nil {
		return robloxservices.GameSettings{}, err
	}
	return service.core.GetGameSettings(ctx, accountID, universeID)
}

func (service *Service) SaveGameSettings(ctx context.Context, accountID, universeID int64, change appservice.GameSettingsChange) (robloxservices.GameSettings, error) {
	if err := validateAccountAndTargetIDs(accountID, universeID, "game-settings-update"); err != nil {
		return robloxservices.GameSettings{}, err
	}
	return service.core.SaveGameSettings(ctx, accountID, universeID, change)
}
