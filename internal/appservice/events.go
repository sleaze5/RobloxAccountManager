package appservice

import (
	"sync"
	"time"
)

type eventEmitter func(string, any)

type Events struct {
	mu        sync.RWMutex
	emit      eventEmitter
	rateUntil time.Time
}

func NewEvents() *Events { return &Events{} }

func (events *Events) SetEmitter(emitter func(string, any)) {
	events.mu.Lock()
	events.emit = emitter
	events.mu.Unlock()
}

func (events *Events) send(name string, data any) {
	events.mu.RLock()
	emitter := events.emit
	events.mu.RUnlock()
	if emitter != nil {
		emitter(name, data)
	}
}

func (events *Events) SessionChanged(accountID int64) {
	events.send("account:session-state-changed", struct {
		AccountID int64 `json:"accountId"`
	}{AccountID: accountID})
}

func (events *Events) BrowserChanged(revision uint64) {
	events.send("browser:state-changed", struct {
		Revision uint64 `json:"revision"`
	}{Revision: revision})
}

func (events *Events) BrowserLaunchFailed(sessionID, message string) {
	events.send("browser:launch-failed", struct {
		SessionID string `json:"sessionId"`
		Message   string `json:"message"`
	}{SessionID: sessionID, Message: message})
}

func (events *Events) UpdateChanged() {
	events.send("update:state-changed", struct{}{})
}

func (events *Events) LaunchConfirmationChanged() {
	events.send("game:launch-confirmation-changed", struct{}{})
}

func (events *Events) RateLimited(accountID int64, retryAt time.Time) {
	events.mu.Lock()
	if retryAt.After(events.rateUntil) {
		events.rateUntil = retryAt
	}
	emitter := events.emit
	events.mu.Unlock()
	if emitter != nil {
		emitter("account:rate-limited", struct {
			AccountID int64 `json:"accountId"`
			RetryAtMS int64 `json:"retryAtMs"`
		}{AccountID: accountID, RetryAtMS: retryAt.UnixMilli()})
	}
}

func (events *Events) BackgroundPaused(now time.Time) bool {
	events.mu.RLock()
	defer events.mu.RUnlock()
	return events.rateUntil.After(now)
}
