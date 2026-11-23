package appservice

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// ListDeveloperProducts lists a universe's developer products with icons.
func (service *Service) ListDeveloperProducts(ctx context.Context, universeID int64) ([]games.DeveloperProduct, error) {
	const operation = "developer-products"
	if err := validGameID(universeID); err != nil {
		return nil, err
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return nil, err
	}
	defer done()
	developerProducts, err := service.games.DeveloperProducts(ctx, universeID)
	if err != nil {
		return nil, service.gamesError(ctx, operation, err)
	}
	targets := make([]robloxservices.ThumbnailTarget, len(developerProducts))
	for index, developerProduct := range developerProducts {
		targets[index] = robloxservices.ThumbnailTarget{Type: "DeveloperProduct", TargetID: developerProduct.DeveloperProductID}
	}
	icons, err := service.thumbnails.Images(ctx, targets, robloxservices.SmallThumbnailSize)
	if ctx.Err() != nil {
		return nil, service.gamesError(ctx, operation, ctx.Err())
	}
	if err != nil {
		service.logs.Module("application.games").WarnContext(ctx, "developer product icons unavailable", "operation", "developer-product-icons", "universe_id", universeID, "error", err)
	}
	for index := range developerProducts {
		developerProducts[index].IconURL = icons[targets[index]]
	}
	return developerProducts, nil
}
