package appservice

import (
	"context"
	"crypto/rand"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/games"
	"github.com/sleaze5/RobloxAccountManager/internal/integration/rovalra"
	"github.com/sleaze5/RobloxAccountManager/internal/logging"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	gamestore "github.com/sleaze5/RobloxAccountManager/internal/storage/games"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/vault"
)

const maxFavoriteNicknameLength = 50

func (service *Service) SearchGames(ctx context.Context, query games.SearchQuery) (games.SearchPage, error) {
	query.Text = strings.TrimSpace(query.Text)
	if query.Text == "" || utf8.RuneCountInString(query.Text) > 100 {
		return games.SearchPage{}, gamesInputError("Enter a game name of up to 100 characters.")
	}
	if len(query.SessionID) > 64 || len(query.PageToken) > 2048 || query.PageToken != "" && query.SessionID == "" {
		return games.SearchPage{}, gamesInputError("This search has expired. Start a new search.")
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return games.SearchPage{}, err
	}
	defer done()
	if universeID, ok, err := gameIDQuery(query.Text, "universe:", "Enter a universe ID after universe:, such as universe:13058."); ok {
		if err != nil {
			return games.SearchPage{}, err
		}
		places, _, err := service.games.UniversePlaces(ctx, universeID)
		if places == nil {
			places = []games.Place{}
		}
		return games.SearchPage{Places: places}, service.gamesError(ctx, "game-search-universe", err)
	}
	if placeID, ok, err := gameIDQuery(query.Text, "id:", "Enter a place ID after id:, such as id:1818."); ok {
		if err != nil {
			return games.SearchPage{}, err
		}
		page := games.SearchPage{Places: []games.Place{}}
		place, exists, err := service.games.PlaceByID(ctx, placeID)
		if exists {
			page.Places = append(page.Places, place)
		}
		return page, service.gamesError(ctx, "game-search-id", err)
	}
	page, err := service.games.Search(ctx, query)
	return page, service.gamesError(ctx, "game-search", err)
}

func gameIDQuery(text, prefix, invalid string) (int64, bool, error) {
	if len(text) < len(prefix) || !strings.EqualFold(text[:len(prefix)], prefix) {
		return 0, false, nil
	}
	id, err := strconv.ParseInt(strings.TrimSpace(text[len(prefix):]), 10, 64)
	if err != nil || validGameID(id) != nil {
		return 0, true, gamesInputError(invalid)
	}
	return id, true, nil
}

func (service *Service) GetGame(ctx context.Context, universeID, placeID int64) (games.Game, error) {
	for _, id := range []int64{universeID, placeID} {
		if err := validGameID(id); err != nil {
			return games.Game{}, err
		}
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return games.Game{}, err
	}
	defer done()
	accountID, err := service.gameAccount(ctx)
	if err != nil {
		return games.Game{}, service.gamesError(ctx, "game-details", err)
	}
	game, err := service.games.Get(ctx, universeID, placeID, accountID)
	return game, service.gamesError(ctx, "game-details", err)
}

func (service *Service) ListGameServers(ctx context.Context, query games.ServerQuery) (games.ServerPage, error) {
	if err := validGameID(query.PlaceID); err != nil {
		return games.ServerPage{}, err
	}
	switch query.Order {
	case games.ServerOrderRecommended, games.ServerOrderBestPing, games.ServerOrderMostPlayers, games.ServerOrderFewest:
	default:
		return games.ServerPage{}, gamesInputError("Choose how to sort the servers.")
	}
	if query.Limit != 10 && query.Limit != 25 && query.Limit != 50 && query.Limit != 100 {
		return games.ServerPage{}, gamesInputError("Choose 10, 25, 50, or 100 servers per page.")
	}
	if len(query.Cursor) > 2048 {
		return games.ServerPage{}, gamesInputError("This server page has expired. Refresh the server list.")
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return games.ServerPage{}, err
	}
	defer done()
	accountID, err := service.gameAccount(ctx)
	if err != nil {
		return games.ServerPage{}, service.gamesError(ctx, "game-servers", err)
	}
	page, err := service.games.Servers(ctx, query, accountID)
	if err != nil {
		return games.ServerPage{}, service.gamesError(ctx, "game-servers", err)
	}
	service.addServerRecords(ctx, query.PlaceID, page.Servers)
	return page, nil
}

