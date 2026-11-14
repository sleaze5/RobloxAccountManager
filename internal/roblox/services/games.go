package services

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/caches"
)

const (
	gameBatchSize         = 50
	maxUniversePlacePages = 10
)

type Games struct {
	cache  *caches.Store
	client *roblox.Client
	logger *slog.Logger
}

func NewGames(client *roblox.Client, logger *slog.Logger, cache *caches.Store) *Games {
	return &Games{cache: cache, client: client, logger: logger}
}

func (service *Games) Search(ctx context.Context, query games.SearchQuery) (games.SearchPage, error) {
	if query.SessionID == "" {
		query.SessionID = randomUUID()
	}
	parameters := url.Values{"searchQuery": {query.Text}, "sessionId": {query.SessionID}}
	if query.PageToken != "" {
		parameters.Set("pageToken", query.PageToken)
	}
	var payload struct {
		SearchResults []struct {
			ContentGroupType string `json:"contentGroupType"`
			Contents         []struct {
				ContentID int64 `json:"contentId"`
			} `json:"contents"`
		} `json:"searchResults"`
		NextPageToken string `json:"nextPageToken"`
	}
	if err := service.get(ctx, "game-search", "apis.roblox.com", "/search-api/omni-search", parameters, &payload); err != nil {
		return games.SearchPage{}, err
	}
	universeIDs := make([]int64, 0)
	seen := make(map[int64]bool)
	for _, section := range payload.SearchResults {
		if section.ContentGroupType != "Game" {
			continue
		}
		for _, content := range section.Contents {
			if games.ValidID(content.ContentID) && !seen[content.ContentID] {
				seen[content.ContentID] = true
				universeIDs = append(universeIDs, content.ContentID)
			}
		}
	}
	if len(universeIDs) > 200 || len(payload.NextPageToken) > 2048 {
		return games.SearchPage{}, gameDataError("Roblox returned an oversized search page.")
	}
	page := games.SearchPage{Places: []games.Place{}, SessionID: query.SessionID, NextPageToken: payload.NextPageToken}
	if err := service.loadSearchPlaces(ctx, universeIDs, &page); err != nil {
		return games.SearchPage{}, err
	}
	if page.NextPageToken == query.PageToken {
		page.NextPageToken = ""
	}
	return page, nil
}

func (service *Games) Place(ctx context.Context, universeID, placeID int64) (games.PlaceSummary, error) {
	details, err := service.details(ctx, []int64{universeID})
	if err != nil {
		return games.PlaceSummary{}, err
	}
	game, exists := details[universeID]
	if !exists {
		return games.PlaceSummary{}, gameUnavailableError()
	}
	root := game.Place
	if root.PlaceID == placeID {
		return games.PlaceSummary{Place: root}, nil
	}
	place, _, found, err := service.subplace(ctx, root, placeID)
	if err != nil {
		return games.PlaceSummary{}, err
	}
	if !found {
		return games.PlaceSummary{}, gameUnavailableError()
	}
	if place.IconURL == "" {
		// Roblox serves no icon for a subplace that has none of its own, so it shows its universe's icon.
		place.IconURL = root.IconURL
	}
	return games.PlaceSummary{Place: place, RootPlace: &root}, nil
}

func (service *Games) PlaceByID(ctx context.Context, placeID int64) (games.Place, bool, error) {
	var payload struct {
		UniverseID int64 `json:"universeId"`
	}
	err := service.get(ctx, "game-place-universe", "apis.roblox.com", "/universes/v1/places/"+strconv.FormatInt(placeID, 10)+"/universe", nil, &payload)
	var remote *roblox.Error
	if errors.As(err, &remote) && (remote.Status == http.StatusBadRequest || remote.Status == http.StatusNotFound) {
		return games.Place{}, false, nil
	}
	if err != nil {
		return games.Place{}, false, err
	}
	if !games.ValidID(payload.UniverseID) {
		return games.Place{}, false, nil
	}
	details, err := service.details(ctx, []int64{payload.UniverseID})
	if err != nil {
		return games.Place{}, false, err
	}
	if root, exists := details[payload.UniverseID]; !exists || root.Place.PlaceID == placeID {
		return root.Place, exists, nil
	}
	place, _, found, err := service.subplace(ctx, details[payload.UniverseID].Place, placeID)
	return place, found, err
}

