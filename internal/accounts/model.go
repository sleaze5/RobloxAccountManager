package accounts

import "time"

type SessionState string

const (
	StateActive         SessionState = "active"
	StateUnknown        SessionState = "unknown"
	StateReauthRequired SessionState = "reauth_required"
	StateChallenged     SessionState = "challenged"
	StateDisabled       SessionState = "disabled"
	// StateModerated marks a working cookie of an account that Roblox warned,
	// banned, or terminated. Roblox answers most of its requests with 403.
	StateModerated SessionState = "moderated"
)

func (state SessionState) Valid() bool {
	switch state {
	case StateActive, StateUnknown, StateReauthRequired, StateChallenged, StateDisabled, StateModerated:
		return true
	default:
		return false
	}
}

type ModerationKind string

const (
	ModerationWarned     ModerationKind = "warned"
	ModerationBanned     ModerationKind = "banned"
	ModerationTerminated ModerationKind = "terminated"
	// ModerationLocked is a moderated account without a punishment, which
	// Roblox unlocks after a human verification on its website.
	ModerationLocked ModerationKind = "locked"
)

// Moderation is the Roblox punishment of a saved account. Only a ban has an
// end. Title and Message are the explanation Roblox wrote for the user.
type Moderation struct {
	Kind       ModerationKind `json:"kind"`
	BeginsAtMs int64          `json:"beginsAtMs"`
	EndsAtMs   *int64         `json:"endsAtMs,omitempty"`
	Title      string         `json:"title"`
	Message    string         `json:"message"`
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
	AccountID       int64
	Roblosecurity   string
	CookieExpiresAt *time.Time
	State           SessionState
	SecretVersion   int64
	ImportedAt      time.Time
	LastValidatedAt *time.Time
}

type AccountView struct {
	ID                int64        `json:"id"`
	RobloxUserID      int64        `json:"robloxUserId"`
	Username          string       `json:"username"`
	DisplayName       string       `json:"displayName"`
	HasSavedPassword  bool         `json:"hasSavedPassword"`
	Tags              []TagView    `json:"tags"`
	State             SessionState `json:"state"`
	StateReason       string       `json:"stateReason,omitempty"`
	Moderation        *Moderation  `json:"moderation,omitempty"`
	SecretVersion     int64        `json:"secretVersion"`
	CookieExpiresAtMs *int64       `json:"cookieExpiresAtMs,omitempty"`
	CreatedAtMs       int64        `json:"createdAtMs"`
	ImportedAtMs      int64        `json:"importedAtMs"`
	RotatedAtMs       *int64       `json:"rotatedAtMs,omitempty"`
	LastValidatedAtMs *int64       `json:"lastValidatedAtMs,omitempty"`
	UpdatedAtMs       int64        `json:"updatedAtMs"`
}

type AccountCursor struct {
	DisplayOrder int64 `json:"displayOrder"`
	AccountID    int64 `json:"accountId"`
}

type AccountPage struct {
	Accounts   []AccountView  `json:"accounts"`
	NextCursor *AccountCursor `json:"nextCursor,omitempty"`
	Total      int64          `json:"total"`
}

type AccountQuery struct {
	Cursor *AccountCursor `json:"cursor,omitempty"`
	Search string         `json:"search,omitempty"`
	TagIDs []int64        `json:"tagIds,omitempty"`
}