func (service *Service) addServerRecords(ctx context.Context, placeID int64, servers []games.Server) {
	if !service.settings.Integrations().RoValra || len(servers) == 0 {
		return
	}
	jobIDs := make([]string, len(servers))
	for index, server := range servers {
		jobIDs[index] = server.JobID
	}
	recordsContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	records, err := service.rovalra.Records(recordsContext, placeID, jobIDs)
	if err != nil && ctx.Err() == nil {
		service.logs.Module("integration.rovalra").WarnContext(ctx, "server records unavailable", "operation", "game-server-records", "place_id", placeID, "error", err)
	}
	for index := range servers {
		if record, ok := records[servers[index].JobID]; ok {
			servers[index].Record = &record
		}
	}
}

func (service *Service) ListRecordedGameServers(ctx context.Context, query games.RecordQuery) (games.RecordPage, error) {
	if err := validGameID(query.PlaceID); err != nil {
		return games.RecordPage{}, err
	}
	if query.Order != games.RecordOrderNewest && query.Order != games.RecordOrderOldest {
		return games.RecordPage{}, gamesInputError("Choose how to sort the servers.")
	}
	if query.Region != "" && (query.Order != games.RecordOrderNewest || !rovalra.ValidRegionCode(query.Region)) {
		return games.RecordPage{}, gamesInputError("Choose a region from the list.")
	}
	if query.Limit != 10 && query.Limit != 50 && query.Limit != 100 {
		return games.RecordPage{}, gamesInputError("Choose 10, 50, or 100 servers per page.")
	}
	if query.Cursor < 0 {
		return games.RecordPage{}, gamesInputError("This server page has expired. Refresh the server list.")
	}
	ctx, done, err := service.beginRoValra(ctx)
	if err != nil {
		return games.RecordPage{}, err
	}
	defer done()
	page, err := service.rovalra.Servers(ctx, query)
	return page, service.rovalraError(ctx, "rovalra-servers", query.PlaceID, err)
}

func (service *Service) GetGameServerStats(ctx context.Context, placeID int64) (games.ServerStats, error) {
	if err := validGameID(placeID); err != nil {
		return games.ServerStats{}, err
	}
	ctx, done, err := service.beginRoValra(ctx)
	if err != nil {
		return games.ServerStats{}, err
	}
	defer done()
	stats, err := service.rovalra.Stats(ctx, placeID)
	return stats, service.rovalraError(ctx, "rovalra-stats", placeID, err)
}

func (service *Service) beginRoValra(ctx context.Context) (context.Context, func(), error) {
	if !service.settings.Integrations().RoValra {
		return nil, nil, gamesInputError("Turn on the RoValra integration in Settings to use RoValra servers.")
	}
	return service.beginGames(ctx)
}

func (service *Service) rovalraError(ctx context.Context, operation string, placeID int64, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) && ctx.Err() != nil {
		return &roblox.Error{Kind: roblox.KindCancelled, Endpoint: operation, Message: "The RoValra request was cancelled."}
	}
	service.logs.Module("integration.rovalra").WarnContext(ctx, "rovalra request failed", "operation", operation, "place_id", placeID, "error", err)
	return &roblox.Error{Kind: roblox.KindServer, Endpoint: operation, Message: "RoValra could not be reached. Try again later.", Cause: err}
}

func (service *Service) gameAccount(ctx context.Context) (int64, error) {
	var cursor *accounts.AccountCursor
	for {
		page, err := service.repo.Page(ctx, accounts.AccountQuery{Cursor: cursor})
		if err != nil {
			return 0, mapRepositoryError(err)
		}
		for _, view := range page.Accounts {
			if view.State == accounts.StateActive {
				return view.ID, nil
			}
		}
		if cursor = page.NextCursor; cursor == nil {
			return 0, nil
		}
	}
}

func (service *Service) GetGamePlace(ctx context.Context, universeID, placeID int64) (games.PlaceSummary, error) {
	for _, id := range []int64{universeID, placeID} {
		if err := validGameID(id); err != nil {
			return games.PlaceSummary{}, err
		}
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return games.PlaceSummary{}, err
	}
	defer done()
	place, err := service.games.Place(ctx, universeID, placeID)
	return place, service.gamesError(ctx, "game-place", err)
}

func (service *Service) ListFavoritePlaces(ctx context.Context) ([]games.Place, error) {
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return nil, err
	}
	defer done()
	places, err := service.favoritePlaces.List(ctx)
	return places, service.gamesError(ctx, "favorite-places-list", err)
}

