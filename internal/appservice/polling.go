package appservice

import (
	"context"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

// pollUntilReady runs readOnce until it reports ready, at most maxAttempts
// times and retryDelay apart, and reports whether a read was ready. A read
// error ends the poll. Cancelling ctx between reads returns a cancellation
// error with cancelledMessage.
func pollUntilReady(ctx context.Context, operation, cancelledMessage string, maxAttempts int, retryDelay time.Duration, readOnce func() (bool, error)) (bool, error) {
	for attempt := range maxAttempts {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return false, &roblox.Error{Kind: roblox.KindCancelled, Endpoint: operation, Message: cancelledMessage, Cause: ctx.Err()}
			case <-time.After(retryDelay):
			}
		}
		isReady, err := readOnce()
		if err != nil || isReady {
			return isReady, err
		}
	}
	return false, nil
}
