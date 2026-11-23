package appservice

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
)

// ListUserGames lists the favorite or created games of a user's profile.
func (service *Service) ListUserGames(ctx context.Context, userID int64, list games.UserGameList) ([]games.Place, error) {
	operation := "user-games-" + string(list)
	if userID <= 0 {
		return nil, gamesInputError("The user is invalid.")
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return nil, err
	}
	defer done()
	places, err := service.games.UserGames(ctx, userID, list)
	if err != nil {
		return nil, service.gamesError(ctx, operation, err)
	}
	return places, nil
}
