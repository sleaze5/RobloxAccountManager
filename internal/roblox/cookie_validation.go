package roblox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
)

type CookieValidation struct {
	Identity  accounts.Identity
	Cookie    string
	BrowserID string
	ExpiresAt *time.Time
}

// ValidateCookie validates an unpersisted cookie without creating a runtime
// session. It returns a server-rotated value when Roblox supplies one.
func (client *Client) ValidateCookie(ctx context.Context, cookie, browserID string) (CookieValidation, error) {
	normalized, err := accounts.NormalizeCookie(cookie)
	if err != nil {
		return CookieValidation{}, &Error{Kind: KindInvalidCookieInput, Endpoint: "cookie-import", Message: err.Error()}
	}
	cookie = normalized
	if browserID == "" {
		browserID = NewBrowserID()
	}
	if !client.credentialPolicy(client.authenticatedUserURL) {
		return CookieValidation{}, &Error{Kind: KindProtocol, Endpoint: "cookie-import", Message: "The cookie validation URL is outside the credential policy."}
	}
	requestContext, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	currentCookie := cookie
	var expires *time.Time
	for attempt := 1; attempt <= 3; attempt++ {
		request, err := http.NewRequestWithContext(requestContext, http.MethodGet, client.authenticatedUserURL.String(), nil)
		if err != nil {
			return CookieValidation{}, &Error{Kind: KindProtocol, Endpoint: "cookie-import", Message: "The cookie validation request is invalid.", Cause: err}
		}
		request.Header.Set("User-Agent", userAgent)
		request.Header.Set("Accept", "application/json")
		request.Header.Set("Cookie", formatAuthCookie(currentCookie, browserID))
		release, err := client.limits.acquire(requestContext, request.URL.Hostname(), 0)
		if err != nil {
			return CookieValidation{}, contextError("cookie-import", err)
		}
		response, err := client.http.Do(request)
		if err != nil {
			release()
			if requestContext.Err() != nil {
				return CookieValidation{}, contextError("cookie-import", requestContext.Err())
			}
			if attempt < 3 && retryableTransport(err) {
				if err := waitForRetry(requestContext, backoff(attempt, secureSample())); err != nil {
					return CookieValidation{}, contextError("cookie-import", err)
				}
				continue
			}
			return CookieValidation{}, &Error{Kind: KindTransport, Endpoint: "cookie-import", Message: "Roblox could not be reached. The cookie was not stored.", Cause: err, SafeToRetry: true}
		}
		body, readErr := readLimited(response.Body, defaultResponseLimit)
		response.Body.Close()
		release()
		if readErr != nil {
			if requestContext.Err() != nil {
				return CookieValidation{}, contextError("cookie-import", requestContext.Err())
			}
			if errors.Is(readErr, errResponseTooLarge) {
				return CookieValidation{}, &Error{Kind: KindResponseTooLarge, Endpoint: "cookie-import", Status: response.StatusCode, Message: "The cookie validation response was too large.", Cause: readErr}
			}
			return CookieValidation{}, &Error{Kind: KindTransport, Endpoint: "cookie-import", Status: response.StatusCode, Message: "The cookie validation response could not be read. The cookie was not stored.", Cause: readErr, SafeToRetry: true}
		}
		for _, replacement := range response.Cookies() {
			if replacement.Name != accounts.RoblosecurityCookieName {
				continue
			}
			if replacement.Value == "" || replacement.MaxAge < 0 {
				return CookieValidation{}, &Error{Kind: KindInvalidSession, Endpoint: "cookie-import", Status: response.StatusCode, Message: "Roblox rejected this cookie."}
			}
			if err := accounts.ValidateCookieValue(replacement.Value); err != nil {
				return CookieValidation{}, &Error{Kind: KindProtocol, Endpoint: "cookie-import", Message: "Roblox returned an invalid replacement cookie.", Cause: err}
			}
			currentCookie = replacement.Value
			if !replacement.Expires.IsZero() {
				value := replacement.Expires.UTC()
				if !value.After(client.now()) {
					return CookieValidation{}, &Error{Kind: KindInvalidSession, Endpoint: "cookie-import", Status: response.StatusCode, Message: "Roblox returned an expired session cookie."}
				}
				expires = &value
			}
		}
		switch {
		case response.StatusCode >= 200 && response.StatusCode < 300:
			var identity struct {
				ID          int64  `json:"id"`
				Name        string `json:"name"`
				DisplayName string `json:"displayName"`
			}
			if err := json.Unmarshal(body, &identity); err != nil || identity.ID <= 0 || strings.TrimSpace(identity.Name) == "" {
				return CookieValidation{}, &Error{Kind: KindProtocol, Endpoint: "cookie-import", Status: response.StatusCode, Message: "Roblox returned an invalid account identity."}
			}
			return CookieValidation{
				Identity:  accounts.Identity{RobloxUserID: identity.ID, Username: identity.Name, DisplayName: identity.DisplayName},
				Cookie:    currentCookie,
				BrowserID: browserID,
				ExpiresAt: expires,
			}, nil
		case response.StatusCode == http.StatusUnauthorized:
			return CookieValidation{}, &Error{Kind: KindInvalidSession, Endpoint: "cookie-import", Status: response.StatusCode, Message: "Roblox rejected this cookie."}
		case response.StatusCode == http.StatusForbidden:
			challengeType := strings.TrimSpace(response.Header.Get(challengeTypeHeader))
			if challengeType != "" {
				message := "Roblox requires user interaction before this cookie can be used."
				return CookieValidation{}, &Error{Kind: KindChallengeRequired, Endpoint: "cookie-import", Status: response.StatusCode, ChallengeType: challengeType, Message: message}
			}
			return CookieValidation{}, &Error{Kind: KindForbidden, Endpoint: "cookie-import", Status: response.StatusCode, Message: "Roblox rejected cookie validation."}
		case response.StatusCode == http.StatusTooManyRequests:
			delay := retryDelay(response.Header, attempt, client.now())
			if attempt < 3 {
				if err := waitForRetry(requestContext, delay); err != nil {
					return CookieValidation{}, contextError("cookie-import", err)
				}
				continue
			}
			return CookieValidation{}, &Error{Kind: KindRateLimited, Endpoint: "cookie-import", Status: response.StatusCode, RetryAt: client.now().Add(delay), Message: "Roblox is rate limiting cookie validation.", SafeToRetry: true}
		case response.StatusCode >= 500:
			if attempt < 3 {
				if err := waitForRetry(requestContext, retryDelay(response.Header, attempt, client.now())); err != nil {
					return CookieValidation{}, contextError("cookie-import", err)
				}
				continue
			}
			return CookieValidation{}, &Error{Kind: KindServer, Endpoint: "cookie-import", Status: response.StatusCode, Message: "Roblox is temporarily unavailable. The cookie was not stored.", SafeToRetry: true}
		default:
			return CookieValidation{}, &Error{Kind: KindProtocol, Endpoint: "cookie-import", Status: response.StatusCode, Message: fmt.Sprintf("Roblox returned status %d. The cookie was not stored.", response.StatusCode)}
		}
	}
	return CookieValidation{}, &Error{Kind: KindProtocol, Endpoint: "cookie-import", Message: "Cookie validation stopped unexpectedly.", Cause: io.EOF}
}
