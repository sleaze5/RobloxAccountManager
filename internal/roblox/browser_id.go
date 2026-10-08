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
		var data [8]byte
		_, _ = rand.Read(data[:])
		// Keep 53 bits, so the ID fits in a JavaScript number.
		if value := binary.BigEndian.Uint64(data[:]) >> 11; value != 0 {
			return strconv.FormatUint(value, 10)
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
	manager.mu.Lock()
	defer manager.mu.Unlock()
	return manager.browserIDLocked(accountID, "")
}

func (manager *SessionManager) ResetBrowserIDs() {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	clear(manager.browserIDs)
}

func (manager *SessionManager) browserIDLocked(accountID int64, preferred string) string {
	if value := manager.browserIDs[accountID]; value != "" {
		return value
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
