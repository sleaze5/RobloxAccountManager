package games

import "strings"

type Place struct {
	PlaceID         int64  `json:"placeId"`
	UniverseID      int64  `json:"universeId"`
	Name            string `json:"name"`
	CreatorID       int64  `json:"creatorId"`
	CreatorName     string `json:"creatorName"`
	CreatorType     string `json:"creatorType"`
	CreatorVerified bool   `json:"creatorVerified"`
	IconURL         string `json:"iconUrl"`
	// Nickname is the user's label for a favorite place and is empty for other places.
	Nickname string `json:"nickname"`
}

type Game struct {
	Place Place `json:"place"`
	// RootPlace is the universe's starting place when Place is a subplace, and nil otherwise.
	RootPlace             *Place         `json:"rootPlace"`
	Description           string         `json:"description"`
	Playing               int64          `json:"playing"`
	Visits                int64          `json:"visits"`
	Favorites             int64          `json:"favorites"`
	UpVotes               int64          `json:"upVotes"`
	DownVotes             int64          `json:"downVotes"`
	MaxPlayers            int64          `json:"maxPlayers"`
	Price                 int64          `json:"price"`
	Genre                 string         `json:"genre"`
	Subgenre              string         `json:"subgenre"`
	AvatarType            string         `json:"avatarType"`
	Maturity              string         `json:"maturity"`
	MaturityDescriptors   []string       `json:"maturityDescriptors"`
	AvatarRules           *AvatarRules   `json:"avatarRules"`
	Communication         *Communication `json:"communication"`
	Versions              *PlaceVersions `json:"versions"`
	CopyingAllowed        bool           `json:"copyingAllowed"`
	PrivateServersAllowed bool           `json:"privateServersAllowed"`
	CreatedAtMS           int64          `json:"createdAtMs"`
	UpdatedAtMS           int64          `json:"updatedAtMs"`
}

// AvatarRules describes how a universe changes avatars that join it.
type AvatarRules struct {
	CustomAnimationsAllowed bool `json:"customAnimationsAllowed"`
	ItemOverrides           int  `json:"itemOverrides"`
}

// Communication describes the voice and camera features a universe enables.
type Communication struct {
	VoiceChat bool `json:"voiceChat"`
	Camera    bool `json:"camera"`
}

// PlaceVersions holds the latest saved and published versions of a root place.
type PlaceVersions struct {
	Saved     int64 `json:"saved"`
	Published int64 `json:"published"`
}

type SearchQuery struct {
	Text      string `json:"text"`
	SessionID string `json:"sessionId"`
	PageToken string `json:"pageToken"`
}

type SearchPage struct {
	Places        []Place `json:"places"`
	SessionID     string  `json:"sessionId"`
	NextPageToken string  `json:"nextPageToken"`
}

// ServerOrder is a server list order Roblox supports.
type ServerOrder string

const (
	ServerOrderRecommended ServerOrder = "Recommended"
	ServerOrderBestPing    ServerOrder = "BestLatency"
	ServerOrderMostPlayers ServerOrder = "OccupancyDesc"
	ServerOrderFewest      ServerOrder = "OccupancyAsc"
)

// ServerQuery selects one page of a place's public servers.
type ServerQuery struct {
	PlaceID int64 `json:"placeId"`
	// Cursor is empty for the first page and otherwise a cursor from a previous ServerPage.
	Cursor      string      `json:"cursor"`
	Order       ServerOrder `json:"order"`
	ExcludeFull bool        `json:"excludeFull"`
	// Limit is one of the page sizes Roblox accepts: 10, 25, 50, or 100.
	Limit int `json:"limit"`
}

type ServerPage struct {
	Servers        []Server `json:"servers"`
	NextCursor     string   `json:"nextCursor"`
	PreviousCursor string   `json:"previousCursor"`
}

// Server is a running public server of a place.
// Roblox reveals ping, language matches, friends, and player headshots only to signed-in users,
// so the pointer fields are nil and PlayerImages is empty for signed-out reads.
type Server struct {
	JobID      string  `json:"jobId"`
	Playing    int64   `json:"playing"`
	MaxPlayers int64   `json:"maxPlayers"`
	FPS        float64 `json:"fps"`
	PingMS     *int64  `json:"pingMs"`
	// LanguageMatches counts players who share the reading account's language.
	LanguageMatches *int64 `json:"languageMatches"`
	// Friends counts friends of the reading account in the server.
	Friends      *int64   `json:"friends"`
	PlayerImages []string `json:"playerImages"`
	// Record is nil when the RoValra integration is off or has no record of the server.
	Record *ServerRecord `json:"record"`
}

// ServerRecord is what the RoValra integration has recorded about a server.
// RoValra collects records from its users, so a server can be missing or already closed.
type ServerRecord struct {
	JobID string `json:"jobId"`
	// PlaceVersion is zero when RoValra did not record the version.
	PlaceVersion int64 `json:"placeVersion"`
	// FirstSeenMs is when RoValra first saw the server, which approximates its start; zero when unknown.
	FirstSeenMs  int64  `json:"firstSeenMs"`
	City         string `json:"city"`
	Region       string `json:"region"`
	Country      string `json:"country"`
	CountryCode  string `json:"countryCode"`
	DatacenterID int64  `json:"datacenterId"`
	Address      string `json:"address"`
}

// RecordOrder is a server list order RoValra supports.
type RecordOrder string

const (
	RecordOrderNewest RecordOrder = "newest"
	RecordOrderOldest RecordOrder = "oldest"
)

// RecordQuery selects one page of the servers RoValra has recorded for a place.
type RecordQuery struct {
	PlaceID int64 `json:"placeId"`
	// Region is a ServerRegion code, or empty for every region. Region pages are always newest first.
	Region string      `json:"region"`
	Order  RecordOrder `json:"order"`
	// Cursor is zero for the first page and otherwise a cursor from a previous RecordPage.
	Cursor int64 `json:"cursor"`
	// Limit is one of the page sizes RoValra accepts: 10, 50, or 100.
	Limit int `json:"limit"`
}

type RecordPage struct {
	Servers []ServerRecord `json:"servers"`
	// NextCursor is zero on the last page.
	NextCursor int64 `json:"nextCursor"`
}

// ServerRegion is a server location RoValra can filter by.
type ServerRegion struct {
	Code        string   `json:"code"`
	CountryCode string   `json:"countryCode"`
	Cities      []string `json:"cities"`
	Servers     int64    `json:"servers"`
}

// ServerStats summarizes the servers RoValra has recorded for a place.
type ServerStats struct {
	Regions []ServerRegion `json:"regions"`
	// NewestVersion is zero when RoValra has no version for the place.
	NewestVersion int64 `json:"newestVersion"`
	TotalServers  int64 `json:"totalServers"`
}

// Region identifies a RoValra server-browser region: a US state or a country elsewhere.
type Region struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
}

// NearestServer is the recorded server found in or near the preferred region.
type NearestServer struct {
	JobID  string `json:"jobId"`
	Region Region `json:"region"`
}

// ValidJobID accepts the GUIDs Roblox uses to identify servers.
func ValidJobID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for index, char := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if char != '-' {
				return false
			}
		} else if !strings.ContainsRune("0123456789abcdefABCDEF", char) {
			return false
		}
	}
	return true
}
