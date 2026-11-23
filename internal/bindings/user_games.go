package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
)

func (service *Service) ListUserGames(ctx context.Context, userID int64, list games.UserGameList) ([]games.Place, error) {
	return service.core.ListUserGames(ctx, userID, list)
}