func (service *Games) subplace(ctx context.Context, root games.Place, placeID int64) (games.Place, string, bool, error) {
	var place games.Place
	description, found := "", false
	err := service.subplaces(ctx, root, func(item games.Place, text string) bool {
		if item.PlaceID != placeID {
			return true
		}
		place, description, found = item, text, true
		return false
	})
	if err != nil || !found {
		return games.Place{}, "", false, err
	}
	icons, err := service.placeIcons(ctx, []int64{placeID})
	if ctx.Err() != nil {
		return games.Place{}, "", false, ctx.Err()
	}
	if err != nil {
		service.logger.WarnContext(ctx, "place icons unavailable", "operation", "place-icons", "error", err)
	}
	place.IconURL = icons[placeID]
	service.cache.SavePlace(ctx, place)
	return place, description, true, nil
}

func (service *Games) UniversePlaces(ctx context.Context, universeID int64) ([]games.Place, bool, error) {
	details, err := service.details(ctx, []int64{universeID})
	if err != nil {
		return nil, false, err
	}
	root, exists := details[universeID]
	if !exists {
		return nil, false, nil
	}
	places := []games.Place{root.Place}
	err = service.subplaces(ctx, root.Place, func(place games.Place, _ string) bool {
		places = append(places, place)
		return true
	})
	if err != nil {
		return nil, false, err
	}
	for start := 1; start < len(places); start += gameBatchSize {
		batch := places[start:min(start+gameBatchSize, len(places))]
		placeIDs := make([]int64, len(batch))
		for index, place := range batch {
			placeIDs[index] = place.PlaceID
		}
		icons, err := service.placeIcons(ctx, placeIDs)
		if ctx.Err() != nil {
			return nil, false, ctx.Err()
		}
		if err != nil {
			service.logger.WarnContext(ctx, "place icons unavailable", "operation", "place-icons", "error", err)
			break
		}
		for index := range batch {
			batch[index].IconURL = icons[batch[index].PlaceID]
		}
	}
	for _, place := range places {
		service.cache.SavePlace(ctx, place)
	}
	return places, true, nil
}

