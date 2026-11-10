package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/gamelaunch"
	"github.com/sleaze5/RobloxAccountManager/internal/games"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	placePrivateServersPageSize = 25
	ownedPrivateServersPageSize = 50
	maxPrivateServerTextLength  = 200
	maxLinkCodePages            = 20
)

type PrivateServers struct {
	client *roblox.Client
}

func NewPrivateServers(client *roblox.Client) *PrivateServers {
	return &PrivateServers{client: client}
}

type privateServerUser struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

// Set one group of fields: Name, Active, and NewJoinCode; FriendsAllowed and users; or Renews.
type PrivateServerUpdate struct {
	Name           *string
	Active         *bool
	NewJoinCode    *bool
	FriendsAllowed *bool
	UsersToAdd     []int64
	UsersToRemove  []int64
	Renews         *bool
}

// Undocumented. Shape from a capture of www.roblox.com/my/account#!/subscriptions on 2026-10-14, not yet verified against a signed-in response.
func (service *PrivateServers) Owned(ctx context.Context, accountID, version int64, cursor string) (games.OwnedPrivateServerPage, error) {
	query := url.Values{"privateServersTab": {"0"}, "itemsPerPage": {strconv.Itoa(ownedPrivateServersPageSize)}}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	var payload struct {
		NextPageCursor string `json:"nextPageCursor"`
		Data           []struct {
			PrivateServerID int64  `json:"privateServerId"`
			Name            string `json:"name"`
			Active          bool   `json:"active"`
			UniverseID      int64  `json:"universeId"`
			PlaceID         int64  `json:"placeId"`
			UniverseName    string `json:"universeName"`
			PriceInRobux    *int64 `json:"priceInRobux"`
			WillRenew       bool   `json:"willRenew"`
			ExpirationDate  string `json:"expirationDate"`
		} `json:"data"`
	}
	if err := service.read(ctx, accountID, version, "private-servers-owned", "/v1/private-servers/my-private-servers", query, &payload); err != nil {
		return games.OwnedPrivateServerPage{}, err
	}
	if len(payload.NextPageCursor) > maxServerCursorLength {
		return games.OwnedPrivateServerPage{}, privateServerDataError("private-servers-owned")
	}
	page := games.OwnedPrivateServerPage{Servers: make([]games.OwnedPrivateServer, 0, len(payload.Data)), NextCursor: payload.NextPageCursor}
	for _, listedServer := range payload.Data {
		if !games.ValidID(listedServer.PrivateServerID) {
			continue
		}
		server := games.OwnedPrivateServer{
			ServerID: listedServer.PrivateServerID, Name: privateServerText(listedServer.Name), Active: listedServer.Active,
			GameName: privateServerText(listedServer.UniverseName), Renews: listedServer.WillRenew, ExpiresAtMs: privateServerTime(listedServer.ExpirationDate),
		}
		if games.ValidID(listedServer.UniverseID) {
			server.UniverseID = listedServer.UniverseID
		}
		if games.ValidID(listedServer.PlaceID) {
			server.PlaceID = listedServer.PlaceID
		}
		if listedServer.PriceInRobux != nil {
			server.Price = max(*listedServer.PriceInRobux, 0)
		}
		page.Servers = append(page.Servers, server)
	}
	return page, nil
}

type placePrivateServerPage struct {
	NextPageCursor string `json:"nextPageCursor"`
	Data           []struct {
		VipServerID int64             `json:"vipServerId"`
		ID          string            `json:"id"`
		Name        string            `json:"name"`
		AccessCode  string            `json:"accessCode"`
		Playing     int64             `json:"playing"`
		MaxPlayers  int64             `json:"maxPlayers"`
		Owner       privateServerUser `json:"owner"`
	} `json:"data"`
}

