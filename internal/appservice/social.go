package appservice

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/logging"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

const (
	maxSocialCursorLength   = 4096
	minPlayerSearchLength   = 3
	maxPlayerSearchLength   = 50
	maxSocialPresenceUsers  = 1000
	maxNicknameLength       = 20
	friendConnectionWorkers = 4
)

var (
	playerIDSearch = regexp.MustCompile(`^(?i)id:\s*(\d{1,18})$`)
	playerUsername = regexp.MustCompile(`^[A-Za-z0-9_]{3,20}$`)
)

type SocialList string

const (
	SocialFriends   SocialList = "friends"
	SocialFollowers SocialList = "followers"
	SocialFollowing SocialList = "following"
	SocialRequests  SocialList = "requests"
)

type FriendRequestView struct {
	SentAtMs      int64    `json:"sentAtMs"`
	Origin        string   `json:"origin"`
	MutualFriends []string `json:"mutualFriends"`
}

type SocialUser struct {
	UserID      int64                        `json:"userId"`
	Username    string                       `json:"username"`
	DisplayName string                       `json:"displayName"`
	Nickname    string                       `json:"nickname"`
	ImageURL    string                       `json:"imageUrl"`
	Presence    *robloxservices.UserPresence `json:"presence"`
	Request     *FriendRequestView           `json:"request"`
	// Trusted reports whether the user is a trusted friend of the browsing account.
	Trusted bool `json:"trusted"`
}

// SocialRelationship describes the user as the browsing account sees them.
type SocialRelationship struct {
	Self       bool                         `json:"self"`
	Friend     robloxservices.FriendStatus  `json:"friend"`
	Trusted    robloxservices.TrustedStatus `json:"trusted"`
	Following  bool                         `json:"following"`
	FollowedBy bool                         `json:"followedBy"`
	Nickname   string                       `json:"nickname"`
}

// SocialRelationshipSnapshot is the relationship and the parts of it that
// could not be read.
type SocialRelationshipSnapshot struct {
	Relationship SocialRelationship `json:"relationship"`
	Unavailable  []string           `json:"unavailable"`
}

type SocialPage struct {
	People       []SocialUser `json:"people"`
	NextCursor   string       `json:"nextCursor"`
	RequestCount *int64       `json:"requestCount"`
	Unavailable  []string     `json:"unavailable"`
}

type FriendConnection struct {
	AccountID   int64                        `json:"accountId"`
	Username    string                       `json:"username"`
	DisplayName string                       `json:"displayName"`
	Friend      robloxservices.FriendStatus  `json:"friend"`
	Trusted     robloxservices.TrustedStatus `json:"trusted"`
	Following   bool                         `json:"following"`
	FollowedBy  bool                         `json:"followedBy"`
}

// SocialProfileSection names one part of a social profile. The frontend reads
// each part on its own so that it shows every part as soon as it arrives.
type SocialProfileSection string

const (
	SocialSectionAbout         SocialProfileSection = "about"
	SocialSectionRelationship  SocialProfileSection = "relationship"
	SocialSectionFriends       SocialProfileSection = "friends"
	SocialSectionFollowers     SocialProfileSection = "followers"
	SocialSectionFollowing     SocialProfileSection = "following"
	SocialSectionPremium       SocialProfileSection = "premium"
	SocialSectionPrimaryGroup  SocialProfileSection = "primaryGroup"
	SocialSectionRobloxBadges  SocialProfileSection = "robloxBadges"
	SocialSectionSocialLinks   SocialProfileSection = "socialLinks"
	SocialSectionInventory     SocialProfileSection = "inventory"
	SocialSectionConnections   SocialProfileSection = "connections"
	SocialSectionPreviousNames SocialProfileSection = "previousNames"
	SocialSectionCommunities   SocialProfileSection = "communities"
	SocialSectionCollectibles  SocialProfileSection = "collectibles"
	SocialSectionTrading       SocialProfileSection = "trading"
)

