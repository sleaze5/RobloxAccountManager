package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

type UserView struct {
	UserID      int64  `json:"userId"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

type Users struct {
	client        *roblox.Client
	authenticated *url.URL
	profiles      *url.URL
}

func NewUsers(client *roblox.Client) *Users {
	authenticated, _ := url.Parse("https://users.roblox.com/v1/users/authenticated")
	profiles, _ := url.Parse("https://users.roblox.com/v1/users/")
	return &Users{client: client, authenticated: authenticated, profiles: profiles}
}

func (service *Users) ValidateCookie(ctx context.Context, cookie, browserID string) (roblox.CookieValidation, error) {
	validation, err := service.client.ValidateCookie(ctx, cookie, browserID)
	if err != nil {
		return roblox.CookieValidation{}, err
	}
	validation.Identity, err = service.Profile(ctx, validation.Identity.RobloxUserID)
	return validation, err
}

func (service *Users) Authenticated(ctx context.Context, accountID int64) (accounts.Identity, error) {
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint:        "authenticated-user",
		AccountID:       accountID,
		Authenticated:   true,
		Method:          http.MethodGet,
		URL:             service.authenticated,
		MaxResponseSize: 256 << 10,
		Retry: roblox.RetryPolicy{
			MaxAttempts:       3,
			RetryServerErrors: true,
			RetryRateLimit:    true,
		},
	})
	if err != nil {
		return accounts.Identity{}, err
	}
	var identity struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
	}
	if err := json.Unmarshal(response.Body, &identity); err != nil || identity.ID <= 0 || strings.TrimSpace(identity.Name) == "" {
		return accounts.Identity{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "authenticated-user", Status: response.Status, Message: "Roblox returned an invalid account identity.", Cause: fmt.Errorf("invalid authenticated-user response")}
	}
	profile, err := service.Profile(ctx, identity.ID)
	if err != nil {
		return accounts.Identity{}, err
	}
	return profile, nil
}

func (service *Users) Profile(ctx context.Context, userID int64) (accounts.Identity, error) {
	if userID <= 0 {
		return accounts.Identity{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "user-profile", Message: "The Roblox user ID is invalid."}
	}
	endpoint := *service.profiles
	endpoint.Path += strconv.FormatInt(userID, 10)
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint:        "user-profile",
		Method:          http.MethodGet,
		URL:             &endpoint,
		MaxResponseSize: 256 << 10,
		Retry: roblox.RetryPolicy{
			MaxAttempts:       3,
			RetryServerErrors: true,
			RetryRateLimit:    true,
		},
	})
	if err != nil {
		return accounts.Identity{}, err
	}
	var profile struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		Created     string `json:"created"`
	}
	if err := json.Unmarshal(response.Body, &profile); err != nil {
		return accounts.Identity{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "user-profile", Status: response.Status, Message: "Roblox returned invalid account details.", Cause: err}
	}
	createdAt, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(profile.Created))
	if parseErr != nil || profile.ID != userID || strings.TrimSpace(profile.Name) == "" {
		return accounts.Identity{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "user-profile", Status: response.Status, Message: "Roblox returned invalid account details.", Cause: parseErr}
	}
	return accounts.Identity{RobloxUserID: profile.ID, Username: profile.Name, DisplayName: profile.DisplayName, CreatedAt: createdAt.UTC()}, nil
}

func (service *Users) UserIDFromUsername(ctx context.Context, username string) (int64, error) {
	body, err := json.Marshal(struct {
		Usernames          []string `json:"usernames"`
		ExcludeBannedUsers bool     `json:"excludeBannedUsers"`
	}{Usernames: []string{username}, ExcludeBannedUsers: true})
	if err != nil {
		return 0, err
	}
	endpoint, _ := url.Parse("https://users.roblox.com/v1/usernames/users")
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: "username-lookup", Method: http.MethodPost, URL: endpoint,
		Body: body, ContentType: "application/json", MaxResponseSize: 256 << 10,
		Retry: roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
	})
	if err != nil {
		return 0, err
	}
	var result struct {
		Data []struct {
			ID                int64  `json:"id"`
			RequestedUsername string `json:"requestedUsername"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body, &result); err != nil {
		return 0, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "username-lookup", Message: "Roblox returned invalid username details.", Cause: err}
	}
	if len(result.Data) != 1 || result.Data[0].ID <= 0 || !strings.EqualFold(result.Data[0].RequestedUsername, username) {
		return 0, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "username-lookup", Message: "That username was not found or is unavailable. Enter a username, not a display name."}
	}
	return result.Data[0].ID, nil
}
