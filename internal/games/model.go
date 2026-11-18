package games

import (
	"net/url"
	"strconv"
	"strings"
)

const (
	MaxFavoriteNicknameLength  = 50
	MaxPrivateServerNameLength = 50
)

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

// GameSection names one part of the game details that Roblox serves from its
// own endpoint. The frontend reads each part after the main details so that
// the main details do not wait for them.
type GameSection string

const (
	GameSectionVotes         GameSection = "votes"
	GameSectionMaturity      GameSection = "maturity"
	GameSectionAvatarRules   GameSection = "avatarRules"
	GameSectionCommunication GameSection = "communication"
	GameSectionVersions      GameSection = "versions"
	GameSectionPlayability   GameSection = "playability"
)

type PlaceSummary struct {
	Place     Place  `json:"place"`
	RootPlace *Place `json:"rootPlace"`
}

type Game struct {
	Place               Place          `json:"place"`
	RootPlace           *Place         `json:"rootPlace"`
	Description         string         `json:"description"`
	Playing             int64          `json:"playing"`
	Visits              int64          `json:"visits"`
	Favorites           int64          `json:"favorites"`
	UpVotes             int64          `json:"upVotes"`
	DownVotes           int64          `json:"downVotes"`
	MaxPlayers          int64          `json:"maxPlayers"`
	Price               int64          `json:"price"`
	Genre               string         `json:"genre"`
	Subgenre            string         `json:"subgenre"`
	AvatarType          string         `json:"avatarType"`
	Maturity            string         `json:"maturity"`
	MaturityDescriptors []string       `json:"maturityDescriptors"`
	AvatarRules         *AvatarRules   `json:"avatarRules"`
	Communication       *Communication `json:"communication"`
	Versions            *PlaceVersions `json:"versions"`
	Playability         *Playability   `json:"playability"`
	// IsUnlisted reports that the games API no longer lists the game, such
	// as one that Roblox banned or put under review. Its details come from
	// the develop API, which lists no player, visit, or favorite counts.
	IsUnlisted            bool  `json:"isUnlisted"`
	CopyingAllowed        bool  `json:"copyingAllowed"`
	PrivateServersAllowed bool  `json:"privateServersAllowed"`
	CreatedAtMs           int64 `json:"createdAtMs"`
	UpdatedAtMs           int64 `json:"updatedAtMs"`
}

type AvatarRules struct {
	CustomAnimationsAllowed bool `json:"customAnimationsAllowed"`
	ItemOverrides           int  `json:"itemOverrides"`
}

type Communication struct {
	VoiceChat bool `json:"voiceChat"`
	Camera    bool `json:"camera"`
}

