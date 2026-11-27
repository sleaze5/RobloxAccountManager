package services

import (
	"context"
	"strconv"
)

// IsGameFavorite reads whether the signed-in account marked the game as a
// favorite. Roblox answers only for the account of the session.
func (service *Users) IsGameFavorite(ctx context.Context, accountID, version, universeID int64) (bool, error) {
	var payload struct {
		IsFavorited *bool `json:"isFavorited"`
	}
	const endpoint = "game-favorite"
	address := "https://games.roblox.com/v1/games/" + strconv.FormatInt(universeID, 10) + "/favorites"
	if err := service.readAccountDetails(ctx, accountID, version, endpoint, address, &payload); err != nil {
		return false, err
	}
	if payload.IsFavorited == nil {
		return false, invalidProfileDetails(endpoint)
	}
	return *payload.IsFavorited, nil
}
