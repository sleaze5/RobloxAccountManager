package roblox

import "log/slog"

type ErrorKind string

const (
	KindVaultLocked        ErrorKind = "vault_locked"
	KindAccountNotFound    ErrorKind = "account_not_found"
	KindInvalidCookieInput ErrorKind = "invalid_cookie_input"
	KindInvalidSession     ErrorKind = "invalid_session"
	KindReauthRequired     ErrorKind = "reauth_required"
	KindModerated          ErrorKind = "moderated"
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
func (failure *Error) Error() string {
	if failure == nil {
		return ""
	}
	if failure.Message == "" {
		return string(failure.Kind)
	}
	return failure.Message
}

func (failure *Error) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("message", failure.Error()),
		slog.String("endpoint", failure.Endpoint),
		slog.String("kind", string(failure.Kind)),
		slog.Int("status", failure.Status),
		slog.Int("roblox_code", failure.RobloxCode),
	)
}

func (failure *Error) Unwrap() error { return failure.Cause }
