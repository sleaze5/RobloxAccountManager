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
	"github.com/sleaze5/RobloxAccountManager/internal/appmeta"
)

type CookieValidation struct {
	Identity  accounts.Identity
	Cookie    string
	BrowserID string
	ExpiresAt *time.Time
}

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
	validation := CookieValidation{Cookie: cookie, BrowserID: browserID}
	for attempt := 1; attempt <= 3; attempt++ {
		response, retry, err := client.sendCookieValidation(requestContext, validation, attempt)
		if err != nil {
			return CookieValidation{}, err
		}
		if retry {
			continue
		}
		if err := client.applyValidationCookies(response, &validation); err != nil {
			return CookieValidation{}, err
		}
		result, retry, err := client.cookieValidationResponse(requestContext, response, validation, attempt)
		if retry {
			continue
		}
		return result, err
	}
	return CookieValidation{}, &Error{Kind: KindProtocol, Endpoint: "cookie-import", Message: "Cookie validation stopped unexpectedly.", Cause: io.EOF}
}

func (client *Client) sendCookieValidation(ctx context.Context, validation CookieValidation, attempt int) (*Response, bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.authenticatedUserURL.String(), http.NoBody)
	if err != nil {
		return nil, false, &Error{Kind: KindProtocol, Endpoint: "cookie-import", Message: "The cookie validation request is invalid.", Cause: err}
	}
	request.Header.Set("User-Agent", appmeta.UserAgent)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Cookie", formatAuthCookie(validation.Cookie, validation.BrowserID))
	release, err := client.limits.acquire(ctx, request.URL.Hostname(), 0)
	if err != nil {
		return nil, false, contextError("cookie-import", err)
	}
	response, err := client.http.Do(request)
	if err != nil {
		release()
		if ctx.Err() != nil {
			return nil, false, contextError("cookie-import", ctx.Err())
		}
		if attempt < 3 && retryableTransport(err) {
			if err := waitForRetry(ctx, backoff(attempt, secureSample())); err != nil {
				return nil, false, contextError("cookie-import", err)
			}
			return nil, true, nil
		}
		return nil, false, &Error{Kind: KindTransport, Endpoint: "cookie-import", Message: "Roblox could not be reached. The cookie was not stored.", Cause: err, SafeToRetry: true}
	}
	body, readErr := readLimited(response.Body, defaultResponseLimit)
	response.Body.Close()
	release()
	if readErr != nil {
		if ctx.Err() != nil {
			return nil, false, contextError("cookie-import", ctx.Err())
		}
		if errors.Is(readErr, errResponseTooLarge) {
			return nil, false, &Error{Kind: KindResponseTooLarge, Endpoint: "cookie-import", Status: response.StatusCode, Message: "The cookie validation response was too large.", Cause: readErr}
		}
		return nil, false, &Error{Kind: KindTransport, Endpoint: "cookie-import", Status: response.StatusCode, Message: "The cookie validation response could not be read. The cookie was not stored.", Cause: readErr, SafeToRetry: true}
	}
	return &Response{Status: response.StatusCode, Header: response.Header, Body: body}, false, nil
}

func (client *Client) applyValidationCookies(response *Response, validation *CookieValidation) error {
	cookies := (&http.Response{Header: response.Header}).Cookies()
	for _, replacement := range cookies {
		if replacement.Name != accounts.RoblosecurityCookieName {
			continue
		}
		if replacement.Value == "" || replacement.MaxAge < 0 {
			return &Error{Kind: KindInvalidSession, Endpoint: "cookie-import", Status: response.Status, Message: "Roblox rejected this cookie."}
		}
		if err := accounts.ValidateCookieValue(replacement.Value); err != nil {
			return &Error{Kind: KindProtocol, Endpoint: "cookie-import", Message: "Roblox returned an invalid replacement cookie.", Cause: err}
		}
		validation.Cookie = replacement.Value
		if !replacement.Expires.IsZero() {
			value := replacement.Expires.UTC()
			if !value.After(client.now()) {
				return &Error{Kind: KindInvalidSession, Endpoint: "cookie-import", Status: response.Status, Message: "Roblox returned an expired session cookie."}
			}
			validation.ExpiresAt = &value
		}
	}
	return nil
}

func (client *Client) cookieValidationResponse(ctx context.Context, response *Response, validation CookieValidation, attempt int) (CookieValidation, bool, error) {
	switch {
	case response.Status >= 200 && response.Status < 300:
		result, err := cookieValidationIdentity(response, validation)
		return result, false, err
	case response.Status == http.StatusUnauthorized:
		return CookieValidation{}, false, &Error{Kind: KindInvalidSession, Endpoint: "cookie-import", Status: response.Status, Message: "Roblox rejected this cookie."}
	case response.Status == http.StatusForbidden:
		challengeType := strings.TrimSpace(response.Header.Get(challengeTypeHeader))
		if challengeType != "" {
			message := "Roblox requires user interaction before this cookie can be used."
			return CookieValidation{}, false, &Error{Kind: KindChallengeRequired, Endpoint: "cookie-import", Status: response.Status, ChallengeType: challengeType, Message: message}
		}
		return CookieValidation{}, false, &Error{Kind: KindForbidden, Endpoint: "cookie-import", Status: response.Status, Message: "Roblox rejected cookie validation."}
	case response.Status == http.StatusTooManyRequests:
		delay := retryDelay(response.Header, attempt, client.now())
		if attempt < 3 {
			if err := waitForRetry(ctx, delay); err != nil {
				return CookieValidation{}, false, contextError("cookie-import", err)
			}
			return CookieValidation{}, true, nil
		}
		return CookieValidation{}, false, &Error{Kind: KindRateLimited, Endpoint: "cookie-import", Status: response.Status, RetryAt: client.now().Add(delay), Message: "Roblox is rate limiting cookie validation.", SafeToRetry: true}
	case response.Status >= 500:
		if attempt < 3 {
			if err := waitForRetry(ctx, retryDelay(response.Header, attempt, client.now())); err != nil {
				return CookieValidation{}, false, contextError("cookie-import", err)
			}
			return CookieValidation{}, true, nil
		}
		return CookieValidation{}, false, &Error{Kind: KindServer, Endpoint: "cookie-import", Status: response.Status, Message: "Roblox is temporarily unavailable. The cookie was not stored.", SafeToRetry: true}
	default:
		return CookieValidation{}, false, &Error{Kind: KindProtocol, Endpoint: "cookie-import", Status: response.Status, Message: fmt.Sprintf("Roblox returned status %d. The cookie was not stored.", response.Status)}
	}
}

func cookieValidationIdentity(response *Response, validation CookieValidation) (CookieValidation, error) {
	var identity struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
	}
	if err := json.Unmarshal(response.Body, &identity); err != nil || identity.ID <= 0 || strings.TrimSpace(identity.Name) == "" {
		return CookieValidation{}, &Error{Kind: KindProtocol, Endpoint: "cookie-import", Status: response.Status, Message: "Roblox returned an invalid account identity."}
	}
	validation.Identity = accounts.Identity{RobloxUserID: identity.ID, Username: identity.Name, DisplayName: identity.DisplayName}
	return validation, nil
}
