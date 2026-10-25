package roblox

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/appmeta"
)

const (
	requestTimeout       = 30 * time.Second
	csrfHeader           = "X-Csrf-Token"
	challengeTypeHeader  = "Rblx-Challenge-Type"
	robloxDefaultOrigin  = "https://www.roblox.com"
	robloxDefaultReferer = "https://www.roblox.com/"
)

var errResponseTooLarge = errors.New("response exceeds configured limit")

type SessionStore interface {
	RotateCookie(context.Context, int64, int64, string, *time.Time) (accounts.SessionRecord, bool, error)
	MarkState(context.Context, int64, int64, accounts.SessionState, string) error
	RecordSuccess(context.Context, int64, int64) error
}

type Observer interface {
	SessionChanged(accountID int64)
	RateLimited(accountID int64, retryAt time.Time)
}

type ClientOptions struct {
	Transport            http.RoundTripper
	Store                SessionStore
	Sessions             *SessionManager
	Observer             Observer
	CredentialPolicy     func(*url.URL) bool
	AuthenticatedUserURL *url.URL
	Now                  func() time.Time
	Logger               *slog.Logger
}

type Client struct {
	http                 *http.Client
	store                SessionStore
	sessions             *SessionManager
	observer             Observer
	credentialPolicy     func(*url.URL) bool
	authenticatedUserURL *url.URL
	now                  func() time.Time
	limits               *requestLimits
	logger               *slog.Logger
}

