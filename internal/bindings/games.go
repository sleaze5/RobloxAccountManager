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

func (service *Service) GetGamePlace(ctx context.Context, universeID, placeID int64) (games.PlaceSummary, error) {
	return service.core.GetGamePlace(ctx, universeID, placeID)
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

func (service *Service) ListOwnedPrivateServers(ctx context.Context, accountID int64, cursor string) (games.OwnedPrivateServerPage, error) {
	if err := positiveID(accountID, "private-servers-owned"); err != nil {
		return games.OwnedPrivateServerPage{}, err
	}
	return service.core.ListOwnedPrivateServers(ctx, accountID, cursor)
}

func (service *Service) ListPlacePrivateServers(ctx context.Context, query games.PrivateServerQuery) (games.PrivateServerPage, error) {
	if err := positiveID(query.AccountID, "private-servers-place"); err != nil {
		return games.PrivateServerPage{}, err
	}
	return service.core.ListPlacePrivateServers(ctx, query)
}

func (service *Service) GetPrivateServer(ctx context.Context, accountID, serverID int64) (games.PrivateServerSettings, error) {
	if err := positiveID(accountID, "private-server"); err != nil {
		return games.PrivateServerSettings{}, err
	}
	return service.core.GetPrivateServer(ctx, accountID, serverID)
}

func (service *Service) RenamePrivateServer(ctx context.Context, accountID, serverID int64, name string) (games.PrivateServerSettings, error) {
	if err := positiveID(accountID, "private-server-rename"); err != nil {
		return games.PrivateServerSettings{}, err
	}
	return service.core.RenamePrivateServer(ctx, accountID, serverID, name)
}

func (service *Service) SetPrivateServerOption(ctx context.Context, accountID, serverID int64, option games.PrivateServerOption, enabled bool) (games.PrivateServerSettings, error) {
	if err := positiveID(accountID, "private-server-option"); err != nil {
		return games.PrivateServerSettings{}, err
	}
	return service.core.SetPrivateServerOption(ctx, accountID, serverID, option, enabled)
}

func (service *Service) ResetPrivateServerLink(ctx context.Context, accountID, serverID int64) (games.PrivateServerSettings, error) {
	if err := positiveID(accountID, "private-server-link-reset"); err != nil {
		return games.PrivateServerSettings{}, err
	}
	return service.core.ResetPrivateServerLink(ctx, accountID, serverID)
}

func (service *Service) AddPrivateServerUser(ctx context.Context, accountID, serverID int64, user string) (games.PrivateServerSettings, error) {
	if err := positiveID(accountID, "private-server-user-add"); err != nil {
		return games.PrivateServerSettings{}, err
	}
	return service.core.AddPrivateServerUser(ctx, accountID, serverID, user)
}

func (service *Service) RemovePrivateServerUser(ctx context.Context, accountID, serverID, userID int64) (games.PrivateServerSettings, error) {
	if err := positiveID(accountID, "private-server-user-remove"); err != nil {
		return games.PrivateServerSettings{}, err
	}
	return service.core.RemovePrivateServerUser(ctx, accountID, serverID, userID)
}

func (service *Service) ListRBXServersPrivateServers(ctx context.Context, placeID int64) (games.ListedPrivateServerPage, error) {
	return service.core.ListRBXServersPrivateServers(ctx, placeID)
}

func (service *Service) ListSavedPrivateServers(ctx context.Context, placeID int64) ([]games.SavedPrivateServer, error) {
	return service.core.ListSavedPrivateServers(ctx, placeID)
}

func (service *Service) SavePrivateServerLink(ctx context.Context, placeID, accountID int64, name, link string) (games.SavedPrivateServer, error) {
	return service.core.SavePrivateServerLink(ctx, placeID, accountID, name, link)
}

func (service *Service) RemoveSavedPrivateServer(ctx context.Context, id int64) error {
	return service.core.RemoveSavedPrivateServer(ctx, id)
}