func (service *Service) AddFavoritePlace(ctx context.Context, placeID int64) (games.Place, error) {
	if err := validGameID(placeID); err != nil {
		return games.Place{}, err
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return games.Place{}, err
	}
	defer done()
	place, exists, err := service.games.PlaceByID(ctx, placeID)
	if err != nil {
		return games.Place{}, service.gamesError(ctx, "favorite-place-save", err)
	}
	if !exists {
		return games.Place{}, gamesInputError("This place is private, deleted, or unavailable.")
	}
	if err := service.favoritePlaces.Save(ctx, place); err != nil {
		return games.Place{}, service.gamesError(ctx, "favorite-place-save", err)
	}
	return place, nil
}

func (service *Service) SetFavoritePlaceNickname(ctx context.Context, placeID int64, nickname string) (string, error) {
	if err := validGameID(placeID); err != nil {
		return "", err
	}
	nickname = strings.TrimSpace(nickname)
	if utf8.RuneCountInString(nickname) > maxFavoriteNicknameLength || strings.ContainsFunc(nickname, unicode.IsControl) {
		return "", gamesInputError("Enter a nickname of up to 50 characters.")
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return "", err
	}
	defer done()
	if err := service.favoritePlaces.SetNickname(ctx, placeID, nickname); errors.Is(err, gamestore.ErrNotFavorite) {
		return "", gamesInputError("This place is no longer a favorite.")
	} else if err != nil {
		return "", service.gamesError(ctx, "favorite-place-nickname", err)
	}
	return nickname, nil
}

func (service *Service) ReorderFavoritePlaces(ctx context.Context, placeIDs []int64) error {
	seen := make(map[int64]bool, len(placeIDs))
	for _, placeID := range placeIDs {
		if err := validGameID(placeID); err != nil {
			return err
		}
		if seen[placeID] {
			return gamesInputError("The favorite order is invalid.")
		}
		seen[placeID] = true
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return err
	}
	defer done()
	if err := service.favoritePlaces.Reorder(ctx, placeIDs); errors.Is(err, gamestore.ErrNotFavorite) {
		return gamesInputError("Favorites changed while they were being reordered. Try again.")
	} else if err != nil {
		return service.gamesError(ctx, "favorite-places-reorder", err)
	}
	return nil
}

func (service *Service) RemoveFavoritePlace(ctx context.Context, placeID int64) error {
	if err := validGameID(placeID); err != nil {
		return err
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return err
	}
	defer done()
	return service.gamesError(ctx, "favorite-place-remove", service.favoritePlaces.Remove(ctx, placeID))
}

func (service *Service) beginGames(parent context.Context) (context.Context, func(), error) {
	service.mu.Lock()
	unlocked := service.gamesContext
	service.mu.Unlock()
	if unlocked == nil || unlocked.Err() != nil || !service.vault.Unlocked() {
		return nil, nil, mapRepositoryError(vault.ErrLocked)
	}
	ctx, cancel := context.WithTimeout(parent, 45*time.Second)
	ctx = logging.WithOperation(ctx, "games-"+rand.Text())
	stop := context.AfterFunc(unlocked, cancel)
	return ctx, func() { stop(); cancel() }, nil
}

func (service *Service) stopGames() {
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.gamesCancel != nil {
		service.gamesCancel()
	}
	service.gamesContext, service.gamesCancel = nil, nil
}

func (service *Service) gamesError(ctx context.Context, operation string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, vault.ErrLocked) {
		return mapRepositoryError(err)
	}
	if errors.Is(err, context.Canceled) {
		return &roblox.Error{Kind: roblox.KindCancelled, Endpoint: operation, Message: "The game request was cancelled."}
	}
	logger := service.logs.Module("application.games")
	logger.WarnContext(ctx, "game operation failed", "operation", operation, "error", err)
	var remote *roblox.Error
	if errors.As(err, &remote) {
		return err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &roblox.Error{Kind: roblox.KindTimeout, Endpoint: operation, Message: "The game request timed out. Try again."}
	}
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: operation, Message: "The game operation could not be completed. Try again.", Cause: err}
}

func validGameID(id int64) error {
	// 1<<53-1 is the largest integer that a JavaScript number holds exactly.
	if id <= 0 || id > 1<<53-1 {
		return gamesInputError("The game identifier is invalid.")
	}
	return nil
}

func gamesInputError(message string) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "games", Message: message}
}