// SocialProfile holds the fields of the one section that was read.
type SocialProfile struct {
	UserID           int64                            `json:"userId"`
	Section          SocialProfileSection             `json:"section"`
	IsComplete       bool                             `json:"isComplete"`
	Description      string                           `json:"description"`
	Verified         bool                             `json:"verified"`
	Banned           bool                             `json:"banned"`
	Premium          *bool                            `json:"premium"`
	InventoryVisible *bool                            `json:"inventoryVisible"`
	CreatedAtMs      int64                            `json:"createdAtMs"`
	FriendCount      *int64                           `json:"friendCount"`
	FollowerCount    *int64                           `json:"followerCount"`
	FollowingCount   *int64                           `json:"followingCount"`
	PreviousNames    []string                         `json:"previousNames"`
	PrimaryGroup     *robloxservices.UserPrimaryGroup `json:"primaryGroup"`
	RobloxBadges     []robloxservices.RobloxBadge     `json:"robloxBadges"`
	SocialLinks      []robloxservices.UserSocialLink  `json:"socialLinks"`
	SavedAccountID   int64                            `json:"savedAccountId"`
	Connections      []FriendConnection               `json:"connections"`
	Relationship     SocialRelationship               `json:"relationship"`
	CommunityCount   *int64                           `json:"communityCount"`
	// TradeEligibility says whether the account can trade with the user.
	TradeEligibility *TradeEligibility `json:"tradeEligibility"`
	// The collectible fields stay nil when the user's inventory is private.
	CollectibleCount                   *int64 `json:"collectibleCount"`
	CollectibleRecentAveragePriceTotal *int64 `json:"collectibleRecentAveragePriceTotal"`
	// Unavailable lists the relationship reads that failed. The relationship
	// section stays complete without them, because the others still apply.
	Unavailable []string `json:"unavailable"`
}

type socialEntries struct {
	userIDs    []int64
	requests   []robloxservices.FriendRequest
	nextCursor string
}

// GetSocialPage lists one page of a user's friends, followers, or followings,
// or the account's own friend requests, with presence as the account sees it.
// Friends come as one page because Roblox caps them.
func (service *Service) GetSocialPage(ctx context.Context, accountID, userID int64, list SocialList, cursor string) (SocialPage, error) {
	if len(cursor) > maxSocialCursorLength {
		return SocialPage{}, socialInputError("This list has expired. Refresh it.")
	}
	ctx, version, logger, done, err := service.beginSocial(ctx, accountID, string(list))
	if err != nil {
		return SocialPage{}, err
	}
	defer done()
	account, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return SocialPage{}, mapRepositoryError(err)
	}
	own := userID == account.RobloxUserID
	entries, err := service.socialEntries(ctx, accountID, version, userID, own, list, cursor)
	if err != nil {
		return SocialPage{}, socialFailure(logger, err)
	}
	optional := socialPageReads{presence: list != SocialRequests, requestCount: own && (list == SocialFriends || list == SocialRequests), selfUserID: account.RobloxUserID}
	page, err := service.socialPage(ctx, accountID, version, logger, entries, optional)
	if err == nil && list == SocialFriends {
		slices.SortStableFunc(page.People, compareFriends)
	}
	return page, err
}

// SearchPlayers finds Roblox users by username or display name, with presence
// as the account sees it.
func (service *Service) SearchPlayers(ctx context.Context, accountID int64, keyword, cursor string) (SocialPage, error) {
	keyword = strings.TrimSpace(keyword)
	match := playerIDSearch.FindStringSubmatch(keyword)
	if length := utf8.RuneCountInString(keyword); match == nil && (length < minPlayerSearchLength || length > maxPlayerSearchLength) {
		return SocialPage{}, socialInputError(fmt.Sprintf("Enter %d to %d characters, or id: and a user ID, to search for players.", minPlayerSearchLength, maxPlayerSearchLength))
	}
	if len(cursor) > maxSocialCursorLength {
		return SocialPage{}, socialInputError("These search results have expired. Search again.")
	}
	ctx, version, logger, done, err := service.beginSocial(ctx, accountID, "player-search")
	if err != nil {
		return SocialPage{}, err
	}
	defer done()
	var entries socialEntries
	if match != nil {
		entries.userIDs, err = service.playerByID(ctx, match[1])
	} else {
		entries.userIDs, entries.nextCursor, err = service.users.SearchUsers(ctx, accountID, version, keyword, cursor)
	}
	if err != nil {
		return SocialPage{}, socialFailure(logger, err)
	}
	account, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return SocialPage{}, mapRepositoryError(err)
	}
	return service.socialPage(ctx, accountID, version, logger, entries, socialPageReads{presence: true, selfUserID: account.RobloxUserID})
}

// playerByID returns the user ID when Roblox knows the user, or no IDs.
func (service *Service) playerByID(ctx context.Context, userIDText string) ([]int64, error) {
	userID, err := strconv.ParseInt(userIDText, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse player id: %w", err)
	}
	if userID <= 0 {
		return []int64{}, nil
	}
	users, err := service.users.Names(ctx, []int64{userID})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 || users[0].Username == "" {
		return []int64{}, nil
	}
	return []int64{userID}, nil
}

