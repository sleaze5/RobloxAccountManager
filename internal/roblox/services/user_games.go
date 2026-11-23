package services

import (
	"cmp"
	"context"
	"net/url"
	"slices"
	"strconv"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
)

// maxProfileGames bounds each list of games on a profile to one page of
// favorites, the most that Roblox returns at once.
const maxProfileGames = 50

// UserGames reads one list of games on the user's profile, as root places with
// icons. Favorites come newest first, and created games most visited first.
func (service *Games) UserGames(ctx context.Context, userID int64, list games.UserGameList) ([]games.Place, error) {
	var universeIDs []int64
	var err error
	switch list {
	case games.UserGameListFavorites:
		universeIDs, err = service.favoriteUniverseIDs(ctx, userID)
	case games.UserGameListCreated:
		universeIDs, err = service.createdUniverseIDs(ctx, userID)
	default:
		return nil, gameDataError("This list of games does not exist.")
	}
	if err != nil {
		return nil, err
	}
	return service.placesInOrder(ctx, universeIDs)
}

// favoriteUniverseIDs reads the user's newest favorite games. Roblox lists
// them without a signed-in request. Verified on 2026-11-23.
func (service *Games) favoriteUniverseIDs(ctx context.Context, userID int64) ([]int64, error) {
	var payload struct {
		Data *[]struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	parameters := url.Values{"accessFilter": {"2"}, "limit": {strconv.Itoa(maxProfileGames)}, "sortOrder": {"Desc"}}
	path := "/v2/users/" + strconv.FormatInt(userID, 10) + "/favorite/games"
	if err := service.get(ctx, "user-favorite-games", "games.roblox.com", path, parameters, &payload); err != nil {
		return nil, err
	}
	if payload.Data == nil || len(*payload.Data) > maxProfileGames {
		return nil, gameDataError("Roblox returned invalid favorite games.")
	}
	universeIDs := make([]int64, 0, len(*payload.Data))
	for _, favorite := range *payload.Data {
		if games.ValidID(favorite.ID) && !slices.Contains(universeIDs, favorite.ID) {
			universeIDs = append(universeIDs, favorite.ID)
		}
	}
	return universeIDs, nil
}

func (service *Games) createdUniverseIDs(ctx context.Context, userID int64) ([]int64, error) {
	createdGames, err := service.creatorGames(ctx, "User", userID)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(createdGames, func(left, right creatorGame) int { return cmp.Compare(right.visits, left.visits) })
	universeIDs := make([]int64, 0, min(len(createdGames), maxProfileGames))
	for _, game := range createdGames[:min(len(createdGames), maxProfileGames)] {
		universeIDs = append(universeIDs, game.universeID)
	}
	return universeIDs, nil
}
