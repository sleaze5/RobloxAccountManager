package services

import (
	"context"
	"encoding/json"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/sleaze5/RobloxAccountManager/internal/gamelaunch"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

type GameLinks struct {
	client *roblox.Client
}

func NewGameLinks(client *roblox.Client) *GameLinks {
	return &GameLinks{client: client}
}

type privateServerInvite struct {
	Status     string `json:"status"`
	PlaceID    int64  `json:"placeId"`
	UniverseID int64  `json:"universeId"`
	LinkCode   string `json:"linkCode"`
}

type experienceInvite struct {
	Status     string `json:"status"`
	PlaceID    int64  `json:"placeId"`
	InstanceID string `json:"instanceId"`
	LaunchData string `json:"launchData"`
}

var privateAccessPatterns = []*regexp.Regexp{
	// joinPrivateGame takes the place ID before the access code and link code.
	regexp.MustCompile(`(?i)Roblox\.GameLauncher\.joinPrivateGame\s*\(\s*(?:[0-9]+|"[0-9]+"|'[0-9]+')\s*,\s*["']([a-z0-9_-]{1,512})["']`),
	regexp.MustCompile(`(?i)["']accessCode["']\s*:\s*["']([a-z0-9_-]{1,512})["']`),
	regexp.MustCompile(`(?i)\baccessCode\s*=\s*["']([a-z0-9_-]{1,512})["']`),
}

func (service *GameLinks) Resolve(ctx context.Context, accountID, version int64, target gamelaunch.Request) (gamelaunch.Request, error) {
	if target.ShareCode != "" {
		resolved, err := service.resolveShare(ctx, accountID, version, target.ShareCode, target.ShareType)
		if err != nil {
			return gamelaunch.Request{}, err
		}
		if target.LaunchData != "" {
			resolved.LaunchData = target.LaunchData
		}
		target = resolved
	}
	if target.LinkCode == "" || target.AccessCode != "" {
		return target, nil
	}
	endpoint := &url.URL{Scheme: "https", Host: "www.roblox.com", Path: "/games/" + strconv.FormatInt(target.PlaceID, 10)}
	endpoint.RawQuery = url.Values{"privateServerLinkCode": {target.LinkCode}}.Encode()
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: "private-server-link", AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodGet, URL: endpoint, MaxResponseSize: 4 << 20,
		FollowRedirects: true, Headers: http.Header{"Accept": {"text/html"}},
		Retry: roblox.RetryPolicy{MaxAttempts: 3, RetryServerErrors: true, RetryRateLimit: true},
	})
	if err != nil {
		return gamelaunch.Request{}, err
	}
	body := strings.ReplaceAll(html.UnescapeString(string(response.Body)), `\"`, `"`)
	for _, pattern := range privateAccessPatterns {
		if match := pattern.FindStringSubmatch(body); len(match) == 2 && !strings.EqualFold(match[1], "null") && !strings.EqualFold(match[1], "undefined") {
			target.AccessCode = match[1]
			return target, nil
		}
	}
	return gamelaunch.Request{}, gameLinkError("private-server-link", response.Status, "Roblox did not return a private-server access code. Check that the link is active and this account can join it.")
}

func (service *GameLinks) resolveShare(ctx context.Context, accountID, version int64, code string, shareType gamelaunch.ShareType) (gamelaunch.Request, error) {
	if shareType != gamelaunch.ShareServer && shareType != gamelaunch.ShareExperienceInvite {
		return gamelaunch.Request{}, gameLinkError("share-link", 0, "This share-link type cannot be used to join a game.")
	}
	body, err := json.Marshal(struct {
		LinkID   string               `json:"linkId"`
		LinkType gamelaunch.ShareType `json:"linkType"`
	}{LinkID: code, LinkType: shareType})
	if err != nil {
		return gamelaunch.Request{}, err
	}
	endpoint, _ := url.Parse("https://apis.roblox.com/sharelinks/v1/resolve-link")
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: "share-link", AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodPost, URL: endpoint,
		Body: body, ContentType: "application/json", RequiresCSRF: true,
		Referer: "https://www.roblox.com/share-links", MaxResponseSize: 256 << 10,
		Retry: roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
	})
	if err != nil {
		return gamelaunch.Request{}, err
	}
	var result struct {
		PrivateServerInviteData *privateServerInvite `json:"privateServerInviteData"`
		ServerData              *privateServerInvite `json:"serverData"`
		ExperienceInviteData    *experienceInvite    `json:"experienceInviteData"`
	}
	if err := json.Unmarshal(response.Body, &result); err != nil {
		return gamelaunch.Request{}, gameLinkError("share-link", response.Status, "Roblox returned invalid share-link details.")
	}
	if shareType == gamelaunch.ShareExperienceInvite {
		return experienceInviteRequest(result.ExperienceInviteData, response.Status)
	}
	invite := result.PrivateServerInviteData
	if invite == nil {
		invite = result.ServerData
	}
	return service.resolvePrivateInvite(ctx, invite, response.Status)
}

