package roblox

import (
	"crypto/rand"
	"encoding/binary"
	"strconv"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
)

const EventTrackerCookieName = "RBXEventTrackerV2"

func NewBrowserID() string {
	for {
		var randomBytes [8]byte
		_, _ = rand.Read(randomBytes[:])
		// Keep 53 bits, so the ID fits in a JavaScript number.
		if browserID := binary.BigEndian.Uint64(randomBytes[:]) >> 11; browserID != 0 {
			return strconv.FormatUint(browserID, 10)
		}
	}
}

func EventTrackerCookieValue(browserID string) string {
	return "browserid=" + browserID
}

func formatAuthCookie(cookie, browserID string) string {
	return accounts.RoblosecurityCookieName + "=" + cookie + "; " + EventTrackerCookieName + "=" + EventTrackerCookieValue(browserID)
}

func (manager *SessionManager) BrowserID(accountID int64) string {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	return manager.browserIDLocked(accountID, "")
}

func (manager *SessionManager) ResetBrowserIDs() {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	clear(manager.browserIDs)
}

func (manager *SessionManager) browserIDLocked(accountID int64, preferred string) string {
	if browserID := manager.browserIDs[accountID]; browserID != "" {
		return browserID
	}
	for {
		if preferred == "" {
			preferred = NewBrowserID()
		}
		if _, exists := manager.usedBrowserIDs[preferred]; !exists {
			manager.browserIDs[accountID] = preferred
			manager.usedBrowserIDs[preferred] = struct{}{}
			return preferred
		}
		preferred = ""
	}
}