// Playability tells whether the account can join the game. Roblox puts a game
// that it moderated under review, as RoSeal reads it. Message is the reason
// that Roblox gives for a game that the account cannot join.
type Playability struct {
	IsPlayable    bool   `json:"isPlayable"`
	IsUnderReview bool   `json:"isUnderReview"`
	Message       string `json:"message"`
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
	PingMs          *int64        `json:"pingMs"`
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

type NearbyRegion struct {
	Region             Region  `json:"region"`
	DistanceKilometers float64 `json:"distanceKilometers"`
}

type RegionServerQuery struct {
	PlaceID        int64    `json:"placeId"`
	Region         string   `json:"region"`
	ExcludedJobIDs []string `json:"excludedJobIds"`
}

type PrivateServerQuery struct {
	PlaceID   int64  `json:"placeId"`
	AccountID int64  `json:"accountId"`
	Cursor    string `json:"cursor"`
}

type PrivateServerPage struct {
	Servers    []PrivateServer `json:"servers"`
	NextCursor string          `json:"nextCursor"`
}

type PrivateServer struct {
	ServerID         int64  `json:"serverId"`
	Name             string `json:"name"`
	OwnerID          int64  `json:"ownerId"`
	OwnerName        string `json:"ownerName"`
	OwnerDisplayName string `json:"ownerDisplayName"`
	Owned            bool   `json:"owned"`
	Playing          int64  `json:"playing"`
	MaxPlayers       int64  `json:"maxPlayers"`
	Running          bool   `json:"running"`
	LaunchLink       string `json:"launchLink"`
}

type OwnedPrivateServerPage struct {
	Servers    []OwnedPrivateServer `json:"servers"`
	NextCursor string               `json:"nextCursor"`
}

type OwnedPrivateServer struct {
	ServerID    int64  `json:"serverId"`
	Name        string `json:"name"`
	Active      bool   `json:"active"`
	UniverseID  int64  `json:"universeId"`
	PlaceID     int64  `json:"placeId"`
	GameName    string `json:"gameName"`
	Price       int64  `json:"price"`
	Renews      bool   `json:"renews"`
	ExpiresAtMs int64  `json:"expiresAtMs"`
}

type PrivateServerSettings struct {
	ServerID       int64               `json:"serverId"`
	Name           string              `json:"name"`
	GameName       string              `json:"gameName"`
	Active         bool                `json:"active"`
	FriendsAllowed bool                `json:"friendsAllowed"`
	Renews         bool                `json:"renews"`
	Expired        bool                `json:"expired"`
	ExpiresAtMs    int64               `json:"expiresAtMs"`
	Price          int64               `json:"price"`
	JoinLink       string              `json:"joinLink"`
	Users          []PrivateServerUser `json:"users"`
}

type PrivateServerUser struct {
	UserID      int64  `json:"userId"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	ImageURL    string `json:"imageUrl"`
}

type PrivateServerOption string

const (
	PrivateServerActive         PrivateServerOption = "active"
	PrivateServerFriendsAllowed PrivateServerOption = "friendsAllowed"
	PrivateServerRenews         PrivateServerOption = "renews"
)

type SavedPrivateServer struct {
	ID          int64  `json:"id"`
	PlaceID     int64  `json:"placeId"`
	Name        string `json:"name"`
	AccessCode  string `json:"accessCode"`
	LinkCode    string `json:"linkCode"`
	LaunchLink  string `json:"launchLink"`
	CreatedAtMs int64  `json:"createdAtMs"`
}

type ListedPrivateServerPage struct {
	Servers []ListedPrivateServer `json:"servers"`
}

type ListedPrivateServer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Badge is a badge of a universe. Roblox owns badges per universe, so every
// place of the universe shares them.
type Badge struct {
	BadgeID     int64  `json:"badgeId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IconURL     string `json:"iconUrl"`
	Enabled     bool   `json:"enabled"`
	// IsModerated reports that Roblox moderated the badge, whose icon it blocks.
	IsModerated        bool    `json:"isModerated"`
	AwardedCount       int64   `json:"awardedCount"`
	AwardedPastDay     int64   `json:"awardedPastDay"`
	PastDayWinFraction float64 `json:"pastDayWinFraction"`
	CreatedAtMs        int64   `json:"createdAtMs"`
}

type BadgeAwardQuery struct {
	AccountID int64   `json:"accountId"`
	BadgeIDs  []int64 `json:"badgeIds"`
}

type BadgeAward struct {
	BadgeID     int64 `json:"badgeId"`
	AwardedAtMs int64 `json:"awardedAtMs"`
}

// GamePass belongs to a universe and is shared by its places.
type GamePass struct {
	GamePassID   int64  `json:"gamePassId"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	IconURL      string `json:"iconUrl"`
	IsForSale    bool   `json:"isForSale"`
	PriceInRobux int64  `json:"priceInRobux"`
	IsOwned      bool   `json:"isOwned"`
	// IsModerated reports that Roblox moderated the pass, whose icon it blocks.
	IsModerated bool  `json:"isModerated"`
	CreatedAtMs int64 `json:"createdAtMs"`
	UpdatedAtMs int64 `json:"updatedAtMs"`
}

// FriendServer is a server that friends of the account are in.
type FriendServer struct {
	JobID   string           `json:"jobId"`
	Friends []FriendInServer `json:"friends"`
}

type FriendInServer struct {
	UserID      int64  `json:"userId"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	ImageURL    string `json:"imageUrl"`
}

func PrivateServerLaunchLink(placeID int64, accessCode, linkCode string) string {
	if linkCode != "" {
		link := url.URL{Scheme: "https", Host: "www.roblox.com", Path: "/games/" + strconv.FormatInt(placeID, 10)}
		link.RawQuery = url.Values{"privateServerLinkCode": {linkCode}}.Encode()
		return link.String()
	}
	return "roblox://experiences/start?placeId=" + strconv.FormatInt(placeID, 10) + "&accessCode=" + url.QueryEscape(accessCode)
}

const javaScriptMaxSafeInteger = 1<<53 - 1

func ValidID(id int64) bool {
	return id > 0 && id <= javaScriptMaxSafeInteger
}

func ValidJobID(jobID string) bool {
	// Canonical UUID: 36 characters with dashes at 8, 13, 18, and 23.
	if len(jobID) != 36 {
		return false
	}
	for index, character := range jobID {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if character != '-' {
				return false
			}
		} else if !strings.ContainsRune("0123456789abcdefABCDEF", character) {
			return false
		}
	}
	return true
}