// socialPageReads chooses the optional parts of a social list. selfUserID is
// the browsing account, whose trusted friends are marked in every list.
type socialPageReads struct {
	presence     bool
	requestCount bool
	selfUserID   int64
}

func (service *Service) socialPage(ctx context.Context, accountID, version int64, logger *slog.Logger, entries socialEntries, optional socialPageReads) (SocialPage, error) {
	users, err := service.users.Names(ctx, entries.userIDs)
	if err != nil {
		return SocialPage{}, socialFailure(logger, err)
	}
	page := SocialPage{NextCursor: entries.nextCursor}
	var presences []robloxservices.UserPresence
	var images, nicknames map[int64]string
	reads := []accountRead{{"avatars", func() error {
		var readError error
		images, readError = service.headshotImages(ctx, entries.userIDs)
		return readError
	}}, {"nicknames", func() error {
		var readError error
		nicknames, readError = service.users.Nicknames(ctx, accountID, version, entries.userIDs)
		return readError
	}}}
	if optional.presence {
		reads = append(reads, accountRead{"presence", func() error {
			var readError error
			presences, readError = service.presence.UserPresences(ctx, accountID, version, entries.userIDs)
			return readError
		}})
	}
	if optional.requestCount {
		reads = append(reads, accountRead{"requestCount", func() error {
			var readError error
			page.RequestCount, readError = service.users.FriendRequestCount(ctx, accountID, version)
			return readError
		}})
	}
	trustedUsers := map[int64]bool{}
	reads = append(reads, accountRead{"trusted", func() error {
		trustedIDs, readError := service.users.TrustedFriendIDs(ctx, accountID, version, optional.selfUserID)
		for _, trustedID := range trustedIDs {
			trustedUsers[trustedID] = true
		}
		return readError
	}})
	if page.Unavailable, err = service.runAccountReads(ctx, accountID, version, "social", logger, reads); err != nil {
		return SocialPage{}, err
	}
	page.People = socialPeople(users, entries.requests, presences, optional.presence)
	for index := range page.People {
		person := &page.People[index]
		person.ImageURL, person.Nickname, person.Trusted = images[person.UserID], nicknames[person.UserID], trustedUsers[person.UserID]
	}
	logger.Debug("social list read", "count", len(page.People), "failed_sections", len(page.Unavailable))
	return page, nil
}

func (service *Service) socialEntries(ctx context.Context, accountID, version, userID int64, own bool, list SocialList, cursor string) (socialEntries, error) {
	var entries socialEntries
	var err error
	switch list {
	case SocialFriends:
		entries.userIDs, err = service.users.FriendIDs(ctx, accountID, version, userID)
	case SocialFollowers:
		entries.userIDs, entries.nextCursor, err = service.users.Follows(ctx, accountID, version, userID, "followers", cursor)
	case SocialFollowing:
		entries.userIDs, entries.nextCursor, err = service.users.Follows(ctx, accountID, version, userID, "followings", cursor)
	case SocialRequests:
		if !own {
			return entries, socialInputError("Friend requests are only available for the selected account.")
		}
		entries.requests, entries.nextCursor, err = service.users.FriendRequests(ctx, accountID, version, cursor)
		for _, request := range entries.requests {
			entries.userIDs = append(entries.userIDs, request.UserID)
		}
	default:
		return entries, socialInputError("Choose friends, followers, following, or requests.")
	}
	return entries, err
}

func socialPeople(users []robloxservices.UserView, requests []robloxservices.FriendRequest, presences []robloxservices.UserPresence, withPresence bool) []SocialUser {
	presenceByUser := indexPresences(presences)
	people := make([]SocialUser, 0, len(users))
	for index, user := range users {
		person := SocialUser{UserID: user.UserID, Username: user.Username, DisplayName: cmp.Or(user.DisplayName, user.Username)}
		if withPresence {
			presence := presenceOrOffline(presenceByUser, user.UserID)
			person.Presence = &presence
		}
		if index < len(requests) {
			request := requests[index]
			person.Request = &FriendRequestView{Origin: request.OriginSourceType, MutualFriends: request.MutualFriends}
			if !request.SentAt.IsZero() {
				person.Request.SentAtMs = request.SentAt.UnixMilli()
			}
		}
		people = append(people, person)
	}
	return people
}

