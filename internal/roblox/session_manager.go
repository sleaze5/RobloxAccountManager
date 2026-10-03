package roblox

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"golang.org/x/sync/singleflight"
)

type SessionSnapshot struct {
	AccountID       int64
	SecretVersion   int64
	Cookie          string
	BrowserID       string
	CookieExpiresAt *time.Time
	State           accounts.SessionState
	Context         context.Context
}

type runtimeSession struct {
	mu              sync.RWMutex
	accountID       int64
	secretVersion   int64
	state           accounts.SessionState
	csrfToken       string
	browserID       string
	context         context.Context
	cancel          context.CancelFunc
	validationGroup singleflight.Group
}

type credentialLoader func(context.Context, int64) (accounts.SessionRecord, error)

type SessionManager struct {
	mu             sync.RWMutex
	sessions       map[int64]*runtimeSession
	loader         credentialLoader
	locked         bool
	browserIDs     map[int64]string
	usedBrowserIDs map[string]struct{}
}

func NewSessionManager() *SessionManager {
	return &SessionManager{sessions: make(map[int64]*runtimeSession), locked: true, browserIDs: make(map[int64]string), usedBrowserIDs: make(map[string]struct{})}
}

func (manager *SessionManager) SetLoader(loader func(context.Context, int64) (accounts.SessionRecord, error)) {
	manager.mu.Lock()
	manager.loader = loader
	manager.mu.Unlock()
}

func (manager *SessionManager) Unlock() {
	manager.mu.Lock()
	manager.locked = false
	manager.mu.Unlock()
}

func (manager *SessionManager) Put(record accounts.SessionRecord, browserID string) error {
	if record.AccountID <= 0 || record.SecretVersion <= 0 {
		return fmt.Errorf("load invalid account session")
	}
	manager.mu.Lock()
	if manager.locked {
		manager.mu.Unlock()
		return &Error{Kind: KindVaultLocked, Message: "The account vault is locked."}
	}
	old := manager.sessions[record.AccountID]
	manager.sessions[record.AccountID] = newRuntimeSession(record, manager.browserIDLocked(record.AccountID, browserID))
	manager.mu.Unlock()
	if old != nil {
		old.close()
	}
	return nil
}

func (manager *SessionManager) ApplyRotation(record accounts.SessionRecord) error {
	session, err := manager.runtime(record.AccountID, record)
	if err != nil {
		return err
	}
	session.mu.Lock()
	currentVersion := session.secretVersion
	if record.SecretVersion <= currentVersion {
		session.mu.Unlock()
		if record.SecretVersion < currentVersion {
			return accounts.ErrStaleSecret
		}
		return nil
	}
	oldCancel := session.cancel
	session.context, session.cancel = context.WithCancel(context.Background())
	session.secretVersion = record.SecretVersion
	session.state = record.State
	session.csrfToken = ""
	session.mu.Unlock()
	oldCancel()
	return nil
}

func (manager *SessionManager) Snapshot(accountID int64) (SessionSnapshot, error) {
	manager.mu.RLock()
	loader := manager.loader
	locked := manager.locked
	manager.mu.RUnlock()
	if locked {
		return SessionSnapshot{}, &Error{Kind: KindVaultLocked, Message: "The account vault is locked."}
	}
	if loader == nil {
		return SessionSnapshot{}, &Error{Kind: KindProtocol, Message: "Account credential storage is unavailable."}
	}
	record, err := loader(context.Background(), accountID)
	if err != nil {
		return SessionSnapshot{}, &Error{Kind: KindAccountNotFound, Message: "The account was not found.", Cause: err}
	}
	session, err := manager.runtime(accountID, record)
	if err != nil {
		return SessionSnapshot{}, err
	}
	session.mu.RLock()
	operationContext := session.context
	browserID := session.browserID
	session.mu.RUnlock()
	return SessionSnapshot{
		AccountID:       accountID,
		SecretVersion:   record.SecretVersion,
		Cookie:          record.Roblosecurity,
		BrowserID:       browserID,
		CookieExpiresAt: copyTime(record.CookieExpiresAt),
		State:           record.State,
		Context:         operationContext,
	}, nil
}

func (manager *SessionManager) VersionCurrent(accountID, version int64) bool {
	session := manager.get(accountID)
	if session == nil {
		return false
	}
	session.mu.RLock()
	defer session.mu.RUnlock()
	return session.secretVersion == version
}

func (manager *SessionManager) CSRF(accountID, version int64) string {
	session := manager.get(accountID)
	if session == nil {
		return ""
	}
	session.mu.RLock()
	defer session.mu.RUnlock()
	if session.secretVersion != version {
		return ""
	}
	return session.csrfToken
}

func (manager *SessionManager) SetCSRF(accountID, version int64, token string) bool {
	session := manager.get(accountID)
	if session == nil || token == "" {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.secretVersion != version {
		return false
	}
	changed := session.csrfToken != token
	session.csrfToken = token
	return changed
}

func (manager *SessionManager) Validate(ctx context.Context, accountID int64, validate func(context.Context, SessionSnapshot) (any, error)) (any, error) {
	snapshot, err := manager.Snapshot(accountID)
	if err != nil {
		return nil, err
	}
	session := manager.get(accountID)
	if session == nil {
		return nil, &Error{Kind: KindAccountNotFound, Message: "The account was not found."}
	}
	value, err, _ := session.validationGroup.Do("validation", func() (any, error) {
		return validate(ctx, snapshot)
	})
	return value, err
}

func (manager *SessionManager) UpdateState(accountID, version int64, state accounts.SessionState) {
	session := manager.get(accountID)
	if session == nil {
		return
	}
	session.mu.Lock()
	if session.secretVersion == version {
		session.state = state
	}
	session.mu.Unlock()
}

func (manager *SessionManager) Remove(accountID int64) {
	manager.mu.Lock()
	session := manager.sessions[accountID]
	delete(manager.sessions, accountID)
	delete(manager.browserIDs, accountID)
	manager.mu.Unlock()
	if session != nil {
		session.close()
	}
}

func (manager *SessionManager) Lock() {
	manager.mu.Lock()
	old := manager.sessions
	manager.sessions = make(map[int64]*runtimeSession)
	manager.locked = true
	manager.mu.Unlock()
	for _, session := range old {
		session.close()
	}
}

func (manager *SessionManager) runtime(accountID int64, record accounts.SessionRecord) (*runtimeSession, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if manager.locked {
		return nil, &Error{Kind: KindVaultLocked, Message: "The account vault is locked."}
	}
	if session := manager.sessions[accountID]; session != nil {
		return session, nil
	}
	session := newRuntimeSession(record, manager.browserIDLocked(accountID, ""))
	manager.sessions[accountID] = session
	return session, nil
}

func (manager *SessionManager) get(accountID int64) *runtimeSession {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	if manager.locked {
		return nil
	}
	return manager.sessions[accountID]
}

func newRuntimeSession(record accounts.SessionRecord, browserID string) *runtimeSession {
	ctx, cancel := context.WithCancel(context.Background())
	return &runtimeSession{
		accountID:     record.AccountID,
		secretVersion: record.SecretVersion,
		state:         record.State,
		browserID:     browserID,
		context:       ctx,
		cancel:        cancel,
	}
}

func (session *runtimeSession) close() {
	session.mu.Lock()
	session.cancel()
	session.csrfToken = ""
	session.mu.Unlock()
}

func copyTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}