func (service *Games) subplaces(ctx context.Context, root games.Place, visit func(games.Place, string) bool) error {
	path := "/v1/universes/" + strconv.FormatInt(root.UniverseID, 10) + "/places"
	cursor := ""
	for range maxUniversePlacePages {
		parameters := url.Values{"sortOrder": {"Asc"}, "limit": {"100"}}
		if cursor != "" {
			parameters.Set("cursor", cursor)
		}
		var payload struct {
			NextPageCursor string `json:"nextPageCursor"`
			Data           []struct {
				ID          int64  `json:"id"`
				UniverseID  int64  `json:"universeId"`
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"data"`
		}
		if err := service.get(ctx, "game-universe-places", "develop.roblox.com", path, parameters, &payload); err != nil {
			return err
		}
		for _, listedPlace := range payload.Data {
			if listedPlace.UniverseID != root.UniverseID || !games.ValidID(listedPlace.ID) || listedPlace.ID == root.PlaceID || strings.TrimSpace(listedPlace.Name) == "" {
				continue
			}
			place := root
			place.PlaceID, place.Name, place.IconURL = listedPlace.ID, listedPlace.Name, ""
			if !visit(place, listedPlace.Description) {
				return nil
			}
		}
		if payload.NextPageCursor == "" || payload.NextPageCursor == cursor || len(payload.NextPageCursor) > 2048 {
			return nil
		}
		cursor = payload.NextPageCursor
	}
	return nil
}

func (service *Games) Get(ctx context.Context, universeID, placeID int64) (games.Game, error) {
	details, err := service.details(ctx, []int64{universeID})
	if err != nil {
		return games.Game{}, err
	}
	game, exists := details[universeID]
	if !exists {
		return games.Game{}, gameUnavailableError()
	}
	if placeID != game.Place.PlaceID {
		root := game.Place
		place, description, found, err := service.subplace(ctx, root, placeID)
		if err != nil {
			return games.Game{}, err
		}
		if !found {
			return games.Game{}, gameUnavailableError()
		}
		game.Place, game.Description, game.RootPlace = place, description, &root
	}
	return game, nil
}

func (service *Games) votes(ctx context.Context, game *games.Game) error {
	var payload struct {
		Data []struct {
			ID        int64 `json:"id"`
			UpVotes   int64 `json:"upVotes"`
			DownVotes int64 `json:"downVotes"`
		} `json:"data"`
	}
	parameters := url.Values{"universeIds": {strconv.FormatInt(game.Place.UniverseID, 10)}}
	if err := service.get(ctx, "game-votes", "games.roblox.com", "/v1/games/votes", parameters, &payload); err != nil {
		return err
	}
	for _, universe := range payload.Data {
		if universe.ID == game.Place.UniverseID {
			game.UpVotes, game.DownVotes = max(universe.UpVotes, 0), max(universe.DownVotes, 0)
		}
	}
	return nil
}

func (service *Games) avatarRules(ctx context.Context, game *games.Game) error {
	var payload struct {
		CustomAnimations string `json:"allowCustomAnimations"`
		AssetOverrides   []struct {
			IsPlayerChoice bool `json:"isPlayerChoice"`
		} `json:"universeAvatarAssetOverrides"`
	}
	parameters := url.Values{"universeId": {strconv.FormatInt(game.Place.UniverseID, 10)}}
	if err := service.get(ctx, "game-avatar-rules", "avatar.roblox.com", "/v1/game-start-info", parameters, &payload); err != nil {
		return err
	}
	rules := &games.AvatarRules{CustomAnimationsAllowed: strings.EqualFold(payload.CustomAnimations, "True")}
	for _, override := range payload.AssetOverrides {
		if !override.IsPlayerChoice {
			rules.ItemOverrides++
		}
	}
	game.AvatarRules = rules
	return nil
}

func (service *Games) communication(ctx context.Context, accountID int64, game *games.Game) error {
	var payload struct {
		Voice  bool `json:"isUniverseEnabledForVoice"`
		Camera bool `json:"isUniverseEnabledForAvatarVideo"`
	}
	endpoint := &url.URL{Scheme: "https", Host: "voice.roblox.com", Path: "/v1/settings/universe/" + strconv.FormatInt(game.Place.UniverseID, 10)}
	if err := service.account(ctx, "game-communication", accountID, http.MethodGet, endpoint, nil, &payload); err != nil {
		return err
	}
	game.Communication = &games.Communication{VoiceChat: payload.Voice, Camera: payload.Camera}
	return nil
}

func (service *Games) versions(ctx context.Context, accountID int64, game *games.Game) error {
	endpoint := &url.URL{Scheme: "https", Host: "develop.roblox.com", Path: "/v1/assets/latest-versions"}
	latest := func(status string) (int64, error) {
		body, err := json.Marshal(map[string]any{"assetIds": []int64{game.Place.PlaceID}, "versionStatus": status})
		if err != nil {
			return 0, fmt.Errorf("encode game version request: %w", err)
		}
		var payload struct {
			Results []struct {
				AssetID       int64 `json:"assetId"`
				VersionNumber int64 `json:"versionNumber"`
			} `json:"results"`
		}
		if err := service.account(ctx, "game-versions", accountID, http.MethodPost, endpoint, body, &payload); err != nil {
			return 0, err
		}
		for _, asset := range payload.Results {
			if asset.AssetID == game.Place.PlaceID && asset.VersionNumber > 0 {
				return asset.VersionNumber, nil
			}
		}
		return 0, gameDataError("Roblox did not return the place version.")
	}
	saved, err := latest("Any")
	if err != nil {
		return err
	}
	published, err := latest("Published")
	if err != nil {
		return err
	}
	game.Versions = &games.PlaceVersions{Saved: saved, Published: published}
	return nil
}

func (service *Games) maturity(ctx context.Context, game *games.Game) error {
	body, err := json.Marshal(map[string][]int64{"universeIds": {game.Place.UniverseID}})
	if err != nil {
		return fmt.Errorf("encode game maturity request: %w", err)
	}
	var payload struct {
		Universes []struct {
			UniverseID int64 `json:"universeId"`
			Details    struct {
				Summary struct {
					Recommendation *struct {
						DisplayName string `json:"displayName"`
					} `json:"ageRecommendation"`
				} `json:"ageRecommendationSummary"`
				Descriptors *struct {
					Items []struct {
						Contains    bool   `json:"contains"`
						DisplayName string `json:"descriptorDisplayName"`
					} `json:"items"`
				} `json:"experienceDescriptorUsages"`
			} `json:"ageRecommendationDetails"`
		} `json:"ageRecommendationDetailsByUniverse"`
	}
	endpoint := &url.URL{Scheme: "https", Host: "apis.roblox.com", Path: "/experience-guidelines-service/v1beta1/multi-age-recommendation"}
	if err := service.do(ctx, "game-maturity", http.MethodPost, endpoint, body, &payload); err != nil {
		return err
	}
	for _, universe := range payload.Universes {
		if universe.UniverseID != game.Place.UniverseID {
			continue
		}
		if recommendation := universe.Details.Summary.Recommendation; recommendation != nil {
			game.Maturity = strings.TrimSpace(recommendation.DisplayName)
		}
		if universe.Details.Descriptors != nil {
			for _, descriptor := range universe.Details.Descriptors.Items {
				if name := strings.TrimSpace(descriptor.DisplayName); descriptor.Contains && name != "" {
					game.MaturityDescriptors = append(game.MaturityDescriptors, name)
				}
			}
		}
	}
	return nil
}

func (service *Games) details(ctx context.Context, universeIDs []int64) (map[int64]games.Game, error) {
	parameters := url.Values{"universeIds": {gameIDs(universeIDs)}}
	var payload struct {
		Data []struct {
			ID          int64  `json:"id"`
			RootPlaceID int64  `json:"rootPlaceId"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Creator     struct {
				ID       int64  `json:"id"`
				Name     string `json:"name"`
				Type     string `json:"type"`
				Verified bool   `json:"hasVerifiedBadge"`
			} `json:"creator"`
			Price                 *int64    `json:"price"`
			Playing               int64     `json:"playing"`
			Visits                int64     `json:"visits"`
			FavoritedCount        int64     `json:"favoritedCount"`
			MaxPlayers            int64     `json:"maxPlayers"`
			Genre                 string    `json:"genre_l1"`
			Subgenre              string    `json:"genre_l2"`
			AvatarType            string    `json:"universeAvatarType"`
			CopyingAllowed        bool      `json:"copyingAllowed"`
			PrivateServersAllowed bool      `json:"createVipServersAllowed"`
			Created               time.Time `json:"created"`
			Updated               time.Time `json:"updated"`
		} `json:"data"`
	}
	if err := service.get(ctx, "game-details", "games.roblox.com", "/v1/games", parameters, &payload); err != nil {
		return nil, err
	}
	requested := make(map[int64]bool, len(universeIDs))
	for _, universeID := range universeIDs {
		requested[universeID] = true
	}
	gamesByUniverseID := make(map[int64]games.Game, len(payload.Data))
	for _, universe := range payload.Data {
		if !requested[universe.ID] || !games.ValidID(universe.RootPlaceID) || strings.TrimSpace(universe.Name) == "" {
			continue
		}
		place := games.Place{PlaceID: universe.RootPlaceID, UniverseID: universe.ID, Name: universe.Name, CreatorName: universe.Creator.Name, CreatorVerified: universe.Creator.Verified}
		if validGameCreator(universe.Creator.ID, universe.Creator.Type) {
			place.CreatorID, place.CreatorType = universe.Creator.ID, universe.Creator.Type
		}
		game := games.Game{
			Place: place, Description: universe.Description,
			Playing: universe.Playing, Visits: universe.Visits, Favorites: universe.FavoritedCount, MaxPlayers: universe.MaxPlayers,
			Genre: gameGenre(universe.Genre), Subgenre: gameGenre(universe.Subgenre), AvatarType: gameAvatarType(universe.AvatarType),
			CopyingAllowed: universe.CopyingAllowed, PrivateServersAllowed: universe.PrivateServersAllowed,
			CreatedAtMs: gameTimestamp(universe.Created), UpdatedAtMs: gameTimestamp(universe.Updated),
		}
		if universe.Price != nil {
			game.Price = max(*universe.Price, 0)
		}
		gamesByUniverseID[universe.ID] = game
	}
	if len(gamesByUniverseID) == 0 {
		return gamesByUniverseID, nil
	}
	icons, err := service.icons(ctx, "/v1/games/icons", parameters)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		service.logger.WarnContext(ctx, "game icons unavailable", "operation", "game-icons", "error", err)
	}
	for universeID, game := range gamesByUniverseID {
		game.Place.IconURL = icons[universeID]
		gamesByUniverseID[universeID] = game
		service.cache.SavePlace(ctx, game.Place)
	}
	return gamesByUniverseID, nil
}

