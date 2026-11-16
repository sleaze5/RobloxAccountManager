package services

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"sync"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	universeGamePassesPageSize   = 100
	maxUniverseGamePasses        = 500
	maxGamePassNameLength        = 100
	maxGamePassDescriptionLength = 1000
	creatorGamesPageSize         = 50
	// maxCreatorGamesSearched bounds the search for a pass's game.
	maxCreatorGamesSearched   = 100
	gamePassSearchWorkerLimit = 4
)

// GamePasses lists a universe's game passes without icons. Ownership needs a
// signed-in request. Verified on 2026-11-16.
func (service *Games) GamePasses(ctx context.Context, universeID, accountID int64) ([]games.GamePass, error) {
	gamePasses := []games.GamePass{}
	err := readCursorPages("game-passes", maxUniverseGamePasses/universeGamePassesPageSize, "This game has too many game passes.", func(pageToken string) (string, error) {
		parameters := url.Values{"passView": {"Full"}, "pageSize": {strconv.Itoa(universeGamePassesPageSize)}}
		if pageToken != "" {
			parameters.Set("pageToken", pageToken)
		}
		request := roblox.Request{
			Endpoint: "game-passes", Method: http.MethodGet, MaxResponseSize: 2 << 20,
			URL:   &url.URL{Scheme: "https", Host: "apis.roblox.com", Path: "/game-passes/v1/universes/" + strconv.FormatInt(universeID, 10) + "/game-passes", RawQuery: parameters.Encode()},
			Retry: roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
		}
		if accountID > 0 {
			request.AccountID, request.Authenticated = accountID, true
		}
		response, err := service.client.Do(ctx, request)
		if err != nil {
			return "", err
		}
		var payload struct {
			GamePasses *[]struct {
				ID                 int64  `json:"id"`
				Name               string `json:"name"`
				DisplayName        string `json:"displayName"`
				DisplayDescription string `json:"displayDescription"`
				IsForSale          bool   `json:"isForSale"`
				Price              *int64 `json:"price"`
				IsOwned            bool   `json:"isOwned"`
				Created            string `json:"created"`
				Updated            string `json:"updated"`
			} `json:"gamePasses"`
			NextPageToken string `json:"nextPageToken"`
		}
		if err := json.Unmarshal(response.Body, &payload); err != nil {
			return "", &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "game-passes", Status: response.Status, Message: "Roblox returned invalid game pass data.", Cause: err}
		}
		if payload.GamePasses == nil || len(*payload.GamePasses) > universeGamePassesPageSize {
			return "", &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "game-passes", Status: response.Status, Message: "Roblox returned invalid game pass data."}
		}
		for _, listedGamePass := range *payload.GamePasses {
			if !games.ValidID(listedGamePass.ID) {
				continue
			}
			gamePass := games.GamePass{
				GamePassID:  listedGamePass.ID,
				Name:        boundedText(listedGamePass.DisplayName, maxGamePassNameLength),
				Description: boundedText(listedGamePass.DisplayDescription, maxGamePassDescriptionLength),
				IsOwned:     accountID > 0 && listedGamePass.IsOwned,
				CreatedAtMs: robloxTimeMs(listedGamePass.Created),
				UpdatedAtMs: robloxTimeMs(listedGamePass.Updated),
			}
			if gamePass.Name == "" {
				gamePass.Name = boundedText(listedGamePass.Name, maxGamePassNameLength)
			}
			if listedGamePass.IsForSale && listedGamePass.Price != nil && *listedGamePass.Price >= 0 {
				gamePass.IsForSale, gamePass.PriceInRobux = true, *listedGamePass.Price
			}
			gamePasses = append(gamePasses, gamePass)
		}
		return payload.NextPageToken, nil
	})
	if err != nil {
		return nil, err
	}
	return gamePasses, nil
}

type creatorGame struct {
	universeID  int64
	rootPlaceID int64
	visits      int64
}

