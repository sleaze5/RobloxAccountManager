package services

import (
	"context"
	"net/http"
	"net/url"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

// Moderation reads the punishment of a moderated account and caches it,
// because the punishment is public metadata that Roblox returns again.
func (service *Users) Moderation(ctx context.Context, accountID, version, userID int64) (accounts.Moderation, error) {
	endpoint, _ := url.Parse(roblox.ModerationURL)
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: "account-moderation", AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, CanUseModeratedSession: true, Method: http.MethodGet, URL: endpoint,
		MaxResponseSize: 64 << 10,
		Retry:           roblox.RetryPolicy{MaxAttempts: 2, RetryServerErrors: true, RetryRateLimit: true},
	})
	if err != nil {
		return accounts.Moderation{}, err
	}
	moderation, _, found := roblox.DecodeModeration(response.Body)
	if !found {
		return accounts.Moderation{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "account-moderation", Status: response.Status, Message: "Roblox returned invalid moderation details."}
	}
	service.cache.SaveModeration(ctx, userID, moderation)
	return moderation, nil
}

// CachedModeration returns the last punishment read for a user, or nil.
func (service *Users) CachedModeration(ctx context.Context, userID int64) *accounts.Moderation {
	return service.cache.Moderation(ctx, userID)
}

// SaveModeration caches a punishment that cookie validation read.
func (service *Users) SaveModeration(ctx context.Context, userID int64, moderation accounts.Moderation) {
	service.cache.SaveModeration(ctx, userID, moderation)
}

// Reactivate accepts the moderation of a warned account, or of a banned
// account whose ban ended, so that the account works again.
func (service *Users) Reactivate(ctx context.Context, accountID, version int64) error {
	endpoint, _ := url.Parse(roblox.ModerationURL + "/reactivate")
	_, err := service.client.Do(ctx, roblox.Request{
		Endpoint: "account-reactivate", AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, CanUseModeratedSession: true, Method: http.MethodPost, URL: endpoint,
		RequiresCSRF: true, MaxResponseSize: 64 << 10,
	})
	return err
}
