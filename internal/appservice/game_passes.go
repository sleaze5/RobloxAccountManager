package appservice

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// ListGamePasses lists the game passes of a universe with their icons, and
// whether the account owns each of them. Without an account, every pass reads
// as not owned. A missing icon leaves the pass without one.
func (service *Service) ListGamePasses(ctx context.Context, accountID, universeID int64) ([]games.GamePass, error) {
	if err := validGameID(universeID); err != nil {
		return nil, err
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return nil, err
	}
	defer done()
	gamePasses, err := service.games.GamePasses(ctx, universeID, accountID)
	if err != nil {
		return nil, service.gamesError(ctx, "game-passes", err)
	}
	targets := make([]robloxservices.ThumbnailTarget, len(gamePasses))
	for index, gamePass := range gamePasses {
		targets[index] = robloxservices.ThumbnailTarget{Type: "GamePass", TargetID: gamePass.GamePassID}
	}
	icons, err := service.thumbnails.Images(ctx, targets, robloxservices.SmallThumbnailSize)
	if ctx.Err() != nil {
		return nil, service.gamesError(ctx, "game-passes", ctx.Err())
	}
	if err != nil {
		service.logs.Module("application.games").WarnContext(ctx, "game pass icons unavailable", "operation", "game-pass-icons", "universe_id", universeID, "error", err)
	}
	for index := range gamePasses {
		gamePasses[index].IconURL = icons[targets[index]]
	}
	return gamePasses, nil
}

// FindGamePassPlace finds the root place of the game that sells the game pass.
func (service *Service) FindGamePassPlace(ctx context.Context, gamePassID int64) (games.Place, error) {
	if err := validGameID(gamePassID); err != nil {
		return games.Place{}, err
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return games.Place{}, err
	}
	defer done()
	rootPlaceID, found, err := service.games.GamePassRootPlaceID(ctx, gamePassID)
	if err != nil {
		return games.Place{}, service.gamesError(ctx, "game-pass-place", err)
	}
	if !found {
		return games.Place{}, gamesInputError("The game that sells this game pass couldn't be found.")
	}
	place, found, err := service.games.PlaceByID(ctx, rootPlaceID)
	if err != nil {
		return games.Place{}, service.gamesError(ctx, "game-pass-place", err)
	}
	if !found {
		return games.Place{}, gamesInputError("The game that sells this game pass is unavailable.")
	}
	return place, nil
}