func indexPresences(presences []robloxservices.UserPresence) map[int64]robloxservices.UserPresence {
	byUser := make(map[int64]robloxservices.UserPresence, len(presences))
	for _, presence := range presences {
		byUser[presence.UserID] = presence
	}
	return byUser
}

func presenceOrOffline(byUser map[int64]robloxservices.UserPresence, userID int64) robloxservices.UserPresence {
	if presence, found := byUser[userID]; found {
		return presence
	}
	return robloxservices.UserPresence{UserID: userID, UserPresenceType: robloxservices.PresenceTypeOffline}
}

func compareFriends(left, right SocialUser) int {
	return cmp.Or(
		cmp.Compare(friendPresenceRank(left.Presence), friendPresenceRank(right.Presence)),
		strings.Compare(strings.ToLower(left.DisplayName), strings.ToLower(right.DisplayName)),
	)
}

// GetSocialPresences reads the presence of listed users as the account sees
// it, with one entry per user so users Roblox omits read as offline.
func (service *Service) GetSocialPresences(ctx context.Context, accountID int64, userIDs []int64) ([]robloxservices.UserPresence, error) {
	if len(userIDs) > maxSocialPresenceUsers {
		return nil, socialInputError("Too many users were requested. Refresh the list.")
	}
	ctx, version, logger, done, err := service.beginSocial(ctx, accountID, "presence")
	if err != nil {
		return nil, err
	}
	defer done()
	presences, err := service.presence.UserPresences(ctx, accountID, version, userIDs)
	if err != nil {
		return nil, socialFailure(logger, err)
	}
	byUser := indexPresences(presences)
	complete := make([]robloxservices.UserPresence, 0, len(userIDs))
	for _, userID := range userIDs {
		complete = append(complete, presenceOrOffline(byUser, userID))
	}
	logger.Debug("social presence read", "count", len(complete))
	return complete, nil
}

func (service *Service) GetSocialProfile(ctx context.Context, accountID, userID int64, section SocialProfileSection) (SocialProfile, error) {
	ctx, version, logger, done, err := service.beginSocial(ctx, accountID, "profile")
	if err != nil {
		return SocialProfile{}, err
	}
	defer done()
	account, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return SocialProfile{}, mapRepositoryError(err)
	}
	logger = logger.With("section", section)
	profile := SocialProfile{UserID: userID, Section: section, Relationship: newSocialRelationship(account.RobloxUserID, userID), Unavailable: []string{}}
	if section == SocialSectionRelationship {
		if !profile.Relationship.Self {
			reads := service.relationshipReads(ctx, accountID, version, account.RobloxUserID, userID, &profile.Relationship)
			if profile.Unavailable, err = service.runAccountReads(ctx, accountID, version, "social-profile", logger, reads); err != nil {
				return SocialProfile{}, err
			}
		}
		profile.IsComplete = true
		return profile, nil
	}
	reads := service.socialProfileSectionReads(ctx, logger, accountID, version, userID, section, &profile)
	if reads == nil {
		return SocialProfile{}, socialInputError("This profile section does not exist.")
	}
	unavailable, err := service.runAccountReads(ctx, accountID, version, "social-profile", logger, reads)
	if err != nil {
		return SocialProfile{}, err
	}
	profile.IsComplete = len(unavailable) == 0
	return profile, nil
}

