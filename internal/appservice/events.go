package appservice

import (
	"sync"
	"time"
)

type Events struct {
	mutex      sync.RWMutex
	emit       func(string, any)
	showWindow func()
	rateUntil  time.Time
}

func NewEvents() *Events { return &Events{} }

func (events *Events) AttachWindow(emit func(string, any), showWindow func()) {
	events.mutex.Lock()
	events.emit, events.showWindow = emit, showWindow
	events.mutex.Unlock()
}

func (events *Events) DetachWindow() {
	events.AttachWindow(nil, nil)
}

func (events *Events) send(name string, payload any) {
	events.mutex.RLock()
	emit := events.emit
	events.mutex.RUnlock()
	if emit != nil {
		emit(name, payload)
	}
}

func (events *Events) SessionChanged(accountID int64) {
	events.send("account:session-state-changed", struct {
		AccountID int64 `json:"accountId"`
	}{AccountID: accountID})
}

func (events *Events) CredentialBackupChanged(message string) {
	events.send("vault:credential-backup-changed", struct {
		Message string `json:"message"`
	}{Message: message})
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

// LaunchConfirmationRequested also brings the window forward, because the
// launch waits until the user answers.
func (events *Events) LaunchConfirmationRequested() {
	events.LaunchConfirmationChanged()
	events.mutex.RLock()
	showWindow := events.showWindow
	events.mutex.RUnlock()
	if showWindow != nil {
		showWindow()
	}
}

func (events *Events) RateLimited(retryAt time.Time) {
	events.mutex.Lock()
	defer events.mutex.Unlock()
	if retryAt.After(events.rateUntil) {
		events.rateUntil = retryAt
	}
}

func (events *Events) BackgroundPaused(now time.Time) bool {
	events.mutex.RLock()
	defer events.mutex.RUnlock()
	return events.rateUntil.After(now)
}
