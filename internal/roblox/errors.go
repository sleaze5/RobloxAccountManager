package roblox

import "log/slog"

type ErrorKind string

const (
	KindVaultLocked        ErrorKind = "vault_locked"
	KindAccountNotFound    ErrorKind = "account_not_found"
	KindInvalidCookieInput ErrorKind = "invalid_cookie_input"
	KindInvalidSession     ErrorKind = "invalid_session"
	KindReauthRequired     ErrorKind = "reauth_required"
	KindCSRFRejected       ErrorKind = "csrf_rejected"
	KindChallengeRequired  ErrorKind = "challenge_required"
	KindForbidden          ErrorKind = "forbidden"
	KindRateLimited        ErrorKind = "rate_limited"
	KindTransport          ErrorKind = "transport"
	KindTimeout            ErrorKind = "timeout"
	KindServer             ErrorKind = "server"
	KindProtocol           ErrorKind = "protocol"
	KindResponseTooLarge   ErrorKind = "response_too_large"
	KindCancelled          ErrorKind = "cancelled"
)

type Error struct {
	Kind       ErrorKind
	Endpoint   string
	Status     int
	RobloxCode int
	Message    string
	Cause      error
}

// Error returns only the message, because the desktop runtime shows it to the
// user. LogValue keeps the request details in logs.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message == "" {
		return string(e.Kind)
	}
	return e.Message
}

func (e *Error) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("message", e.Error()),
		slog.String("endpoint", e.Endpoint),
		slog.String("kind", string(e.Kind)),
		slog.Int("status", e.Status),
	)
}

func (e *Error) Unwrap() error { return e.Cause }
