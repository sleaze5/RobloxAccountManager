package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

type AuthenticationTicket struct {
	value string
}

func (ticket AuthenticationTicket) Value() string { return ticket.value }

type Authentication struct {
	client                  *roblox.Client
	logger                  *slog.Logger
	authenticationTicketURL *url.URL
	clientAssertionURL      *url.URL
}

func NewAuthentication(client *roblox.Client, logger *slog.Logger) *Authentication {
	authenticationTicketURL, _ := url.Parse("https://auth.roblox.com/v1/authentication-ticket")
	clientAssertionURL, _ := url.Parse("https://auth.roblox.com/v1/client-assertion")
	return &Authentication{client: client, logger: logger, authenticationTicketURL: authenticationTicketURL, clientAssertionURL: clientAssertionURL}
}

func (service *Authentication) RefreshCookie(ctx context.Context, accountID, version int64) error {
	endpoint, _ := url.Parse("https://auth.roblox.com/v2/session/refresh")
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: "session-refresh", AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodPost, URL: endpoint,
		RequiresCSRF: true, MaxResponseSize: 64 << 10,
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

type authenticationTicketBody struct {
	ClientAssertion string `json:"clientAssertion,omitempty"`
}

// AuthenticationTicket sends a client assertion with the ticket request, as the
// Play button on www.roblox.com does. The Roblox Player does not join with a
// ticket that was issued without one. Like the website, a failed assertion
// request still requests the ticket.
func (service *Authentication) AuthenticationTicket(ctx context.Context, accountID int64) (AuthenticationTicket, error) {
	clientAssertion, err := service.clientAssertion(ctx, accountID)
	if err != nil {
		if ctx.Err() != nil {
			return AuthenticationTicket{}, err
		}
		service.logger.WarnContext(ctx, "Requesting the authentication ticket without a client assertion", "operation", "client-assertion", "account_id", accountID, "error", err)
	}
	body, err := json.Marshal(authenticationTicketBody{ClientAssertion: clientAssertion})
	if err != nil {
		return AuthenticationTicket{}, fmt.Errorf("encode authentication ticket request: %w", err)
	}
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint:                  "authentication-ticket",
		AccountID:                 accountID,
		Authenticated:             true,
		Method:                    http.MethodPost,
		URL:                       service.authenticationTicketURL,
		Body:                      body,
		ContentType:               "application/json",
		MaxResponseSize:           256 << 10,
		RequiresCSRF:              true,
		AuthenticationNegotiation: true,
		Referer:                   "https://www.roblox.com/",
	})
	if err != nil {
		return AuthenticationTicket{}, err
	}
	ticket := strings.TrimSpace(response.Header.Get("Rbx-Authentication-Ticket"))
	if ticket == "" {
		return AuthenticationTicket{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "authentication-ticket", Status: response.Status, Message: "Roblox did not return an authentication ticket."}
	}
	if strings.ContainsAny(ticket, "\r\n\x00+") {
		return AuthenticationTicket{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "authentication-ticket", Status: response.Status, Message: "Roblox returned an invalid authentication ticket."}
	}
	return AuthenticationTicket{value: ticket}, nil
}

func (service *Authentication) clientAssertion(ctx context.Context, accountID int64) (string, error) {
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint:        "client-assertion",
		AccountID:       accountID,
		Authenticated:   true,
		Method:          http.MethodGet,
		URL:             service.clientAssertionURL,
		MaxResponseSize: 64 << 10,
		Referer:         "https://www.roblox.com/",
		Retry:           roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
	})
	if err != nil {
		return "", err
	}
	var payload struct {
		ClientAssertion string `json:"clientAssertion"`
	}
	if err := json.Unmarshal(response.Body, &payload); err != nil {
		return "", &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "client-assertion", Status: response.Status, Message: "Roblox returned an invalid client assertion.", Cause: err}
	}
	if strings.TrimSpace(payload.ClientAssertion) == "" {
		return "", &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "client-assertion", Status: response.Status, Message: "Roblox did not return a client assertion.", Cause: errors.New("empty client assertion")}
	}
	return payload.ClientAssertion, nil
}