func (service *PrivateServers) ForPlace(ctx context.Context, accountID, version, ownerID int64, query games.PrivateServerQuery) (games.PrivateServerPage, error) {
	payload, err := service.placePage(ctx, accountID, version, query.PlaceID, query.Cursor)
	if err != nil {
		return games.PrivateServerPage{}, err
	}
	page := games.PrivateServerPage{Servers: make([]games.PrivateServer, 0, len(payload.Data)), NextCursor: payload.NextPageCursor}
	for _, listedServer := range payload.Data {
		if !games.ValidID(listedServer.VipServerID) {
			continue
		}
		server := games.PrivateServer{
			ServerID: listedServer.VipServerID, Name: privateServerText(listedServer.Name),
			OwnerID: max(listedServer.Owner.ID, 0), OwnerName: privateServerText(listedServer.Owner.Name), OwnerDisplayName: privateServerText(listedServer.Owner.DisplayName),
			Owned: ownerID > 0 && listedServer.Owner.ID == ownerID, Playing: max(listedServer.Playing, 0), MaxPlayers: max(listedServer.MaxPlayers, 0),
			Running: games.ValidJobID(listedServer.ID),
		}
		if games.ValidJobID(listedServer.AccessCode) {
			server.LaunchLink = games.PrivateServerLaunchLink(query.PlaceID, listedServer.AccessCode, "")
		}
		page.Servers = append(page.Servers, server)
	}
	return page, nil
}

// Roblox shows the link only to the owner, so it returns "" for other servers.
func (service *PrivateServers) LinkCode(ctx context.Context, accountID, version, ownerID, placeID int64, accessCode string) (string, error) {
	cursor := ""
	for range maxLinkCodePages {
		page, err := service.placePage(ctx, accountID, version, placeID, cursor)
		if err != nil {
			return "", err
		}
		for _, server := range page.Data {
			if !strings.EqualFold(server.AccessCode, accessCode) {
				continue
			}
			if ownerID <= 0 || server.Owner.ID != ownerID || !games.ValidID(server.VipServerID) {
				return "", nil
			}
			settings, err := service.Settings(ctx, accountID, version, server.VipServerID)
			if err != nil || settings.JoinLink == "" {
				return "", err
			}
			target, err := gamelaunch.Parse(gamelaunch.Input{Method: gamelaunch.MethodLink, Link: settings.JoinLink})
			if err != nil {
				return "", privateServerDataError("private-server")
			}
			return target.LinkCode, nil
		}
		if page.NextPageCursor == "" {
			return "", nil
		}
		cursor = page.NextPageCursor
	}
	return "", nil
}

func (service *PrivateServers) placePage(ctx context.Context, accountID, version, placeID int64, cursor string) (placePrivateServerPage, error) {
	parameters := url.Values{"limit": {strconv.Itoa(placePrivateServersPageSize)}}
	if cursor != "" {
		parameters.Set("cursor", cursor)
	}
	var payload placePrivateServerPage
	path := "/v1/games/" + strconv.FormatInt(placeID, 10) + "/private-servers"
	if err := service.read(ctx, accountID, version, "private-servers-place", path, parameters, &payload); err != nil {
		return placePrivateServerPage{}, err
	}
	if len(payload.Data) > 100 || len(payload.NextPageCursor) > maxServerCursorLength {
		return placePrivateServerPage{}, privateServerDataError("private-servers-place")
	}
	return payload, nil
}

func (service *PrivateServers) Settings(ctx context.Context, accountID, version, serverID int64) (games.PrivateServerSettings, error) {
	var payload struct {
		ID     int64  `json:"id"`
		Name   string `json:"name"`
		Active bool   `json:"active"`
		Link   string `json:"link"`
		Game   *struct {
			Name string `json:"name"`
		} `json:"game"`
		Subscription *struct {
			Active         bool   `json:"active"`
			Expired        bool   `json:"expired"`
			ExpirationDate string `json:"expirationDate"`
			Price          *int64 `json:"price"`
		} `json:"subscription"`
		Permissions *struct {
			FriendsAllowed bool                `json:"friendsAllowed"`
			Users          []privateServerUser `json:"users"`
		} `json:"permissions"`
	}
	if err := service.read(ctx, accountID, version, "private-server", privateServerPath(serverID, ""), nil, &payload); err != nil {
		return games.PrivateServerSettings{}, err
	}
	if payload.ID != serverID {
		return games.PrivateServerSettings{}, privateServerDataError("private-server")
	}
	settings := games.PrivateServerSettings{ServerID: serverID, Name: privateServerText(payload.Name), Active: payload.Active, Users: []games.PrivateServerUser{}}
	if payload.Game != nil {
		settings.GameName = privateServerText(payload.Game.Name)
	}
	if link, err := url.Parse(payload.Link); err == nil && link.Scheme == "https" && strings.EqualFold(link.Hostname(), "www.roblox.com") {
		settings.JoinLink = link.String()
	}
	if subscription := payload.Subscription; subscription != nil {
		settings.Renews, settings.Expired = subscription.Active, subscription.Expired
		settings.ExpiresAtMs = privateServerTime(subscription.ExpirationDate)
		if subscription.Price != nil {
			settings.Price = max(*subscription.Price, 0)
		}
	}
	if permissions := payload.Permissions; permissions != nil {
		settings.FriendsAllowed = permissions.FriendsAllowed
		for _, user := range permissions.Users {
			if games.ValidID(user.ID) {
				settings.Users = append(settings.Users, games.PrivateServerUser{UserID: user.ID, Username: privateServerText(user.Name), DisplayName: privateServerText(user.DisplayName)})
			}
		}
	}
	return settings, nil
}