// GamePassRootPlaceID finds the root place of the game selling the pass.
// Roblox doesn't expose it, so this searches the creator's public games, most
// visited first. Verified on 2026-11-16.
func (service *Games) GamePassRootPlaceID(ctx context.Context, gamePassID int64) (int64, bool, error) {
	var productPayload struct {
		Creator struct {
			CreatorType     string `json:"CreatorType"`
			CreatorTargetID int64  `json:"CreatorTargetId"`
		} `json:"Creator"`
	}
	err := service.get(ctx, "game-pass-creator", "apis.roblox.com", "/game-passes/v1/game-passes/"+strconv.FormatInt(gamePassID, 10)+"/product-info", nil, &productPayload)
	if remote, ok := errors.AsType[*roblox.Error](err); ok && (remote.Status == http.StatusBadRequest || remote.Status == http.StatusNotFound) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	creator := productPayload.Creator
	if !validGameCreator(creator.CreatorTargetID, creator.CreatorType) {
		return 0, false, nil
	}
	creatorGames, err := service.creatorGames(ctx, creator.CreatorType, creator.CreatorTargetID)
	if err != nil {
		return 0, false, err
	}
	slices.SortFunc(creatorGames, func(left, right creatorGame) int { return cmp.Compare(right.visits, left.visits) })
	return service.findGameSellingGamePass(ctx, creatorGames, gamePassID)
}

// findGameSellingGamePass checks games with a few workers and stops at the
// first match.
func (service *Games) findGameSellingGamePass(ctx context.Context, creatorGames []creatorGame, gamePassID int64) (int64, bool, error) {
	searchContext, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan creatorGame, len(creatorGames))
	for _, game := range creatorGames {
		jobs <- game
	}
	close(jobs)
	var mutex sync.Mutex
	var rootPlaceID int64
	var firstError error
	var workers sync.WaitGroup
	for range min(gamePassSearchWorkerLimit, len(creatorGames)) {
		workers.Go(func() {
			for game := range jobs {
				gamePasses, err := service.GamePasses(searchContext, game.universeID, 0)
				mutex.Lock()
				if err != nil && firstError == nil && searchContext.Err() == nil {
					firstError = err
				}
				if err == nil && slices.ContainsFunc(gamePasses, func(gamePass games.GamePass) bool { return gamePass.GamePassID == gamePassID }) && rootPlaceID == 0 {
					rootPlaceID = game.rootPlaceID
					cancel()
				}
				mutex.Unlock()
			}
		})
	}
	workers.Wait()
	switch {
	case rootPlaceID != 0:
		return rootPlaceID, true, nil
	case ctx.Err() != nil:
		return 0, false, ctx.Err()
	case firstError != nil:
		return 0, false, firstError
	}
	return 0, false, nil
}

// creatorGames lists up to maxCreatorGamesSearched public games of a creator.
func (service *Games) creatorGames(ctx context.Context, creatorType string, creatorID int64) ([]creatorGame, error) {
	path := "/v2/users/" + strconv.FormatInt(creatorID, 10) + "/games"
	if creatorType == "Group" {
		path = "/v2/groups/" + strconv.FormatInt(creatorID, 10) + "/gamesV2"
	}
	creatorGames := []creatorGame{}
	cursor := ""
	for range maxCreatorGamesSearched / creatorGamesPageSize {
		parameters := url.Values{"accessFilter": {"Public"}, "limit": {strconv.Itoa(creatorGamesPageSize)}, "sortOrder": {"Desc"}}
		if cursor != "" {
			parameters.Set("cursor", cursor)
		}
		var payload struct {
			NextPageCursor string `json:"nextPageCursor"`
			Data           []struct {
				ID        int64 `json:"id"`
				RootPlace struct {
					ID int64 `json:"id"`
				} `json:"rootPlace"`
				PlaceVisits int64 `json:"placeVisits"`
			} `json:"data"`
		}
		if err := service.get(ctx, "game-pass-creator-games", "games.roblox.com", path, parameters, &payload); err != nil {
			return nil, err
		}
		if len(payload.Data) > creatorGamesPageSize || len(payload.NextPageCursor) > maxServerCursorLength {
			return nil, gameDataError("Roblox returned an oversized game list.")
		}
		for _, listedGame := range payload.Data {
			if games.ValidID(listedGame.ID) && games.ValidID(listedGame.RootPlace.ID) {
				creatorGames = append(creatorGames, creatorGame{universeID: listedGame.ID, rootPlaceID: listedGame.RootPlace.ID, visits: listedGame.PlaceVisits})
			}
		}
		if payload.NextPageCursor == "" || payload.NextPageCursor == cursor {
			break
		}
		cursor = payload.NextPageCursor
	}
	return creatorGames, nil
}