func (service *Service) socialProfileSectionReads(ctx context.Context, logger *slog.Logger, accountID, version, userID int64, section SocialProfileSection, profile *SocialProfile) []accountRead {
	users := service.users
	readsBySection := map[SocialProfileSection]func() []accountRead{
		SocialSectionAbout: func() []accountRead {
			return []accountRead{applyRead("about", func() (robloxservices.UserAbout, error) { return users.About(ctx, accountID, version, userID) }, func(about robloxservices.UserAbout) {
				profile.Description, profile.Verified, profile.Banned = about.Description, about.VerifiedBadge, about.Banned
				if !about.CreatedAt.IsZero() {
					profile.CreatedAtMs = about.CreatedAt.UnixMilli()
				}
			})}
		},
		SocialSectionFriends: func() []accountRead {
			return []accountRead{service.socialCountRead(ctx, accountID, version, userID, "friends", &profile.FriendCount)}
		},
		SocialSectionFollowers: func() []accountRead {
			return []accountRead{service.socialCountRead(ctx, accountID, version, userID, "followers", &profile.FollowerCount)}
		},
		SocialSectionFollowing: func() []accountRead {
			return []accountRead{service.socialCountRead(ctx, accountID, version, userID, "followings", &profile.FollowingCount)}
		},
		SocialSectionPreviousNames: func() []accountRead {
			return []accountRead{applyRead("previousNames", func() ([]string, error) {
				names, err := users.PreviousNames(ctx, accountID, version, userID)
				return names, ignoreTerminatedUser(err)
			}, func(names []string) { profile.PreviousNames = names })}
		},
		SocialSectionPrimaryGroup: func() []accountRead {
			return []accountRead{applyRead("primaryGroup", func() (*robloxservices.UserPrimaryGroup, error) {
				return users.PrimaryGroup(ctx, accountID, version, userID)
			}, func(group *robloxservices.UserPrimaryGroup) {
				profile.PrimaryGroup = group
			})}
		},
		SocialSectionRobloxBadges: func() []accountRead {
			return []accountRead{applyRead("robloxBadges", func() ([]robloxservices.RobloxBadge, error) {
				return users.RobloxBadges(ctx, accountID, version, userID)
			}, func(badges []robloxservices.RobloxBadge) { profile.RobloxBadges = badges })}
		},
		SocialSectionSocialLinks: func() []accountRead {
			return []accountRead{applyRead("socialLinks", func() ([]robloxservices.UserSocialLink, error) {
				return users.UserSocialLinks(ctx, accountID, version, userID)
			}, func(links []robloxservices.UserSocialLink) { profile.SocialLinks = links })}
		},
		SocialSectionPremium: func() []accountRead {
			return []accountRead{applyRead("premium", func() (*bool, error) { return users.Premium(ctx, accountID, version, userID) }, func(premium *bool) { profile.Premium = premium })}
		},
		SocialSectionInventory: func() []accountRead {
			return []accountRead{applyRead("inventory", func() (*bool, error) {
				visible, err := users.InventoryVisible(ctx, accountID, version, userID)
				return visible, ignoreTerminatedUser(err)
			}, func(visible *bool) { profile.InventoryVisible = visible })}
		},
		SocialSectionCommunities: func() []accountRead {
			return []accountRead{applyRead("communities", func() (int64, error) { return users.CommunityCount(ctx, accountID, version, userID) }, func(count int64) {
				profile.CommunityCount = &count
			})}
		},
		SocialSectionCollectibles: func() []accountRead {
			return []accountRead{applyRead("collectibles", func() (*robloxservices.UserCollectibles, error) {
				collectibles, err := users.Collectibles(ctx, accountID, version, userID)
				// Roblox refuses to list a private inventory, which is not a failure.
				if remote, ok := errors.AsType[*roblox.Error](err); ok && remote.Status == http.StatusForbidden {
					return nil, nil
				}
				return &collectibles, ignoreTerminatedUser(err)
			}, func(collectibles *robloxservices.UserCollectibles) {
				if collectibles != nil {
					profile.CollectibleCount, profile.CollectibleRecentAveragePriceTotal = &collectibles.Count, &collectibles.RecentAveragePriceTotal
				}
			})}
		},
		SocialSectionTrading: func() []accountRead {
			if profile.Relationship.Self {
				return []accountRead{}
			}
			return []accountRead{applyRead("trading", func() (robloxservices.UserTradeEligibility, error) {
				return users.CanTradeWith(ctx, accountID, version, userID)
			}, func(eligibility robloxservices.UserTradeEligibility) {
				profile.TradeEligibility = &TradeEligibility{CanTrade: eligibility.CanTrade, Eligibility: eligibility.Eligibility}
			})}
		},
		SocialSectionConnections: func() []accountRead {
			return []accountRead{{"connections", func() error {
				var readError error
				profile.SavedAccountID, profile.Connections, readError = service.friendConnections(ctx, logger, accountID, userID)
				return readError
			}}}
		},
	}
	sectionReads, ok := readsBySection[section]
	if !ok {
		return nil
	}
	return sectionReads()
}

// GetSocialRelationship reads only how the account relates to the user, for
// places that do not need the whole profile.
func (service *Service) GetSocialRelationship(ctx context.Context, accountID, userID int64) (SocialRelationshipSnapshot, error) {
	ctx, version, logger, done, err := service.beginSocial(ctx, accountID, "relationship")
	if err != nil {
		return SocialRelationshipSnapshot{}, err
	}
	defer done()
	account, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return SocialRelationshipSnapshot{}, mapRepositoryError(err)
	}
	snapshot := SocialRelationshipSnapshot{Relationship: newSocialRelationship(account.RobloxUserID, userID)}
	if snapshot.Relationship.Self {
		return snapshot, nil
	}
	reads := service.relationshipReads(ctx, accountID, version, account.RobloxUserID, userID, &snapshot.Relationship)
	if snapshot.Unavailable, err = service.runAccountReads(ctx, accountID, version, "social-relationship", logger, reads); err != nil {
		return SocialRelationshipSnapshot{}, err
	}
	return snapshot, nil
}

