package roblox

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	defaultResponseLimit int64 = 2 << 20
	maximumRequestBody         = 4 << 20
)

type RetryPolicy struct {
	MaxAttempts       int
	Idempotent        bool
	IdempotencyKey    string
	RetryServerErrors bool
	RetryRateLimit    bool
}

type Request struct {
	Endpoint  string
	AccountID int64
	// ExpectedSecretVersion pins a multi-request workflow to its initiating session.
	ExpectedSecretVersion     int64
	Authenticated             bool
	Method                    string
	URL                       *url.URL
	Body                      []byte
	ContentType               string
	Headers                   http.Header
	MaxResponseSize           int64
	RequiresCSRF              bool
	AuthenticationNegotiation bool
	Referer                   string
	// FollowRedirects permits up to five same-host HTTPS redirects for GET requests.
	FollowRedirects bool
	Retry           RetryPolicy
}

// Response is internal and intentionally has no JSON tags. Headers can contain
// short-lived secrets such as an authentication ticket.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
	// RotatedSecretVersion is nonzero only after a replacement cookie was saved.
	RotatedSecretVersion int64
}

func (request Request) validate(policy func(*url.URL) bool) error {
	if request.Endpoint == "" {
		return fmt.Errorf("request endpoint name is empty")
	}
	if request.URL == nil || !request.URL.IsAbs() {
		return fmt.Errorf("%s: URL must be absolute", request.Endpoint)
	}
	if request.URL.User != nil {
		return fmt.Errorf("%s: URL user information is forbidden", request.Endpoint)
	}
	if request.Authenticated {
		if request.AccountID <= 0 {
			return fmt.Errorf("%s: account ID is required", request.Endpoint)
		}
		if !policy(request.URL) {
			return fmt.Errorf("%s: credentials are not allowed for this URL", request.Endpoint)
		}
	}
	method := strings.ToUpper(request.Method)
	if method == "" {
		return fmt.Errorf("%s: HTTP method is empty", request.Endpoint)
	}
	if request.FollowRedirects && (method != http.MethodGet || request.URL.Scheme != "https") {
		return fmt.Errorf("%s: redirects require an HTTPS GET request", request.Endpoint)
	}
	if len(request.Body) > maximumRequestBody {
		return fmt.Errorf("%s: request body is too large", request.Endpoint)
	}
	for name := range request.Headers {
		switch strings.ToLower(name) {
		case "cookie", "set-cookie", "authorization", "x-csrf-token", "x-bound-auth-token", "rbxauthenticationnegotiation":
			return fmt.Errorf("%s: protected header %s must be set by its endpoint adapter", request.Endpoint, name)
		}
	}
	return nil
}

func AllowedCredentialURL(value *url.URL) bool {
	if value == nil || !strings.EqualFold(value.Scheme, "https") || value.User != nil {
		return false
	}
	if value.Port() != "" && value.Port() != "443" {
		return false
	}
	host := strings.ToLower(strings.TrimSuffix(value.Hostname(), "."))
	return host == "roblox.com" || strings.HasSuffix(host, ".roblox.com")
}

func isRedirect(status int) bool {
	switch status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

func (request *Request) followRedirect(response *Response) error {
	location := response.Header.Get("Location")
	target, err := request.URL.Parse(location)
	if err != nil || location == "" || target.User != nil || target.Scheme != "https" || !strings.EqualFold(target.Host, request.URL.Host) {
		return &Error{Kind: KindProtocol, Endpoint: request.Endpoint, Status: response.Status, Message: "Roblox returned an unsafe or invalid redirect."}
	}
	target.Fragment = ""
	request.URL = target
	return nil
}

func requiresCSRF(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