func (service *PrivateServers) Update(ctx context.Context, accountID, version, serverID int64, update PrivateServerUpdate) error {
	var path string
	var body any
	switch {
	case update.Name != nil || update.Active != nil || update.NewJoinCode != nil:
		body = struct {
			Name        *string `json:"name,omitempty"`
			Active      *bool   `json:"active,omitempty"`
			NewJoinCode *bool   `json:"newJoinCode,omitempty"`
		}{update.Name, update.Active, update.NewJoinCode}
	case update.FriendsAllowed != nil || update.UsersToAdd != nil || update.UsersToRemove != nil:
		path = "permissions"
		body = struct {
			FriendsAllowed *bool   `json:"friendsAllowed,omitempty"`
			UsersToAdd     []int64 `json:"usersToAdd,omitempty"`
			UsersToRemove  []int64 `json:"usersToRemove,omitempty"`
		}{update.FriendsAllowed, update.UsersToAdd, update.UsersToRemove}
	case update.Renews != nil:
		path = "subscription"
		body = struct {
			Active bool `json:"active"`
		}{*update.Renews}
	default:
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "private-server-update", Message: "Choose a private server change."}
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "private-server-update", Message: "The request could not be encoded.", Cause: err}
	}
	endpoint := &url.URL{Scheme: "https", Host: "games.roblox.com", Path: privateServerPath(serverID, path)}
	_, err = service.client.Do(ctx, roblox.Request{
		Endpoint: "private-server-update", AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodPatch, URL: endpoint, Body: encoded,
		ContentType: "application/json", RequiresCSRF: true, MaxResponseSize: 64 << 10,
	})
	return err
}

func (service *PrivateServers) read(ctx context.Context, accountID, version int64, endpoint, path string, query url.Values, target any) error {
	address := &url.URL{Scheme: "https", Host: "games.roblox.com", Path: path, RawQuery: query.Encode()}
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: endpoint, AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodGet, URL: address, MaxResponseSize: 1 << 20,
		Retry: roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
	})
	if err != nil {
		return err
	}
	if err := json.Unmarshal(response.Body, target); err != nil {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Status: response.Status, Message: "Roblox returned invalid private server data.", Cause: err}
	}
	return nil
}

func privateServerPath(serverID int64, suffix string) string {
	path := "/v1/vip-servers/" + strconv.FormatInt(serverID, 10)
	if suffix != "" {
		path += "/" + suffix
	}
	return path
}

func privateServerText(rawText string) string {
	rawText = strings.TrimSpace(rawText)
	if !utf8.ValidString(rawText) || utf8.RuneCountInString(rawText) > maxPrivateServerTextLength {
		return ""
	}
	return rawText
}

func privateServerTime(timestamp string) int64 {
	timestamp = strings.TrimSpace(timestamp)
	for _, candidate := range []string{timestamp, timestamp + "Z"} {
		if parsed, err := time.Parse(time.RFC3339Nano, candidate); err == nil {
			return parsed.UnixMilli()
		}
	}
	return 0
}

func privateServerDataError(endpoint string) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Roblox returned invalid private server data."}
}
