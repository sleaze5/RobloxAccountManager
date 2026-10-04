package services

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/url"
	"strconv"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	maxServerCursorLength = 2048
	maxServerPlayerImages = 5
)

func (service *Games) Servers(ctx context.Context, query games.ServerQuery, accountID int64) (games.ServerPage, error) {
	parameters := url.Values{
		"orderBy":          {string(query.Order)},
		"excludeFullGames": {strconv.FormatBool(query.ExcludeFull)},
		"limit":            {strconv.Itoa(query.Limit)},
	}
	if query.Cursor != "" {
		parameters.Set("cursor", query.Cursor)
	}
	endpoint := &url.URL{Scheme: "https", Host: "games.roblox.com", Path: "/v2/games/" + strconv.FormatInt(query.PlaceID, 10) + "/servers/Public", RawQuery: parameters.Encode()}
	var payload struct {
		PreviousPageCursor string `json:"previousPageCursor"`
		NextPageCursor     string `json:"nextPageCursor"`
		Data               []struct {
			ID                 string   `json:"id"`
			MaxPlayers         int64    `json:"maxPlayers"`
			Playing            int64    `json:"playing"`
			PlayerTokens       []string `json:"playerTokens"`
			FPS                float64  `json:"fps"`
			Ping               *int64   `json:"ping"`
			LanguageMatchCount *int64   `json:"languageMatchCount"`
			FriendCount        *int64   `json:"friendCount"`
		} `json:"data"`
	}
	request := roblox.Request{
		Endpoint: "game-servers", Method: http.MethodGet, URL: endpoint,
		MaxResponseSize: 1 << 20,
		Retry:           roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
	}
	if accountID > 0 {
		request.AccountID, request.Authenticated = accountID, true
	}
	response, err := service.client.Do(ctx, request)
	if err != nil {
		return games.ServerPage{}, err
	}
	if err := json.Unmarshal(response.Body, &payload); err != nil {
		return games.ServerPage{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "game-servers", Message: "Roblox returned invalid server data.", Cause: err}
	}
	if len(payload.Data) > 100 || len(payload.NextPageCursor) > maxServerCursorLength || len(payload.PreviousPageCursor) > maxServerCursorLength {
		return games.ServerPage{}, gameDataError("Roblox returned an oversized server page.")
	}
	page := games.ServerPage{Servers: make([]games.Server, 0, len(payload.Data)), NextCursor: payload.NextPageCursor, PreviousCursor: payload.PreviousPageCursor}
	tokens := make([][]string, 0, len(payload.Data))
	for _, item := range payload.Data {
		if !games.ValidJobID(item.ID) {
			continue
		}
		fps := item.FPS
		if math.IsNaN(fps) || math.IsInf(fps, 0) {
			fps = 0
		}
		page.Servers = append(page.Servers, games.Server{
			JobID: item.ID, Playing: max(item.Playing, 0), MaxPlayers: max(item.MaxPlayers, 0),
			FPS: max(fps, 0), PingMS: nonNegative(item.Ping), PlayerImages: []string{},
			LanguageMatches: nonNegative(item.LanguageMatchCount),
		})
		if accountID > 0 {
			page.Servers[len(page.Servers)-1].Friends = nonNegative(item.FriendCount)
		}
		tokens = append(tokens, item.PlayerTokens[:min(len(item.PlayerTokens), maxServerPlayerImages)])
	}
	images, err := service.playerImages(ctx, tokens)
	if ctx.Err() != nil {
		return games.ServerPage{}, ctx.Err()
	}
	if err != nil {
		service.logger.WarnContext(ctx, "server player images unavailable", "operation", "game-server-players", "error", err)
	}
	for index, server := range tokens {
		for _, token := range server {
			if image := images[token]; image != "" {
				page.Servers[index].PlayerImages = append(page.Servers[index].PlayerImages, image)
			}
		}
	}
	return page, nil
}

func (service *Games) Joinable(ctx context.Context, accountID, placeID int64, jobID string) (bool, error) {
	body, err := json.Marshal(struct {
		PlaceID int64  `json:"placeId"`
		GameID  string `json:"gameId"`
	}{placeID, jobID})
	if err != nil {
		return false, err
	}
	var payload struct {
		JoinScript *struct{} `json:"joinScript"`
	}
	endpoint := &url.URL{Scheme: "https", Host: "gamejoin.roblox.com", Path: "/v1/join-game-instance"}
	if err := service.account(ctx, "game-server-join-check", accountID, http.MethodPost, endpoint, body, &payload); err != nil {
		return false, err
	}
	return payload.JoinScript != nil, nil
}

func (service *Games) playerImages(ctx context.Context, servers [][]string) (map[string]string, error) {
	type item struct {
		RequestID string `json:"requestId"`
		Token     string `json:"token"`
		Type      string `json:"type"`
		Size      string `json:"size"`
		Format    string `json:"format"`
	}
	requests := make([]item, 0)
	for _, tokens := range servers {
		for _, token := range tokens {
			if validPlayerToken(token) {
				requests = append(requests, item{RequestID: token, Token: token, Type: "AvatarHeadShot", Size: "48x48", Format: "png"})
			}
		}
	}
	result := make(map[string]string, len(requests))
	endpoint := &url.URL{Scheme: "https", Host: "thumbnails.roblox.com", Path: "/v1/batch"}
	for start := 0; start < len(requests); start += headshotBatchSize {
		body, err := json.Marshal(requests[start:min(start+headshotBatchSize, len(requests))])
		if err != nil {
			return result, err
		}
		var payload struct {
			Data []struct {
				RequestID string `json:"requestId"`
				State     string `json:"state"`
				ImageURL  string `json:"imageUrl"`
			} `json:"data"`
		}
		if err := service.do(ctx, "game-server-players", http.MethodPost, endpoint, body, &payload); err != nil {
			return result, err
		}
		for _, image := range payload.Data {
			if image.State == "Completed" && safeThumbnailURL(image.ImageURL) {
				result[image.RequestID] = image.ImageURL
			}
		}
	}
	return result, nil
}

func nonNegative(value *int64) *int64 {
	if value == nil {
		return nil
	}
	result := max(*value, 0)
	return &result
}

func validPlayerToken(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	for _, char := range value {
		if !isHex(char) {
			return false
		}
	}
	return true
}

func isHex(char rune) bool {
	return '0' <= char && char <= '9' || 'a' <= char && char <= 'f' || 'A' <= char && char <= 'F'
}