func (service *Games) placeIcons(ctx context.Context, placeIDs []int64) (map[int64]string, error) {
	return service.icons(ctx, "/v1/places/gameicons", url.Values{"placeIds": {gameIDs(placeIDs)}})
}

func (service *Games) icons(ctx context.Context, path string, parameters url.Values) (map[int64]string, error) {
	parameters.Set("returnPolicy", "PlaceHolder")
	parameters.Set("size", "150x150")
	parameters.Set("format", "Png")
	parameters.Set("isCircular", "false")
	var payload struct {
		Data []struct {
			TargetID int64  `json:"targetId"`
			State    string `json:"state"`
			ImageURL string `json:"imageUrl"`
		} `json:"data"`
	}
	if err := service.get(ctx, "game-icons", "thumbnails.roblox.com", path, parameters, &payload); err != nil {
		return nil, err
	}
	imageURLByTargetID := make(map[int64]string, len(payload.Data))
	for _, thumbnail := range payload.Data {
		if thumbnail.State == "Completed" && safeThumbnailURL(thumbnail.ImageURL) {
			imageURLByTargetID[thumbnail.TargetID] = thumbnail.ImageURL
		}
	}
	return imageURLByTargetID, nil
}

func (service *Games) get(ctx context.Context, operation, host, path string, parameters url.Values, target any) error {
	endpoint := &url.URL{Scheme: "https", Host: host, Path: path, RawQuery: parameters.Encode()}
	return service.do(ctx, operation, http.MethodGet, endpoint, nil, target)
}

