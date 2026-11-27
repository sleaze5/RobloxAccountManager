package appservice

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// ListGamePasses lists a universe's game passes with icons and ownership.
// Without an account, none read as owned.
func (service *Service) ListGamePasses(ctx context.Context, accountID, universeID int64) ([]games.GamePass, error) {
	if err := validRobloxID(universeID); err != nil {
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
	icons, err := service.thumbnails.Thumbnails(ctx, targets, robloxservices.SmallThumbnailSize)
	if ctx.Err() != nil {
		return nil, service.gamesError(ctx, "game-passes", ctx.Err())
	}
	if err != nil {
		service.logs.Module("application.games").WarnContext(ctx, "game pass icons unavailable", "operation", "game-pass-icons", "universe_id", universeID, "error", err)
	}
	for index := range gamePasses {
		gamePasses[index].IconURL, gamePasses[index].IsModerated = icons[targets[index]].ImageURL, icons[targets[index]].IsBlocked
	}
	return gamePasses, nil
}

// FindGamePassPlace returns the root place of the game selling the pass.
func (service *Service) FindGamePassPlace(ctx context.Context, gamePassID int64) (games.Place, error) {
	if err := validRobloxID(gamePassID); err != nil {
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
		return games.Place{}, gamesInputError("Could not find the game that sells this pass.")
	}
	place, found, err := service.games.PlaceByID(ctx, rootPlaceID)
	if err != nil {
		return games.Place{}, service.gamesError(ctx, "game-pass-place", err)
	}
	if !found {
		return games.Place{}, gamesInputError("The game that sells this pass is unavailable.")
	}
	return place, nil
}
