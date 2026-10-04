package accounts

import "time"

type SessionState string

const (
	StateActive         SessionState = "active"
	StateUnknown        SessionState = "unknown"
	StateReauthRequired SessionState = "reauth_required"
	StateChallenged     SessionState = "challenged"
	StateDisabled       SessionState = "disabled"
)

func (s SessionState) Valid() bool {
	switch s {
	case StateActive, StateUnknown, StateReauthRequired, StateChallenged, StateDisabled:
		return true
	default:
		return false
	}
}

type Identity struct {
	RobloxUserID int64
	Username     string
	DisplayName  string
	CreatedAt    time.Time
}

type TagKind string

const (
	TagKindCustom   TagKind = "custom"
	TagKindFavorite TagKind = "favorite"
)

type TagView struct {
	ID   int64   `json:"id"`
	Name string  `json:"name"`
	Kind TagKind `json:"kind"`
}

type SessionRecord struct {
	AccountID         int64
	Roblosecurity     string
	CookieExpiresAt   *time.Time
	State             SessionState
	StateReason       string
	SecretVersion     int64
	ImportedAt        time.Time
	ReplacedAt        *time.Time
	RotatedAt         *time.Time
	LastValidatedAt   *time.Time
	LastSuccessAt     *time.Time
	LastAuthFailureAt *time.Time
	UpdatedAt         time.Time
}

type AccountView struct {
	ID                int64        `json:"id"`
	RobloxUserID      int64        `json:"robloxUserId"`
	Username          string       `json:"username"`
	DisplayName       string       `json:"displayName"`
	Tags              []TagView    `json:"tags"`
	State             SessionState `json:"state"`
	StateReason       string       `json:"stateReason,omitempty"`
	SecretVersion     int64        `json:"secretVersion"`
	CookieExpiresAtMS *int64       `json:"cookieExpiresAtMs,omitempty"`
	CreatedAtMS       int64        `json:"createdAtMs"`
	ImportedAtMS      int64        `json:"importedAtMs"`
	RotatedAtMS       *int64       `json:"rotatedAtMs,omitempty"`
	LastValidatedAtMS *int64       `json:"lastValidatedAtMs,omitempty"`
	UpdatedAtMS       int64        `json:"updatedAtMs"`
}

type AccountCursor struct {
	DisplayOrder int64 `json:"displayOrder"`
	AccountID    int64 `json:"accountId"`
}

type AccountPage struct {
	Accounts   []AccountView  `json:"accounts"`
	NextCursor *AccountCursor `json:"nextCursor,omitempty"`
}

type AccountQuery struct {
	Cursor *AccountCursor `json:"cursor,omitempty"`
	Search string         `json:"search,omitempty"`
	TagIDs []int64        `json:"tagIds,omitempty"`
}