func (service *GameLinks) resolvePrivateInvite(ctx context.Context, invite *privateServerInvite, status int) (gamelaunch.Request, error) {
	if invite == nil || strings.TrimSpace(invite.LinkCode) == "" {
		return gamelaunch.Request{}, gameLinkError("share-link", status, "The server-share link is invalid, expired, or unavailable to this account.")
	}
	if inviteStatus := strings.TrimSpace(invite.Status); inviteStatus != "" && !strings.EqualFold(inviteStatus, "Valid") {
		return gamelaunch.Request{}, gameLinkError("share-link", status, "The server-share link is no longer valid. Request a new link.")
	}
	if invite.PlaceID <= 0 && invite.UniverseID > 0 {
		var err error
		invite.PlaceID, err = service.rootPlace(ctx, invite.UniverseID)
		if err != nil {
			return gamelaunch.Request{}, err
		}
	}
	// Reuse launch validation before using remote data in another request.
	link := url.URL{Scheme: "https", Host: "www.roblox.com", Path: "/games/" + strconv.FormatInt(invite.PlaceID, 10)}
	link.RawQuery = url.Values{"privateServerLinkCode": {strings.TrimSpace(invite.LinkCode)}}.Encode()
	target, err := gamelaunch.Parse(gamelaunch.Input{Method: gamelaunch.MethodLink, Link: link.String()})
	if err != nil {
		return gamelaunch.Request{}, gameLinkError("share-link", status, "Roblox returned an invalid private-server destination.")
	}
	return target, nil
}

func experienceInviteRequest(invite *experienceInvite, status int) (gamelaunch.Request, error) {
	if invite == nil {
		return gamelaunch.Request{}, gameLinkError("experience-invite", status, "This experience invite is invalid or unavailable to this account.")
	}
	switch strings.ToLower(strings.TrimSpace(invite.Status)) {
	case "valid":
	case "expired":
		return gamelaunch.Request{}, gameLinkError("experience-invite", status, "This experience invite has expired. Request a new link.")
	case "inviternotinexperience":
		return gamelaunch.Request{}, gameLinkError("experience-invite", status, "The person who sent this invite is no longer in the experience. Request a new link.")
	default:
		return gamelaunch.Request{}, gameLinkError("experience-invite", status, "This experience invite is no longer valid. Request a new link.")
	}
	if strings.TrimSpace(invite.InstanceID) == "" {
		return gamelaunch.Request{}, gameLinkError("experience-invite", status, "This experience invite has no joinable server. Request a new link.")
	}
	target, err := gamelaunch.Parse(gamelaunch.Input{Method: gamelaunch.MethodPlace, PlaceID: strconv.FormatInt(invite.PlaceID, 10), JobID: invite.InstanceID})
	if err != nil {
		return gamelaunch.Request{}, gameLinkError("experience-invite", status, "Roblox returned an invalid server for this experience invite.")
	}
	target.LaunchData = invite.LaunchData
	return target, nil
}

func (service *GameLinks) rootPlace(ctx context.Context, universeID int64) (int64, error) {
	endpoint := &url.URL{Scheme: "https", Host: "games.roblox.com", Path: "/v1/games"}
	endpoint.RawQuery = url.Values{"universeIds": {strconv.FormatInt(universeID, 10)}}.Encode()
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: "server-share-game", Method: http.MethodGet, URL: endpoint, MaxResponseSize: 256 << 10,
		Retry: roblox.RetryPolicy{MaxAttempts: 3, RetryServerErrors: true, RetryRateLimit: true},
	})
	if err != nil {
		return 0, err
	}
	var result struct {
		Data []struct {
			ID          int64 `json:"id"`
			RootPlaceID int64 `json:"rootPlaceId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body, &result); err != nil {
		return 0, gameLinkError("server-share-game", response.Status, "Roblox returned invalid game details for this server-share link.")
	}
	if len(result.Data) != 1 || result.Data[0].ID != universeID || result.Data[0].RootPlaceID <= 0 {
		return 0, gameLinkError("server-share-game", response.Status, "The game for this server-share link is unavailable.")
	}
	return result.Data[0].RootPlaceID, nil
}

func gameLinkError(endpoint string, status int, message string) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Status: status, Message: message}
}