func newSocialRelationship(selfID, userID int64) SocialRelationship {
	return SocialRelationship{Self: userID == selfID, Friend: robloxservices.FriendStatusNone, Trusted: robloxservices.TrustedStatusNone}
}

func (service *Service) relationshipReads(ctx context.Context, accountID, version, selfID, userID int64, relationship *SocialRelationship) []accountRead {
	return []accountRead{
		{"friendship", func() error {
			var readError error
			relationship.Friend, readError = service.users.FriendStatus(ctx, accountID, version, selfID, userID)
			return readError
		}},
		{"trusted", func() error {
			var readError error
			relationship.Trusted, readError = service.users.TrustedStatus(ctx, accountID, version, userID)
			return readError
		}},
		{"follow", func() error {
			state, readError := service.users.FollowState(ctx, accountID, version, userID)
			relationship.Following, relationship.FollowedBy = state.Following, state.FollowedBy
			return readError
		}},
		{"nickname", func() error {
			nicknames, readError := service.users.Nicknames(ctx, accountID, version, []int64{userID})
			relationship.Nickname = nicknames[userID]
			return readError
		}},
	}
}

func (service *Service) Follow(ctx context.Context, accountID, userID int64) error {
	return service.editAccount(ctx, accountID, "follow", "", func(ctx context.Context, version int64) error {
		return service.users.Follow(ctx, accountID, version, userID)
	})
}

func (service *Service) Unfollow(ctx context.Context, accountID, userID int64) error {
	return service.editAccount(ctx, accountID, "unfollow", "", func(ctx context.Context, version int64) error {
		return service.users.Unfollow(ctx, accountID, version, userID)
	})
}

// SetNickname saves the private name the account shows for a friend, and
// returns the saved name. An empty name removes it.
func (service *Service) SetNickname(ctx context.Context, accountID, userID int64, nickname string) (string, error) {
	nickname = strings.TrimSpace(nickname)
	if utf8.RuneCountInString(nickname) > maxNicknameLength || !utf8.ValidString(nickname) {
		return "", socialInputError(fmt.Sprintf("Shorten the nickname to %d characters or fewer.", maxNicknameLength))
	}
	err := service.editAccount(ctx, accountID, "nickname-update", "", func(ctx context.Context, version int64) error {
		return service.users.SetNickname(ctx, accountID, version, userID, nickname)
	})
	return nickname, err
}

// friendConnections reads how the other saved accounts are connected to the
// user. Each status read uses that account's own session.
func (service *Service) friendConnections(ctx context.Context, logger *slog.Logger, viewerID, userID int64) (int64, []FriendConnection, error) {
	views, err := service.savedAccounts(ctx)
	if err != nil {
		return 0, nil, err
	}
	var savedID int64
	candidates := make([]accounts.AccountView, 0, len(views))
	for _, view := range views {
		switch {
		case view.RobloxUserID == userID:
			savedID = view.ID
		case view.ID != viewerID && view.State != accounts.StateReauthRequired && view.State != accounts.StateDisabled && view.State != accounts.StateModerated:
			candidates = append(candidates, view)
		}
	}
	states, failures := service.friendConnectionStates(ctx, candidates, userID)
	connections := []FriendConnection{}
	failed := 0
	for index := range candidates {
		if failures[index] != nil {
			failed++
			continue
		}
		if state := states[index]; state.isConnected() {
			connections = append(connections, state)
		}
	}
	if failed > 0 {
		logger.Debug("friend connections incomplete", "accounts", len(candidates), "failed", failed)
	}
	if failed > 0 && failed == len(candidates) {
		return savedID, nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "friend-connections", Message: "Your other accounts could not be checked.", Cause: failures[0]}
	}
	return savedID, connections, nil
}

