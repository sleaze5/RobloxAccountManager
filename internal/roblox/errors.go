package roblox

import (
	"fmt"
	"time"
)

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

// Error carries protocol state without including request credentials or raw
// response bodies.
type Error struct {
	Kind          ErrorKind
	Endpoint      string
	Status        int
	RobloxCode    int
	ChallengeType string
	RetryAt       time.Time
	SafeToRetry   bool
	Message       string
	Cause         error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	message := e.Message
	if message == "" {
		message = string(e.Kind)
	}
	if e.Endpoint != "" {
		return fmt.Sprintf("%s: %s", e.Endpoint, message)
	}
	return message
}

func (e *Error) Unwrap() error { return e.Cause }
