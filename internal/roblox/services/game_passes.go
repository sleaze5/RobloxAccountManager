package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	universeGamePassesPageSize = 100
	// MaxUniverseGamePasses bounds the game passes read for one universe, so
	// that a universe with thousands of passes cannot make one request run
	// for minutes.
	MaxUniverseGamePasses        = 500
	maxGamePassNameLength        = 100
	maxGamePassDescriptionLength = 1000
)

// GamePasses lists the game passes of the universe without icons. Roblox
// reports whether the account owns each pass when the request is signed in
// as the account, so a request without an account reads every pass as not
// owned. Shape verified on 2026-11-16.
func (service *Games) GamePasses(ctx context.Context, universeID, accountID int64) ([]games.GamePass, error) {
	gamePasses := []games.GamePass{}
	err := readCursorPages("game-passes", MaxUniverseGamePasses/universeGamePassesPageSize, "This game has too many game passes to list.", func(pageToken string) (string, error) {
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
