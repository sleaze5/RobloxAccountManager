package services

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

type AuthenticationTicket struct {
	value string
}

func (ticket AuthenticationTicket) Value() string { return ticket.value }

type Auth struct {
	client   *roblox.Client
	endpoint *url.URL
}

func NewAuth(client *roblox.Client) *Auth {
	endpoint, _ := url.Parse("https://auth.roblox.com/v1/authentication-ticket")
	return &Auth{client: client, endpoint: endpoint}
}

// RefreshCookie replaces only the current session. The shared client persists
// Set-Cookie with a version check; an ambiguous POST is never retried automatically.
func (service *Auth) RefreshCookie(ctx context.Context, accountID, version int64) error {
	endpoint, _ := url.Parse("https://auth.roblox.com/v2/session/refresh")
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: "session-refresh", AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodPost, URL: endpoint,
		RequiresCSRF: true, MaxResponseSize: 64 << 10,
		Retry: roblox.RetryPolicy{MaxAttempts: 1},
	})
	// This endpoint's body is empty. A successful status and saved replacement
	// are sufficient even if the connection closed before its body finished.
	if response != nil && response.Status >= 200 && response.Status < 300 && response.RotatedSecretVersion > version {
		return nil
	}
	if err != nil {
		return err
	}
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "session-refresh", Message: "Roblox did not return a renewed cookie. The saved cookie was not replaced by this response."}
}

func (service *Auth) AuthenticationTicket(ctx context.Context, accountID int64) (AuthenticationTicket, error) {
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint:                  "authentication-ticket",
		AccountID:                 accountID,
		Authenticated:             true,
		Method:                    http.MethodPost,
		URL:                       service.endpoint,
		Body:                      []byte{},
		ContentType:               "application/json",
		MaxResponseSize:           256 << 10,
		RequiresCSRF:              true,
		AuthenticationNegotiation: true,
		Referer:                   "https://www.roblox.com/",
		Retry:                     roblox.RetryPolicy{MaxAttempts: 1},
	})
	if err != nil {
		return AuthenticationTicket{}, err
	}
	value := strings.TrimSpace(response.Header.Get("rbx-authentication-ticket"))
	if value == "" {
		return AuthenticationTicket{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "authentication-ticket", Status: response.Status, Message: "Roblox did not return an authentication ticket."}
	}
	if strings.ContainsAny(value, "\r\n\x00+") {
		return AuthenticationTicket{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "authentication-ticket", Status: response.Status, Message: "Roblox returned an invalid authentication ticket."}
	}
	return AuthenticationTicket{value: value}, nil
}
