package roblox

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type requestLimitKey struct {
	host      string
	accountID int64
	endpoint  string
}

type requestLimits struct {
	global           chan struct{}
	mutex            sync.Mutex
	hosts            map[string]chan struct{}
	users            map[int64]chan struct{}
	retryAtByRequest map[requestLimitKey]time.Time
}

func newRequestLimits() *requestLimits {
	return &requestLimits{global: make(chan struct{}, 16), hosts: make(map[string]chan struct{}), users: make(map[int64]chan struct{}), retryAtByRequest: make(map[requestLimitKey]time.Time)}
}

func (limits *requestLimits) isRateLimited(key requestLimitKey) bool {
	limits.mutex.Lock()
	defer limits.mutex.Unlock()
	return limits.retryAtByRequest[key].After(time.Now())
}

func (limits *requestLimits) deferRequestsUntil(key requestLimitKey, retryAt time.Time) {
	limits.mutex.Lock()
	defer limits.mutex.Unlock()
	for previousKey, previousRetryAt := range limits.retryAtByRequest {
		if !previousRetryAt.After(time.Now()) {
			delete(limits.retryAtByRequest, previousKey)
		}
	}
	if retryAt.After(limits.retryAtByRequest[key]) {
		limits.retryAtByRequest[key] = retryAt
	}
}

func (limits *requestLimits) acquire(ctx context.Context, host string, accountID int64) (func(), error) {
	limits.mutex.Lock()
	hostLimit := limits.hosts[host]
	if hostLimit == nil {
		hostLimit = make(chan struct{}, 8)
		limits.hosts[host] = hostLimit
	}
	var accountLimit chan struct{}
	if accountID > 0 {
		accountLimit = limits.users[accountID]
		if accountLimit == nil {
			accountLimit = make(chan struct{}, 4)
			limits.users[accountID] = accountLimit
		}
	}
	limits.mutex.Unlock()
	if accountLimit != nil {
		select {
		case accountLimit <- struct{}{}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	select {
	case hostLimit <- struct{}{}:
	case <-ctx.Done():
		if accountLimit != nil {
			<-accountLimit
		}
		return nil, ctx.Err()
	}
	select {
	case limits.global <- struct{}{}:
	case <-ctx.Done():
		<-hostLimit
		if accountLimit != nil {
			<-accountLimit
		}
		return nil, ctx.Err()
	}
	return func() {
		<-limits.global
		<-hostLimit
		if accountLimit != nil {
			<-accountLimit
		}
	}, nil
}

func contextError(endpoint string, err error) error {
	if err == nil {
		return nil
	}
	kind := KindCancelled
	message := "The request was cancelled."
	if errors.Is(err, context.DeadlineExceeded) {
		kind = KindTimeout
		message = "The request timed out."
	}
	return &Error{Kind: kind, Endpoint: endpoint, Message: message, Cause: fmt.Errorf("request context: %w", err)}
}
