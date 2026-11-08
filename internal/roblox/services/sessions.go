package services

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	sessionsPageSize = 25
	maxSessionsPages = 40
)

type AccountSession struct {
	Agent       string
	Browser     string
	OS          string
	City        string
	Subdivision string
	Country     string
	IP          string
	LastActive  time.Time
	Current     bool
}

// Shape verified against signed-in responses on 2026-11-07. Roblox adds the
// current session to the first page only.
func (service *Users) Sessions(ctx context.Context, accountID, version int64) ([]AccountSession, error) {
	sessions := []AccountSession{}
	seen := map[string]struct{}{}
	cursor := ""
	for range maxSessionsPages {
		var payload struct {
			Sessions []struct {
				Token string `json:"token"`
				Agent *struct {
					Type  *string `json:"type"`
					Value *string `json:"value"`
					OS    *string `json:"os"`
				} `json:"agent"`
				Location *struct {
					City        *string `json:"city"`
					Subdivision *string `json:"subdivision"`
					Country     *string `json:"country"`
				} `json:"location"`
				IP           *string `json:"lastAccessedIp"`
				LastAccessed *string `json:"lastAccessedTimestampEpochMilliseconds"`
				Current      bool    `json:"isCurrentSession"`
			} `json:"sessions"`
			HasMore    bool    `json:"hasMore"`
			NextCursor *string `json:"nextCursor"`
		}
		query := url.Values{"desiredLimit": {strconv.Itoa(sessionsPageSize)}}
		if cursor != "" {
			query.Set("nextCursor", cursor)
		}
		if err := service.readAccountDetails(ctx, accountID, version, "account-sessions", "https://apis.roblox.com/token-metadata-service/v1/sessions?"+query.Encode(), &payload); err != nil {
			return nil, err
		}
		for _, listedSession := range payload.Sessions {
			if _, exists := seen[listedSession.Token]; listedSession.Token == "" || exists {
				continue
			}
			seen[listedSession.Token] = struct{}{}
			session := AccountSession{IP: sessionText(listedSession.IP), Current: listedSession.Current}
			if listedSession.Agent != nil {
				session.Agent, session.Browser, session.OS = sessionText(listedSession.Agent.Type), sessionText(listedSession.Agent.Value), sessionText(listedSession.Agent.OS)
			}
			if listedSession.Location != nil {
				session.City, session.Subdivision, session.Country = sessionText(listedSession.Location.City), sessionText(listedSession.Location.Subdivision), sessionText(listedSession.Location.Country)
			}
			if listedSession.LastAccessed != nil {
				if milliseconds, err := strconv.ParseInt(*listedSession.LastAccessed, 10, 64); err == nil && milliseconds > 0 {
					session.LastActive = time.UnixMilli(milliseconds).UTC()
				}
			}
			sessions = append(sessions, session)
		}
		if !payload.HasMore || payload.NextCursor == nil || *payload.NextCursor == "" {
			return sessions, nil
		}
		cursor = *payload.NextCursor
	}
	return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "account-sessions", Message: "Roblox returned too many sessions to list."}
}

// Roblox replaces the session cookie in this response. The client saves it.
func (service *Auth) SignOutOtherSessions(ctx context.Context, accountID, version int64) error {
	endpoint, _ := url.Parse("https://auth.roblox.com/v2/logoutfromallsessionsandreauthenticate")
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: "sessions-sign-out", AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodPost, URL: endpoint,
		Body: []byte("{}"), ContentType: "application/json",
		RequiresCSRF: true, MaxResponseSize: 64 << 10,
	})
	if response != nil && response.Status >= 200 && response.Status < 300 && response.RotatedSecretVersion > version {
		return nil
	}
	if err != nil {
		return err
	}
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "sessions-sign-out", Message: "Roblox did not return a new cookie. Refresh sessions to check whether other sessions were signed out."}
}

// Roblox writes "Unknown" when it cannot identify a value.
func sessionText(value *string) string {
	if value == nil {
		return ""
	}
	text := strings.TrimSpace(*value)
	if strings.EqualFold(text, "unknown") || len(text) > 256 {
		return ""
	}
	return text
}