// friendConnectionStates reads how each saved account relates to the user.
// Each read uses that account's own session, because Roblox answers
// relationship requests only for the signed-in user.
func (service *Service) friendConnectionStates(ctx context.Context, views []accounts.AccountView, userID int64) ([]FriendConnection, []error) {
	connections := make([]FriendConnection, len(views))
	failures := make([]error, len(views))
	jobs := make(chan int, len(views))
	for index := range views {
		jobs <- index
	}
	close(jobs)
	var workers sync.WaitGroup
	for range min(friendConnectionWorkers, len(views)) {
		workers.Go(func() {
			for index := range jobs {
				connections[index], failures[index] = service.friendConnection(ctx, views[index], userID)
			}
		})
	}
	workers.Wait()
	return connections, failures
}

func (connection FriendConnection) isConnected() bool {
	return connection.Friend != robloxservices.FriendStatusNone || connection.Trusted != robloxservices.TrustedStatusNone || connection.Following || connection.FollowedBy
}

func (service *Service) friendConnection(ctx context.Context, view accounts.AccountView, userID int64) (FriendConnection, error) {
	connection := FriendConnection{AccountID: view.ID, Username: view.Username, DisplayName: cmp.Or(view.DisplayName, view.Username)}
	var err error
	if connection.Friend, err = service.users.FriendStatus(ctx, view.ID, 0, view.RobloxUserID, userID); err != nil {
		return FriendConnection{}, err
	}
	if connection.Trusted, err = service.users.TrustedStatus(ctx, view.ID, 0, userID); err != nil {
		return FriendConnection{}, err
	}
	follow, err := service.users.FollowState(ctx, view.ID, 0, userID)
	if err != nil {
		return FriendConnection{}, err
	}
	connection.Following, connection.FollowedBy = follow.Following, follow.FollowedBy
	return connection, nil
}

func (service *Service) AcceptFriendRequest(ctx context.Context, accountID, userID int64) error {
	return service.editAccount(ctx, accountID, "friend-request-accept", "", func(ctx context.Context, version int64) error {
		return service.users.AcceptFriendRequest(ctx, accountID, version, userID)
	})
}

func (service *Service) DeclineFriendRequest(ctx context.Context, accountID, userID int64) error {
	return service.editAccount(ctx, accountID, "friend-request-decline", "", func(ctx context.Context, version int64) error {
		return service.users.DeclineFriendRequest(ctx, accountID, version, userID)
	})
}

func (service *Service) Unfriend(ctx context.Context, accountID, userID int64) error {
	return service.editAccount(ctx, accountID, "unfriend", "", func(ctx context.Context, version int64) error {
		return service.users.Unfriend(ctx, accountID, version, userID)
	})
}

func (service *Service) SendFriendRequest(ctx context.Context, accountID, userID int64) error {
	return service.editAccount(ctx, accountID, "friend-request-send", "Roblox did not allow this friend request.", func(ctx context.Context, version int64) error {
		return service.users.SendFriendRequest(ctx, accountID, version, userID)
	})
}

func (service *Service) SendTrustedFriendRequest(ctx context.Context, accountID, userID int64) error {
	return service.editAccount(ctx, accountID, "trusted-friend-request-send", "Roblox did not allow this trusted friend request.", func(ctx context.Context, version int64) error {
		return service.users.SendTrustedFriendRequest(ctx, accountID, version, userID)
	})
}

func (service *Service) AcceptTrustedFriendRequest(ctx context.Context, accountID, userID int64) error {
	return service.editAccount(ctx, accountID, "trusted-friend-request-accept", "Roblox did not allow this trusted friend request.", func(ctx context.Context, version int64) error {
		return service.users.AcceptTrustedFriendRequest(ctx, accountID, version, userID)
	})
}

func (service *Service) DeclineTrustedFriendRequest(ctx context.Context, accountID, userID int64) error {
	return service.editAccount(ctx, accountID, "trusted-friend-request-decline", "", func(ctx context.Context, version int64) error {
		return service.users.DeclineTrustedFriendRequest(ctx, accountID, version, userID)
	})
}

// RemoveTrustedFriend ends a trusted friendship, or cancels a sent trusted
// friend request.
func (service *Service) RemoveTrustedFriend(ctx context.Context, accountID, userID int64) error {
	return service.editAccount(ctx, accountID, "trusted-friend-remove", "", func(ctx context.Context, version int64) error {
		return service.users.RemoveTrustedFriend(ctx, accountID, version, userID)
	})
}

func (service *Service) beginSocial(parent context.Context, accountID int64, operation string) (context.Context, int64, *slog.Logger, func(), error) {
	ctx, version, done, err := service.beginAccountSession(parent, accountID, "social")
	if err != nil {
		return nil, 0, nil, nil, err
	}
	logger := service.logs.Module("application.social").With("operation_id", logging.NewOperationID("social"), "operation", operation, "account_id", accountID)
	return ctx, version, logger, done, nil
}

