package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
)

func (service *Service) ListDeveloperProducts(ctx context.Context, universeID int64) ([]games.DeveloperProduct, error) {
	return service.core.ListDeveloperProducts(ctx, universeID)
}
