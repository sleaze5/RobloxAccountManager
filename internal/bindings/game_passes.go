package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
)

// ListGamePasses skips ownership for an account ID of 0.
func (service *Service) ListGamePasses(ctx context.Context, accountID, universeID int64) ([]games.GamePass, error) {
	if accountID != 0 {
		if err := positiveID(accountID, "game-passes"); err != nil {
			return nil, err
		}
	}
	return service.core.ListGamePasses(ctx, accountID, universeID)
}

func (service *Service) FindGamePassPlace(ctx context.Context, gamePassID int64) (games.Place, error) {
	return service.core.FindGamePassPlace(ctx, gamePassID)
}