func NewClient(options ClientOptions) *Client {
	transport := options.Transport
	if transport == nil {
		transport = &http.Transport{
			Proxy:                  http.ProxyFromEnvironment,
			DialContext:            (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			ForceAttemptHTTP2:      true,
			TLSClientConfig:        &tls.Config{MinVersion: tls.VersionTLS12},
			MaxIdleConns:           100,
			MaxIdleConnsPerHost:    10,
			MaxResponseHeaderBytes: 1 << 20,
			IdleConnTimeout:        90 * time.Second,
			TLSHandshakeTimeout:    10 * time.Second,
			ResponseHeaderTimeout:  15 * time.Second,
			ExpectContinueTimeout:  time.Second,
		}
	}
	policy := options.CredentialPolicy
	if policy == nil {
		policy = AllowedCredentialURL
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	authenticatedUserURL := options.AuthenticatedUserURL
	if authenticatedUserURL == nil {
		authenticatedUserURL, _ = url.Parse("https://users.roblox.com/v1/users/authenticated")
	}
	logger := options.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Client{
		http: &http.Client{
			Transport: transport,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		store:                options.Store,
		sessions:             options.Sessions,
		observer:             options.Observer,
		credentialPolicy:     policy,
		authenticatedUserURL: cloneURL(authenticatedUserURL),
		now:                  now,
		limits:               newRequestLimits(),
		logger:               logger,
	}
}

type requestExecution struct {
	request       Request
	maxAttempts   int
	serverAttempt int
	redirects     int
	csrfRetried   bool
}

func (client *Client) Do(ctx context.Context, request Request) (*Response, error) {
	if err := client.prepareRequest(&request); err != nil {
		return nil, err
	}
	maxAttempts := min(max(request.Retry.MaxAttempts, 1), 3)
	if !safeToRetryRequest(request) {
		maxAttempts = 1
	}
	execution := requestExecution{request: request, maxAttempts: maxAttempts}
	logicalContext, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	for {
		if logicalContext.Err() != nil {
			return nil, contextError(execution.request.Endpoint, logicalContext.Err())
		}
		execution.serverAttempt++
		response, retry, err := client.executeRequestAttempt(logicalContext, &execution)
		if retry {
			continue
		}
		return response, err
	}
}

func (client *Client) prepareRequest(request *Request) error {
	if client.sessions == nil {
		return &Error{Kind: KindProtocol, Endpoint: request.Endpoint, Message: "The session manager is unavailable."}
	}
	if err := request.validate(client.credentialPolicy); err != nil {
		return &Error{Kind: KindProtocol, Endpoint: request.Endpoint, Message: "The request contract is invalid.", Cause: err}
	}
	request.Method = strings.ToUpper(request.Method)
	request.Body = append([]byte(nil), request.Body...)
	request.URL = cloneURL(request.URL)
	request.Headers = request.Headers.Clone()
	if request.Headers == nil {
		request.Headers = make(http.Header)
	}
	if request.MaxResponseSize <= 0 {
		request.MaxResponseSize = defaultResponseLimit
	}
	if request.MaxResponseSize > 16<<20 {
		return &Error{Kind: KindProtocol, Endpoint: request.Endpoint, Message: "The response limit is too large."}
	}
	if request.RequiresCSRF && !requiresCSRF(request.Method) {
		return &Error{Kind: KindProtocol, Endpoint: request.Endpoint, Message: "CSRF was requested for a safe HTTP method."}
	}
	return nil
}

func (client *Client) requestSnapshot(ctx context.Context, request Request) (SessionSnapshot, error) {
	var snapshot SessionSnapshot
	if request.Authenticated {
		var err error
		snapshot, err = client.sessions.Snapshot(request.AccountID)
		if err != nil {
			return SessionSnapshot{}, err
		}
		if request.ExpectedSecretVersion != 0 && snapshot.SecretVersion != request.ExpectedSecretVersion {
			return SessionSnapshot{}, &Error{Kind: KindCancelled, Endpoint: request.Endpoint, Message: "The account session changed. Refresh settings to continue."}
		}
		if snapshot.State == accounts.StateDisabled {
			return SessionSnapshot{}, &Error{Kind: KindForbidden, Endpoint: request.Endpoint, Message: "Network use is disabled for this account."}
		}
		if snapshot.CookieExpiresAt != nil && !snapshot.CookieExpiresAt.After(client.now()) {
			client.setState(ctx, snapshot, accounts.StateReauthRequired, "The known cookie expiry passed.")
			return SessionSnapshot{}, &Error{Kind: KindReauthRequired, Endpoint: request.Endpoint, Message: "The Roblox session has expired."}
		}
	}

	return snapshot, nil
}

func (client *Client) executeRequestAttempt(ctx context.Context, execution *requestExecution) (*Response, bool, error) {
	request := &execution.request
	snapshot, err := client.requestSnapshot(ctx, *request)
	if err != nil {
		return nil, false, err
	}
	if request.Authenticated && !client.sessions.VersionCurrent(snapshot.AccountID, snapshot.SecretVersion) {
		execution.serverAttempt--
		return nil, true, nil
	}
	response, sentCSRFToken, sendErr := client.sendSessionAttempt(ctx, *request, snapshot)
	if response != nil && request.Authenticated {
		if err := client.rotateResponseCookie(ctx, request, &snapshot, response); err != nil {
			return nil, false, err
		}
	}
	if sendErr != nil {
		return client.handleSendError(ctx, execution, snapshot, response, sendErr)
	}
	if request.Authenticated {
		if !client.sessions.VersionCurrent(snapshot.AccountID, snapshot.SecretVersion) {
			return client.handleStaleResponse(execution, response)
		}
	}
	return client.handleAttemptResponse(ctx, execution, snapshot, response, sentCSRFToken)
}

func (client *Client) sendSessionAttempt(ctx context.Context, request Request, snapshot SessionSnapshot) (*Response, string, error) {
	attemptContext := ctx
	stopMerge := func() {}
	if request.Authenticated {
		attemptContext, stopMerge = mergeContext(ctx, snapshot.Context)
	}
	sentCSRFToken := ""
	if request.Authenticated && request.RequiresCSRF {
		sentCSRFToken = client.sessions.CSRF(snapshot.AccountID, snapshot.SecretVersion)
	}
	response, sendErr := client.sendAttempt(attemptContext, request, snapshot, sentCSRFToken)
	stopMerge()
	return response, sentCSRFToken, sendErr
}

func (client *Client) rotateResponseCookie(ctx context.Context, request *Request, snapshot *SessionSnapshot, response *Response) error {
	// A server-issued replacement must be saved even if reading the body
	// failed or the caller cancelled after the headers arrived.
	rotationContext, finishRotation := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	version, deleted, rotationErr := client.processSetCookie(rotationContext, *request, *snapshot, response.Header)
	finishRotation()
	if rotationErr != nil {
		return rotationErr
	}
	if deleted {
		return &Error{Kind: KindReauthRequired, Endpoint: request.Endpoint, Status: response.Status, Message: "Roblox ended this session."}
	}
	response.RotatedSecretVersion = version
	if version != 0 {
		latest, snapshotErr := client.sessions.Snapshot(request.AccountID)
		if snapshotErr != nil {
			return snapshotErr
		}
		if latest.SecretVersion != version {
			return &Error{Kind: KindCancelled, Endpoint: request.Endpoint, Message: "The account session changed while the replacement was being saved."}
		}
		*snapshot = latest
		if request.ExpectedSecretVersion != 0 {
			request.ExpectedSecretVersion = version
		}
	}
	return nil
}

func (client *Client) handleSendError(ctx context.Context, execution *requestExecution, snapshot SessionSnapshot, response *Response, sendErr error) (*Response, bool, error) {
	request := execution.request
	if typed, ok := errors.AsType[*Error](sendErr); ok {
		return response, false, typed
	}
	if errors.Is(sendErr, accounts.ErrStaleSecret) {
		execution.serverAttempt--
		return nil, true, nil
	}
	if ctx.Err() != nil {
		return response, false, contextError(request.Endpoint, ctx.Err())
	}
	if request.Authenticated && !client.sessions.VersionCurrent(snapshot.AccountID, snapshot.SecretVersion) {
		if safeToRetryRequest(request) {
			execution.serverAttempt--
			return nil, true, nil
		}
		return nil, false, &Error{Kind: KindCancelled, Endpoint: request.Endpoint, Message: "The account session changed while the request was in progress."}
	}
	if execution.serverAttempt < execution.maxAttempts && retryableTransport(sendErr) {
		if err := waitForRetry(ctx, backoff(execution.serverAttempt, 0)); err != nil {
			return nil, false, contextError(request.Endpoint, err)
		}
		return nil, true, nil
	}
	kind := KindTransport
	var netErr net.Error
	if errors.As(sendErr, &netErr) && netErr.Timeout() {
		kind = KindTimeout
	}
	return response, false, &Error{Kind: kind, Endpoint: request.Endpoint, Message: "The Roblox request failed.", Cause: sendErr, SafeToRetry: safeToRetryRequest(request)}
}

func (client *Client) handleStaleResponse(execution *requestExecution, response *Response) (*Response, bool, error) {
	request := execution.request
	if _, snapshotErr := client.sessions.Snapshot(request.AccountID); snapshotErr != nil {
		return nil, false, snapshotErr
	}
	if response.Status >= 200 && response.Status < 300 {
		return response, false, nil
	}
	if safeToRetryRequest(request) {
		execution.serverAttempt--
		return nil, true, nil
	}
	return nil, false, &Error{Kind: KindCancelled, Endpoint: request.Endpoint, Status: response.Status, Message: "The account session changed while the request was in progress."}
}

func (client *Client) handleAttemptResponse(ctx context.Context, execution *requestExecution, snapshot SessionSnapshot, response *Response, sentCSRFToken string) (*Response, bool, error) {
	request := execution.request
	if request.FollowRedirects && isRedirect(response.Status) {
		return execution.followRedirect(response)
	}
	if client.retryCSRF(execution, snapshot, response, sentCSRFToken) {
		return nil, true, nil
	}
	if response.Status >= 200 && response.Status < 300 {
		if request.Authenticated {
			client.recordSuccess(ctx, snapshot)
		}
		return response, false, nil
	}
	return client.handleFailedResponse(ctx, execution, snapshot, response)
}

func (execution *requestExecution) followRedirect(response *Response) (*Response, bool, error) {
	request := &execution.request
	if execution.redirects >= 5 {
		return nil, false, &Error{Kind: KindProtocol, Endpoint: request.Endpoint, Status: response.Status, Message: "Roblox returned too many redirects."}
	}
	if err := request.followRedirect(response); err != nil {
		return nil, false, err
	}
	execution.redirects++
	execution.serverAttempt = 0
	return nil, true, nil
}

func (client *Client) retryCSRF(execution *requestExecution, snapshot SessionSnapshot, response *Response, sentCSRFToken string) bool {
	request := execution.request
	if response.Status == http.StatusForbidden && request.RequiresCSRF && response.Header.Get(challengeTypeHeader) == "" {
		newToken := strings.TrimSpace(response.Header.Get(csrfHeader))
		if newToken != "" {
			client.sessions.SetCSRF(snapshot.AccountID, snapshot.SecretVersion, newToken)
			if !execution.csrfRetried && (sentCSRFToken == "" || sentCSRFToken != newToken) {
				execution.csrfRetried = true
				execution.serverAttempt--
				return true
			}
		}
	}
	return false
}

func (client *Client) handleFailedResponse(ctx context.Context, execution *requestExecution, snapshot SessionSnapshot, response *Response) (*Response, bool, error) {
	request := execution.request
	if response.Status == http.StatusUnauthorized && request.Authenticated {
		return nil, false, client.unauthorizedResponse(ctx, request, snapshot, response)
	}
	if response.Status == http.StatusForbidden {
		return nil, false, client.forbiddenResponse(ctx, request, snapshot, response)
	}
	if response.Status == http.StatusTooManyRequests {
		return client.rateLimitedResponse(ctx, execution, response)
	}
	if isRetryableStatus(response.Status) && request.Retry.RetryServerErrors && execution.serverAttempt < execution.maxAttempts {
		if err := waitForRetry(ctx, retryDelay(response.Header, execution.serverAttempt, client.now())); err != nil {
			return nil, false, contextError(request.Endpoint, err)
		}
		return nil, true, nil
	}
	return nil, false, responseError(request, response)
}

func (client *Client) unauthorizedResponse(ctx context.Context, request Request, snapshot SessionSnapshot, response *Response) error {
	if request.Endpoint == "authenticated-user" {
		client.setState(ctx, snapshot, accounts.StateReauthRequired, "Roblox rejected the session.")
		return &Error{Kind: KindReauthRequired, Endpoint: request.Endpoint, Status: response.Status, Message: "Roblox rejected this session."}
	}
	invalid := client.confirmSessionInvalid(ctx, snapshot)
	if invalid {
		return &Error{Kind: KindReauthRequired, Endpoint: request.Endpoint, Status: response.Status, Message: "Roblox rejected this session."}
	}
	return &Error{Kind: KindForbidden, Endpoint: request.Endpoint, Status: response.Status, Message: "Roblox rejected this operation."}
}

func (client *Client) forbiddenResponse(ctx context.Context, request Request, snapshot SessionSnapshot, response *Response) error {
	challengeType := strings.TrimSpace(response.Header.Get(challengeTypeHeader))
	if challengeType != "" {
		message := "Roblox requires user interaction."
		client.setState(ctx, snapshot, accounts.StateChallenged, message)
		return &Error{Kind: KindChallengeRequired, Endpoint: request.Endpoint, Status: response.Status, ChallengeType: challengeType, Message: message}
	}
	kind := KindForbidden
	message := "Roblox rejected this operation."
	if request.RequiresCSRF && strings.TrimSpace(response.Header.Get(csrfHeader)) != "" {
		kind = KindCSRFRejected
		message = "Roblox rejected the CSRF token."
	}
	return &Error{Kind: kind, Endpoint: request.Endpoint, Status: response.Status, RobloxCode: responseError(request, response).RobloxCode, Message: message}
}

func (client *Client) rateLimitedResponse(ctx context.Context, execution *requestExecution, response *Response) (*Response, bool, error) {
	request := execution.request
	delay := retryDelay(response.Header, execution.serverAttempt, client.now())
	retryAt := client.now().Add(delay)
	if client.observer != nil {
		client.observer.RateLimited(request.AccountID, retryAt)
	}
	if request.Retry.RetryRateLimit && execution.serverAttempt < execution.maxAttempts {
		if err := waitForRetry(ctx, delay); err != nil {
			return nil, false, contextError(request.Endpoint, err)
		}
		return nil, true, nil
	}
	return nil, false, &Error{Kind: KindRateLimited, Endpoint: request.Endpoint, Status: response.Status, RetryAt: retryAt, SafeToRetry: true, Message: "Roblox is rate limiting requests."}
}

func (client *Client) sendAttempt(ctx context.Context, contract Request, snapshot SessionSnapshot, csrfToken string) (*Response, error) {
	if contract.Authenticated && !client.sessions.VersionCurrent(snapshot.AccountID, snapshot.SecretVersion) {
		return nil, accounts.ErrStaleSecret
	}
	request, err := http.NewRequestWithContext(ctx, contract.Method, contract.URL.String(), bytes.NewReader(contract.Body))
	if err != nil {
		return nil, fmt.Errorf("create Roblox request: %w", err)
	}
	if err := client.prepareAttemptHeaders(request, contract, snapshot, csrfToken); err != nil {
		return nil, err
	}
	release, err := client.limits.acquire(ctx, request.URL.Hostname(), contract.AccountID)
	if err != nil {
		return nil, err
	}
	defer release()
	started := client.now()
	// slog.LevelDebug-4 is the trace level.
	client.logger.Log(ctx, slog.LevelDebug-4, "Roblox request started",
		"endpoint", contract.Endpoint,
		"account_id", contract.AccountID,
		"method", contract.Method,
		"host", request.URL.Hostname(),
	)
	response, err := client.http.Do(request)
	if err != nil {
		// net/http wraps transport errors with the full URL, including invite codes.
		if urlError, ok := errors.AsType[*url.Error](err); ok {
			err = urlError.Err
		}
		client.logger.Debug("Roblox request transport failed",
			"endpoint", contract.Endpoint,
			"account_id", contract.AccountID,
			"host", request.URL.Hostname(),
			"duration_ms", client.now().Sub(started).Milliseconds(),
			"error", err,
		)
		return nil, err
	}
	defer response.Body.Close()
	result := &Response{Status: response.StatusCode, Header: response.Header.Clone()}
	body, err := readLimited(response.Body, contract.MaxResponseSize)
	if err != nil {
		if errors.Is(err, errResponseTooLarge) {
			return result, &Error{Kind: KindResponseTooLarge, Endpoint: contract.Endpoint, Status: response.StatusCode, Message: "The Roblox response was too large.", Cause: err}
		}
		return result, err
	}
	// slog.LevelDebug-4 is the trace level.
	client.logger.Log(ctx, slog.LevelDebug-4, "Roblox response received",
		"endpoint", contract.Endpoint,
		"account_id", contract.AccountID,
		"status", response.StatusCode,
		"duration_ms", client.now().Sub(started).Milliseconds(),
	)
	result.Body = body
	return result, nil
}

func (client *Client) processSetCookie(ctx context.Context, request Request, snapshot SessionSnapshot, headers http.Header) (int64, bool, error) {
	dummy := &http.Response{Header: headers}
	for _, cookie := range dummy.Cookies() {
		if cookie.Name != accounts.RoblosecurityCookieName {
			continue
		}
		if cookie.Value == "" || cookie.MaxAge < 0 {
			client.setState(ctx, snapshot, accounts.StateReauthRequired, "Roblox deleted the session cookie.")
			return 0, true, nil
		}
		if err := accounts.ValidateCookieValue(cookie.Value); err != nil {
			return 0, false, &Error{Kind: KindProtocol, Endpoint: request.Endpoint, Message: "Roblox returned an invalid replacement cookie.", Cause: err}
		}
		var expires *time.Time
		if !cookie.Expires.IsZero() {
			value := cookie.Expires.UTC()
			expires = &value
			if !value.After(client.now()) {
				client.setState(ctx, snapshot, accounts.StateReauthRequired, "Roblox expired the session cookie.")
				return 0, true, nil
			}
		}
		if cookie.Value == snapshot.Cookie && sameTime(expires, snapshot.CookieExpiresAt) {
			return 0, false, nil
		}
		return client.storeRotatedCookie(ctx, request, snapshot, cookie.Value, expires)
	}
	return 0, false, nil
}

func (client *Client) confirmSessionInvalid(ctx context.Context, snapshot SessionSnapshot) bool {
	value, err := client.sessions.Validate(ctx, snapshot.AccountID, func(probeContext context.Context, current SessionSnapshot) (any, error) {
		status, err := client.probeAuthenticated(probeContext, current)
		if err != nil {
			return false, err
		}
		return status == http.StatusUnauthorized, nil
	})
	if err != nil {
		return false
	}
	invalid, _ := value.(bool)
	if invalid {
		current, err := client.sessions.Snapshot(snapshot.AccountID)
		if err == nil {
			client.setState(ctx, current, accounts.StateReauthRequired, "Roblox rejected the session.")
		}
	}
	return invalid
}

func (client *Client) probeAuthenticated(ctx context.Context, snapshot SessionSnapshot) (int, error) {
	probeURL := cloneURL(client.authenticatedUserURL)
	if !client.credentialPolicy(probeURL) {
		return 0, errors.New("authenticated-user URL is outside the credential policy")
	}
	merged, stop := mergeContext(ctx, snapshot.Context)
	defer stop()
	request, err := http.NewRequestWithContext(merged, http.MethodGet, probeURL.String(), http.NoBody)
	if err != nil {
		return 0, fmt.Errorf("create authenticated-user request: %w", err)
	}
	request.Header.Set("User-Agent", appmeta.UserAgent)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Cookie", formatAuthCookie(snapshot.Cookie, snapshot.BrowserID))
	release, err := client.limits.acquire(merged, request.URL.Hostname(), snapshot.AccountID)
	if err != nil {
		return 0, err
	}
	defer release()
	response, err := client.http.Do(request)
	if err != nil {
		return 0, fmt.Errorf("send authenticated-user request: %w", err)
	}
	defer response.Body.Close()
	if _, err := io.Copy(io.Discard, io.LimitReader(response.Body, defaultResponseLimit)); err != nil {
		return 0, fmt.Errorf("read authenticated-user response: %w", err)
	}
	rotated, deleted, err := client.processSetCookie(ctx, Request{Endpoint: "authenticated-user-probe", AccountID: snapshot.AccountID, Authenticated: true}, snapshot, response.Header.Clone())
	if err != nil {
		return 0, err
	}
	if rotated != 0 || deleted {
		return 0, accounts.ErrStaleSecret
	}
	return response.StatusCode, nil
}

func (client *Client) setState(ctx context.Context, snapshot SessionSnapshot, state accounts.SessionState, reason string) {
	if client.store == nil {
		return
	}
	if err := client.store.MarkState(ctx, snapshot.AccountID, snapshot.SecretVersion, state, reason); err != nil {
		return
	}
	client.sessions.UpdateState(snapshot.AccountID, snapshot.SecretVersion, state)
	client.logger.Warn("Roblox session state changed", "account_id", snapshot.AccountID, "state", state)
	if client.observer != nil {
		client.observer.SessionChanged(snapshot.AccountID)
	}
}

func (client *Client) recordSuccess(ctx context.Context, snapshot SessionSnapshot) {
	if client.store == nil {
		return
	}
	if err := client.store.RecordSuccess(ctx, snapshot.AccountID, snapshot.SecretVersion); err != nil {
		return
	}
	client.sessions.UpdateState(snapshot.AccountID, snapshot.SecretVersion, accounts.StateActive)
}

func safeToRetryRequest(request Request) bool {
	switch request.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return request.Retry.Idempotent || request.Retry.IdempotencyKey != ""
	}
}

func retryableTransport(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}

func isRetryableStatus(status int) bool {
	switch status {
	case http.StatusRequestTimeout, http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func retryDelay(headers http.Header, attempt int, now time.Time) time.Duration {
	if raw := strings.TrimSpace(headers.Get("Retry-After")); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds >= 0 {
			if seconds >= 30 {
				return 30 * time.Second
			}
			return time.Duration(seconds) * time.Second
		}
		if when, err := http.ParseTime(raw); err == nil && when.After(now) {
			return minDuration(when.Sub(now), 30*time.Second)
		}
	}
	return backoff(attempt, secureSample())
}

func backoff(attempt, sample int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	base := time.Duration(math.Pow(2, float64(attempt-1))) * 250 * time.Millisecond
	jitter := time.Duration(sample%251) * time.Millisecond
	return minDuration(base+jitter, 5*time.Second)
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read Roblox response: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w: %d bytes", errResponseTooLarge, limit)
	}
	return data, nil
}

func responseError(request Request, response *Response) *Error {
	kind := KindProtocol
	message := "Roblox returned an unexpected response."
	if response.Status >= 500 {
		kind = KindServer
		message = "Roblox is temporarily unavailable."
	}
	robloxCode := 0
	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Error   string `json:"Error"`
		Errors  []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if json.Unmarshal(response.Body, &body) == nil {
		robloxCode = body.Code
		switch {
		case len(body.Errors) > 0:
			robloxCode = body.Errors[0].Code
			if body.Errors[0].Message != "" {
				message = body.Errors[0].Message
			}
		case body.Message != "":
			message = body.Message
		case body.Error != "":
			message = body.Error
		}
	} else {
		var singleString string
		if json.Unmarshal(response.Body, &singleString) == nil && singleString != "" {
			message = singleString
		}
	}
	return &Error{Kind: kind, Endpoint: request.Endpoint, Status: response.Status, RobloxCode: robloxCode, Message: message, SafeToRetry: safeToRetryRequest(request) && isRetryableStatus(response.Status)}
}

func mergeContext(primary, cancellation context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancel(primary)
	stop := context.AfterFunc(cancellation, cancel)
	return ctx, func() {
		stop()
		cancel()
	}
}

func cloneURL(value *url.URL) *url.URL {
	if value == nil {
		return nil
	}
	result := *value
	if value.User != nil {
		user := *value.User
		result.User = &user
	}
	return &result
}

func secureSample() int {
	var value [8]byte
	if _, err := rand.Read(value[:]); err != nil {
		return int(time.Now().UnixNano() & math.MaxInt32)
	}
	return int(binary.LittleEndian.Uint64(value[:]) & math.MaxInt32)
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func sameTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func (client *Client) prepareAttemptHeaders(request *http.Request, contract Request, snapshot SessionSnapshot, csrfToken string) error {
	request.Header = contract.Headers.Clone()
	request.Header.Set("User-Agent", appmeta.UserAgent)
	if request.Header.Get("Accept") == "" {
		request.Header.Set("Accept", "application/json")
	}
	if contract.ContentType != "" {
		request.Header.Set("Content-Type", contract.ContentType)
	}
	if contract.Referer != "" {
		request.Header.Set("Referer", contract.Referer)
	}
	if contract.AuthenticationNegotiation {
		request.Header.Set("Rbxauthenticationnegotiation", "1")
	}
	if contract.Retry.IdempotencyKey != "" {
		request.Header.Set("Idempotency-Key", contract.Retry.IdempotencyKey)
	}
	if contract.Authenticated {
		if !client.credentialPolicy(request.URL) {
			return errors.New("credential host failed final validation")
		}
		request.Header.Set("Cookie", formatAuthCookie(snapshot.Cookie, snapshot.BrowserID))
		if contract.RequiresCSRF {
			if request.Header.Get("Origin") == "" {
				request.Header.Set("Origin", robloxDefaultOrigin)
			}
			if request.Header.Get("Referer") == "" {
				request.Header.Set("Referer", robloxDefaultReferer)
			}
			if csrfToken != "" {
				request.Header.Set(csrfHeader, csrfToken)
			}
		}
	}
	return nil
}

func (client *Client) storeRotatedCookie(ctx context.Context, request Request, snapshot SessionSnapshot, value string, expires *time.Time) (int64, bool, error) {
	if client.store == nil {
		return 0, false, &Error{Kind: KindProtocol, Endpoint: request.Endpoint, Message: "Session rotation storage is unavailable."}
	}
	record, changed, err := client.store.RotateCookie(ctx, snapshot.AccountID, snapshot.SecretVersion, value, expires)
	if err != nil {
		if errors.Is(err, accounts.ErrStaleSecret) {
			return 0, false, &Error{Kind: KindCancelled, Endpoint: request.Endpoint, Message: "The saved session changed. The newer cookie was not overwritten.", Cause: err}
		}
		return 0, false, &Error{Kind: KindProtocol, Endpoint: request.Endpoint, Message: "The rotated Roblox session could not be stored. Sign in again to restore access.", Cause: err}
	}
	if changed {
		if err := client.sessions.ApplyRotation(record); err != nil {
			return 0, false, err
		}
		if client.observer != nil {
			client.observer.SessionChanged(snapshot.AccountID)
		}
		client.logger.Info("Roblox session cookie rotated", "account_id", snapshot.AccountID)
		return record.SecretVersion, false, nil
	}
	return 0, false, nil
}
