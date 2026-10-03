package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
)

func (service *Service) SearchGames(ctx context.Context, query games.SearchQuery) (games.SearchPage, error) {
	return service.core.SearchGames(ctx, query)
}

func (service *Service) GetGame(ctx context.Context, universeID, placeID int64) (games.Game, error) {
	return service.core.GetGame(ctx, universeID, placeID)
}

func (service *Service) ListGameServers(ctx context.Context, query games.ServerQuery) (games.ServerPage, error) {
	return service.core.ListGameServers(ctx, query)
}

func (service *Service) ListRecordedGameServers(ctx context.Context, query games.RecordQuery) (games.RecordPage, error) {
	return service.core.ListRecordedGameServers(ctx, query)
}

func (service *Service) GetGameServerStats(ctx context.Context, placeID int64) (games.ServerStats, error) {
	return service.core.GetGameServerStats(ctx, placeID)
}

func (service *Service) ListServerRegions(ctx context.Context, refresh bool) ([]games.Region, error) {
	return service.core.ListServerRegions(ctx, refresh)
}

func (service *Service) SetRoValraRegion(ctx context.Context, region string) error {
	return service.core.SetRoValraRegion(ctx, region)
}

func (service *Service) FindNearestGameServer(ctx context.Context, placeID, accountID int64) (games.NearestServer, error) {
	return service.core.FindNearestGameServer(ctx, placeID, accountID)
}

func (service *Service) GetGamePlace(ctx context.Context, universeID int64) (games.Place, error) {
	return service.core.GetGamePlace(ctx, universeID)
}

func (service *Service) ListFavoritePlaces(ctx context.Context) ([]games.Place, error) {
	return service.core.ListFavoritePlaces(ctx)
}

func (service *Service) AddFavoritePlace(ctx context.Context, placeID int64) (games.Place, error) {
	return service.core.AddFavoritePlace(ctx, placeID)
}

func (service *Service) SetFavoritePlaceNickname(ctx context.Context, placeID int64, nickname string) (string, error) {
	return service.core.SetFavoritePlaceNickname(ctx, placeID, nickname)
}

func (service *Service) ReorderFavoritePlaces(ctx context.Context, placeIDs []int64) error {
	return service.core.ReorderFavoritePlaces(ctx, placeIDs)
}

func (service *Service) RemoveFavoritePlace(ctx context.Context, placeID int64) error {
	return service.core.RemoveFavoritePlace(ctx, placeID)
}
