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
	Nickname        string `json:"nickname"`
}

type PlaceSummary struct {
	Place     Place  `json:"place"`
	RootPlace *Place `json:"rootPlace"`
}

type Game struct {
	Place                 Place          `json:"place"`
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

type AvatarRules struct {
	CustomAnimationsAllowed bool `json:"customAnimationsAllowed"`
	ItemOverrides           int  `json:"itemOverrides"`
}

type Communication struct {
	VoiceChat bool `json:"voiceChat"`
	Camera    bool `json:"camera"`
}

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

type ServerOrder string

const (
	ServerOrderRecommended ServerOrder = "Recommended"
	ServerOrderBestPing    ServerOrder = "BestLatency"
	ServerOrderMostPlayers ServerOrder = "OccupancyDesc"
	ServerOrderFewest      ServerOrder = "OccupancyAsc"
)

type ServerQuery struct {
	PlaceID     int64       `json:"placeId"`
	Cursor      string      `json:"cursor"`
	Order       ServerOrder `json:"order"`
	ExcludeFull bool        `json:"excludeFull"`
	Limit       int         `json:"limit"`
}

type ServerPage struct {
	Servers        []Server `json:"servers"`
	NextCursor     string   `json:"nextCursor"`
	PreviousCursor string   `json:"previousCursor"`
}

type Server struct {
	JobID           string        `json:"jobId"`
	Playing         int64         `json:"playing"`
	MaxPlayers      int64         `json:"maxPlayers"`
	FPS             float64       `json:"fps"`
	PingMS          *int64        `json:"pingMs"`
	LanguageMatches *int64        `json:"languageMatches"`
	Friends         *int64        `json:"friends"`
	PlayerImages    []string      `json:"playerImages"`
	Record          *ServerRecord `json:"record"`
}

type ServerRecord struct {
	JobID        string `json:"jobId"`
	PlaceVersion int64  `json:"placeVersion"`
	FirstSeenMs  int64  `json:"firstSeenMs"`
	City         string `json:"city"`
	Region       string `json:"region"`
	Country      string `json:"country"`
	CountryCode  string `json:"countryCode"`
	DatacenterID int64  `json:"datacenterId"`
	Address      string `json:"address"`
}

type RecordOrder string

const (
	RecordOrderNewest RecordOrder = "newest"
	RecordOrderOldest RecordOrder = "oldest"
)

type RecordQuery struct {
	PlaceID int64       `json:"placeId"`
	Region  string      `json:"region"`
	Order   RecordOrder `json:"order"`
	Cursor  int64       `json:"cursor"`
	Limit   int         `json:"limit"`
}

type RecordPage struct {
	Servers    []ServerRecord `json:"servers"`
	NextCursor int64          `json:"nextCursor"`
}

type ServerRegion struct {
	Code        string   `json:"code"`
	CountryCode string   `json:"countryCode"`
	Cities      []string `json:"cities"`
	Servers     int64    `json:"servers"`
}

type ServerStats struct {
	Regions       []ServerRegion `json:"regions"`
	NewestVersion int64          `json:"newestVersion"`
	TotalServers  int64          `json:"totalServers"`
}

type Region struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
}

type NearestServer struct {
	JobID  string `json:"jobId"`
	Region Region `json:"region"`
}

func ValidJobID(value string) bool {
	// Canonical UUID: 36 characters with dashes at 8, 13, 18, and 23.
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