func (service *Games) do(ctx context.Context, operation, method string, endpoint *url.URL, body []byte, target any) error {
	request := roblox.Request{
		Endpoint: operation, Method: method, URL: endpoint,
		MaxResponseSize: 2 << 20,
		Retry:           roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
	}
	if body != nil {
		request.Body, request.ContentType = body, "application/json"
	}
	response, err := service.client.Do(ctx, request)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(response.Body, target); err != nil {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: operation, Message: "Roblox returned invalid game data.", Cause: err}
	}
	return nil
}

func (service *Games) account(ctx context.Context, operation string, accountID int64, method string, endpoint *url.URL, body []byte, target any) error {
	request := roblox.Request{
		Endpoint: operation, AccountID: accountID, Authenticated: true, Method: method, URL: endpoint,
		MaxResponseSize: 64 << 10,
		Retry:           roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
	}
	if body != nil {
		request.Body, request.ContentType, request.RequiresCSRF = body, "application/json", true
	}
	response, err := service.client.Do(ctx, request)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(response.Body, target); err != nil {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: operation, Message: "Roblox returned invalid game data.", Cause: err}
	}
	return nil
}

func randomUUID() string {
	var id [16]byte
	rand.Read(id[:])
	// Mark the ID as a random (version 4) UUID with the RFC 4122 variant.
	id[6], id[8] = id[6]&0x0f|0x40, id[8]&0x3f|0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", id[:4], id[4:6], id[6:8], id[8:10], id[10:])
}

