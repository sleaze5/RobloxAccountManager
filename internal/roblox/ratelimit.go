package roblox

import (
	"context"
	"fmt"
	"sync"
)

type requestLimits struct {
	global chan struct{}
	mu     sync.Mutex
	hosts  map[string]chan struct{}
	users  map[int64]chan struct{}
}

func newRequestLimits() *requestLimits {
	return &requestLimits{global: make(chan struct{}, 16), hosts: make(map[string]chan struct{}), users: make(map[int64]chan struct{})}
}

func (limits *requestLimits) acquire(ctx context.Context, host string, accountID int64) (func(), error) {
	limits.mu.Lock()
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
	limits.mu.Unlock()
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
	if err == context.DeadlineExceeded {
		kind = KindTimeout
		message = "The request timed out."
	}
	return &Error{Kind: kind, Endpoint: endpoint, Message: message, Cause: fmt.Errorf("request context: %w", err)}
}
