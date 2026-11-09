package roblox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/appmeta"
)

const AuthenticatedUserURL = "https://users.roblox.com/v1/users/authenticated"

func DecodeAuthenticatedUser(body []byte) (accounts.Identity, bool) {
	var identity struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
	}
	if err := json.Unmarshal(body, &identity); err != nil || identity.ID <= 0 || strings.TrimSpace(identity.Name) == "" {
		return accounts.Identity{}, false
	}
	return accounts.Identity{RobloxUserID: identity.ID, Username: identity.Name, DisplayName: identity.DisplayName}, true
}

// newCookieRequest builds a GET that carries a raw cookie outside the request
// pipeline, for validation and session probes.
func newCookieRequest(ctx context.Context, address, cookie, browserID string) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create cookie request: %w", err)
	}
	request.Header.Set("User-Agent", appmeta.UserAgent)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Cookie", formatAuthCookie(cookie, browserID))
	return request, nil
}
