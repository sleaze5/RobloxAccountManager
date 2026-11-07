package roblox

import (
	"context"
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

func (client *Client) ValidateCookie(ctx context.Context, cookie, browserID string) (CookieValidation, error) {
	normalized, err := accounts.NormalizeCookie(cookie)
	if err != nil {
		return CookieValidation{}, &Error{Kind: KindInvalidCookieInput, Endpoint: "cookie-import", Message: err.Error()}
	}
	cookie = normalized
	if browserID == "" {
		browserID = NewBrowserID()
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
		if err := applyValidationCookies(response, &validation); err != nil {
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
	request, err := newAuthenticatedUserRequest(ctx, validation.Cookie, validation.BrowserID)
	if err != nil {
		return nil, false, &Error{Kind: KindProtocol, Endpoint: "cookie-import", Message: "The cookie validation request is invalid.", Cause: err}
	}
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
		return nil, false, &Error{Kind: KindTransport, Endpoint: "cookie-import", Message: "Roblox could not be reached. The cookie was not stored.", Cause: err}
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
		return nil, false, &Error{Kind: KindTransport, Endpoint: "cookie-import", Status: response.StatusCode, Message: "The cookie validation response could not be read. The cookie was not stored.", Cause: readErr}
	}
	return &Response{Status: response.StatusCode, Header: response.Header, Body: body}, false, nil
}

func applyValidationCookies(response *Response, validation *CookieValidation) error {
	replacement, found, err := sessionCookieReplacement(response.Header)
	switch {
	case !found:
		return nil
	case err != nil:
		return &Error{Kind: KindProtocol, Endpoint: "cookie-import", Message: "Roblox returned an invalid replacement cookie.", Cause: err}
	case replacement.deleted:
		return &Error{Kind: KindInvalidSession, Endpoint: "cookie-import", Status: response.Status, Message: "Roblox rejected this cookie."}
	case replacement.expired:
		return &Error{Kind: KindInvalidSession, Endpoint: "cookie-import", Status: response.Status, Message: "Roblox returned an expired session cookie."}
	}
	validation.Cookie = replacement.value
	if replacement.expires != nil {
		validation.ExpiresAt = replacement.expires
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
			return CookieValidation{}, false, &Error{Kind: KindChallengeRequired, Endpoint: "cookie-import", Status: response.Status, Message: message}
		}
		return CookieValidation{}, false, &Error{Kind: KindForbidden, Endpoint: "cookie-import", Status: response.Status, Message: "Roblox rejected cookie validation."}
	case response.Status == http.StatusTooManyRequests:
		if attempt < 3 {
			if err := waitForRetry(ctx, retryDelay(response.Header, attempt)); err != nil {
				return CookieValidation{}, false, contextError("cookie-import", err)
			}
			return CookieValidation{}, true, nil
		}
		return CookieValidation{}, false, &Error{Kind: KindRateLimited, Endpoint: "cookie-import", Status: response.Status, Message: "Roblox is rate limiting cookie validation."}
	case response.Status >= 500:
		if attempt < 3 {
			if err := waitForRetry(ctx, retryDelay(response.Header, attempt)); err != nil {
				return CookieValidation{}, false, contextError("cookie-import", err)
			}
			return CookieValidation{}, true, nil
		}
		return CookieValidation{}, false, &Error{Kind: KindServer, Endpoint: "cookie-import", Status: response.Status, Message: "Roblox is temporarily unavailable. The cookie was not stored."}
	default:
		return CookieValidation{}, false, &Error{Kind: KindProtocol, Endpoint: "cookie-import", Status: response.Status, Message: fmt.Sprintf("Roblox returned status %d. The cookie was not stored.", response.Status)}
	}
}

func cookieValidationIdentity(response *Response, validation CookieValidation) (CookieValidation, error) {
	identity, ok := DecodeAuthenticatedUser(response.Body)
	if !ok {
		return CookieValidation{}, &Error{Kind: KindProtocol, Endpoint: "cookie-import", Status: response.Status, Message: "Roblox returned an invalid account identity."}
	}
	validation.Identity = identity
	return validation, nil
}