func (service *Service) headshotImages(ctx context.Context, userIDs []int64) (map[int64]string, error) {
	headshots, err := service.thumbnails.AvatarHeadshots(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	images := make(map[int64]string, len(headshots))
	for _, headshot := range headshots {
		images[headshot.RobloxUserID] = headshot.ImageURL
	}
	return images, nil
}

func friendPresenceRank(presence *robloxservices.UserPresence) int {
	if presence == nil {
		return 3
	}
	switch presence.UserPresenceType {
	case robloxservices.PresenceTypeInGame:
		return 0
	case robloxservices.PresenceTypeInStudio:
		return 1
	case robloxservices.PresenceTypeOnline:
		return 2
	default:
		return 3
	}
}

// ignoreTerminatedUser hides the 400 that Roblox returns for the username
// history and inventory of a terminated user, which have nothing to show.
// Verified against a terminated user on 2026-11-09.
func ignoreTerminatedUser(err error) error {
	if remote, ok := errors.AsType[*roblox.Error](err); ok && remote.Status == http.StatusBadRequest {
		return nil
	}
	return err
}

func socialFailure(logger *slog.Logger, err error) error {
	return operationFailure(context.Background(), logger, err, "social", "The request failed. Try again.")
}

func socialInputError(message string) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "social", Message: message}
}

const maxSocialRemovalUsers = 200

// SocialRemoval reports a bulk removal. Removed lists the users that are no
// longer in the list. FailureMessage explains the first failure.
type SocialRemoval struct {
	Removed        []int64 `json:"removed"`
	FailedCount    int     `json:"failedCount"`
	FailureMessage string  `json:"failureMessage"`
}

// RemoveSocialConnections unfriends, unfollows, or declines the requests of
// several users of the account's own list, one at a time.
func (service *Service) RemoveSocialConnections(ctx context.Context, accountID int64, list SocialList, userIDs []int64) (SocialRemoval, error) {
	if len(userIDs) == 0 || len(userIDs) > maxSocialRemovalUsers {
		return SocialRemoval{}, socialInputError(fmt.Sprintf("Select 1 to %d people.", maxSocialRemovalUsers))
	}
	var remove func(context.Context, int64) error
	switch list {
	case SocialFriends:
		remove = func(ctx context.Context, userID int64) error { return service.Unfriend(ctx, accountID, userID) }
	case SocialFollowing:
		remove = func(ctx context.Context, userID int64) error { return service.Unfollow(ctx, accountID, userID) }
	case SocialRequests:
		remove = func(ctx context.Context, userID int64) error {
			return service.DeclineFriendRequest(ctx, accountID, userID)
		}
	default:
		return SocialRemoval{}, socialInputError("Choose friends, following, or requests.")
	}
	removal := SocialRemoval{Removed: []int64{}}
	for _, userID := range slices.Compact(slices.Sorted(slices.Values(userIDs))) {
		if ctx.Err() != nil {
			removal.FailedCount++
			continue
		}
		if err := remove(ctx, userID); err != nil {
			removal.FailedCount++
			if removal.FailureMessage == "" {
				removal.FailureMessage = err.Error()
			}
			continue
		}
		removal.Removed = append(removal.Removed, userID)
	}
	service.logs.Module("application.social").Info("social connections removed", "operation", "social-bulk-remove", "account_id", accountID, "list", list, "removed", len(removal.Removed), "failed", removal.FailedCount)
	return removal, nil
}

// FindPlayerByUsername returns the user with exactly this username, or nil
// when Roblox has no such user.
func (service *Service) FindPlayerByUsername(ctx context.Context, username string) (*SocialUser, error) {
	username = strings.TrimPrefix(strings.TrimSpace(username), "@")
	if !playerUsername.MatchString(username) {
		return nil, nil
	}
	userID, err := service.users.UserIDFromUsername(ctx, username)
	if errors.Is(err, robloxservices.ErrUsernameNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	users, err := service.users.Names(ctx, []int64{userID})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 || users[0].Username == "" {
		return nil, nil
	}
	person := SocialUser{UserID: userID, Username: users[0].Username, DisplayName: cmp.Or(users[0].DisplayName, users[0].Username)}
	if images, err := service.headshotImages(ctx, []int64{userID}); err == nil {
		person.ImageURL = images[userID]
	}
	return &person, nil
}
