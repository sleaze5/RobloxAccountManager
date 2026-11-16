package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
)

// ListGamePasses lists the game passes of a universe. An account ID of 0 lists
// them without reading which ones an account owns.
func (service *Service) ListGamePasses(ctx context.Context, accountID, universeID int64) ([]games.GamePass, error) {
	if accountID != 0 {
		if err := positiveID(accountID, "game-passes"); err != nil {
			return nil, err
		}
	}
	return service.core.ListGamePasses(ctx, accountID, universeID)
}