func gameIDs(ids []int64) string {
	formattedIDs := make([]string, len(ids))
	for index, id := range ids {
		formattedIDs[index] = strconv.FormatInt(id, 10)
	}
	return strings.Join(formattedIDs, ",")
}

func gameTimestamp(timestamp time.Time) int64 {
	if timestamp.IsZero() {
		return 0
	}
	return timestamp.UnixMilli()
}

// robloxTimeMs reads a Roblox timestamp, which some endpoints write without a
// time zone although they mean UTC. It returns 0 for an invalid timestamp.
func robloxTimeMs(timestamp string) int64 {
	timestamp = strings.TrimSpace(timestamp)
	for _, candidate := range []string{timestamp, timestamp + "Z"} {
		if parsed, err := time.Parse(time.RFC3339Nano, candidate); err == nil {
			return parsed.UnixMilli()
		}
	}
	return 0
}

func gameGenre(genre string) string {
	genre = strings.TrimSpace(genre)
	if strings.EqualFold(genre, "All") {
		return ""
	}
	return genre
}

func gameAvatarType(avatarType string) string {
	switch avatarType {
	case "MorphToR6":
		return "R6"
	case "MorphToR15":
		return "R15"
	case "PlayerChoice":
		return "Player choice"
	}
	return ""
}

func gameUnavailableError() error {
	return gameDataError("This game is private, deleted, or unavailable.")
}

func gameDataError(message string) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "games", Message: message}
}

func (service *Games) loadSearchPlaces(ctx context.Context, universeIDs []int64, page *games.SearchPage) error {
	for start := 0; start < len(universeIDs); start += gameBatchSize {
		batch := universeIDs[start:min(start+gameBatchSize, len(universeIDs))]
		details, err := service.details(ctx, batch)
		if err != nil {
			return err
		}
		for _, universeID := range batch {
			if game, exists := details[universeID]; exists {
				page.Places = append(page.Places, game.Place)
			}
		}
	}
	return nil
}

// Section reads one part of the game details for the universe. The sections
// that need a signed-in account stay empty without one.
func (service *Games) Section(ctx context.Context, universeID, placeID, accountID int64, section games.GameSection) (games.Game, error) {
	game := games.Game{Place: games.Place{UniverseID: universeID, PlaceID: placeID}}
	var err error
	switch section {
	case games.GameSectionVotes:
		err = service.votes(ctx, &game)
	case games.GameSectionMaturity:
		err = service.maturity(ctx, &game)
	case games.GameSectionAvatarRules:
		err = service.avatarRules(ctx, &game)
	case games.GameSectionCommunication:
		if accountID > 0 {
			err = service.communication(ctx, accountID, &game)
		}
	case games.GameSectionVersions:
		if accountID > 0 {
			err = service.versions(ctx, accountID, &game)
		}
	default:
		return games.Game{}, gameDataError("This game section does not exist.")
	}
	return game, err
}

func validGameCreator(creatorID int64, kind string) bool {
	return creatorID > 0 && creatorID < 1<<53 && (kind == "User" || kind == "Group")
}

func (service *Games) CachedPlace(ctx context.Context, placeID int64) (games.Place, bool) {
	return service.cache.Place(ctx, placeID)
}
