package appservice

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

const (
	maxSocialCursorLength   = 4096
	minPlayerSearchLength   = 3
	maxPlayerSearchLength   = 50
	maxNicknameLength       = 20
	friendConnectionWorkers = 4
)

var (
	socialOperationID atomic.Uint64
	playerIDSearch    = regexp.MustCompile(`^(?i)id:\s*(\d{1,18})$`)
)

type SocialList string

const (
	SocialFriends   SocialList = "friends"
	SocialFollowers SocialList = "followers"
	SocialFollowing SocialList = "following"
	SocialRequests  SocialList = "requests"
)

type FriendRequestInfo struct {
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
	Request     *FriendRequestInfo           `json:"request"`
}

// SocialRelationship describes the user as the browsing account sees them.
type SocialRelationship struct {
	Self       bool                        `json:"self"`
	Friend     robloxservices.FriendStatus `json:"friend"`
	Trusted    bool                        `json:"trusted"`
	Following  bool                        `json:"following"`
	FollowedBy bool                        `json:"followedBy"`
	Nickname   string                      `json:"nickname"`
}

type SocialPage struct {
	People       []SocialUser `json:"people"`
	NextCursor   string       `json:"nextCursor"`
	RequestCount *int64       `json:"requestCount"`
	Unavailable  []string     `json:"unavailable"`
}

type FriendConnection struct {
	AccountID   int64                       `json:"accountId"`
	Username    string                      `json:"username"`
	DisplayName string                      `json:"displayName"`
	Status      robloxservices.FriendStatus `json:"status"`
}

type SocialProfile struct {
	UserID           int64                            `json:"userId"`
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
	RobloxBadges     []string                         `json:"robloxBadges"`
	SocialLinks      []robloxservices.UserSocialLink  `json:"socialLinks"`
	SavedAccountID   int64                            `json:"savedAccountId"`
	Connections      []FriendConnection               `json:"connections"`
	Relationship     SocialRelationship               `json:"relationship"`
	Unavailable      []string                         `json:"unavailable"`
}

type socialEntries struct {
	ids        []int64
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
	account, err := service.repo.GetView(ctx, accountID)
	if err != nil {
		return SocialPage{}, mapRepositoryError(err)
	}
	own := userID == account.RobloxUserID
	entries, err := service.socialEntries(ctx, accountID, version, userID, own, list, cursor)
	if err != nil {
		return SocialPage{}, socialFailure(logger, err)
	}
	page, err := service.socialPage(ctx, accountID, version, logger, entries, list != SocialRequests, own && (list == SocialFriends || list == SocialRequests))
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
		entries.ids, err = service.playerByID(ctx, match[1])
	} else {
		entries.ids, entries.nextCursor, err = service.users.SearchUsers(ctx, accountID, version, keyword, cursor)
	}
	if err != nil {
		return SocialPage{}, socialFailure(logger, err)
	}
	return service.socialPage(ctx, accountID, version, logger, entries, true, false)
}

// playerByID returns the user ID when Roblox knows the user, or no IDs.
func (service *Service) playerByID(ctx context.Context, value string) ([]int64, error) {
	userID, err := strconv.ParseInt(value, 10, 64)
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

func (service *Service) socialPage(ctx context.Context, accountID, version int64, logger *slog.Logger, entries socialEntries, withPresence, withRequestCount bool) (SocialPage, error) {
	users, err := service.users.Names(ctx, entries.ids)
	if err != nil {
		return SocialPage{}, socialFailure(logger, err)
	}
	page := SocialPage{NextCursor: entries.nextCursor}
	var presences []robloxservices.UserPresence
	var images, nicknames map[int64]string
	reads := []accountRead{{"avatars", func() error {
		var readErr error
		images, readErr = service.headshotImages(ctx, entries.ids)
		return readErr
	}}, {"nicknames", func() error {
		var readErr error
		nicknames, readErr = service.users.Nicknames(ctx, accountID, version, entries.ids)
		return readErr
	}}}
	if withPresence {
		reads = append(reads, accountRead{"presence", func() error {
			var readErr error
			presences, readErr = service.presence.UserPresences(ctx, accountID, version, entries.ids)
			return readErr
		}})
	}
	if withRequestCount {
		reads = append(reads, accountRead{"requestCount", func() error {
			var readErr error
			page.RequestCount, readErr = service.users.FriendRequestCount(ctx, accountID, version)
			return readErr
		}})
	}
	if page.Unavailable, err = service.runAccountReads(ctx, accountID, version, "social", logger, reads); err != nil {
		return SocialPage{}, err
	}
	page.People = socialPeople(users, entries.requests, presences, withPresence)
	for index := range page.People {
		person := &page.People[index]
		person.ImageURL, person.Nickname = images[person.UserID], nicknames[person.UserID]
	}
	logger.Debug("social list read", "count", len(page.People), "failed_sections", len(page.Unavailable))
	return page, nil
}

func (service *Service) socialEntries(ctx context.Context, accountID, version, userID int64, own bool, list SocialList, cursor string) (socialEntries, error) {
	var entries socialEntries
	var err error
	switch list {
	case SocialFriends:
		entries.ids, err = service.users.FriendIDs(ctx, accountID, version, userID)
	case SocialFollowers:
		entries.ids, entries.nextCursor, err = service.users.Follows(ctx, accountID, version, userID, "followers", cursor)
	case SocialFollowing:
		entries.ids, entries.nextCursor, err = service.users.Follows(ctx, accountID, version, userID, "followings", cursor)
	case SocialRequests:
		if !own {
			return entries, socialInputError("Friend requests are only available for the selected account.")
		}
		entries.requests, entries.nextCursor, err = service.users.FriendRequests(ctx, accountID, version, cursor)
		for _, request := range entries.requests {
			entries.ids = append(entries.ids, request.UserID)
		}
	default:
		return entries, socialInputError("Choose friends, followers, following, or requests.")
	}
	return entries, err
}

func socialPeople(users []robloxservices.UserView, requests []robloxservices.FriendRequest, presences []robloxservices.UserPresence, withPresence bool) []SocialUser {
	presenceByUser := make(map[int64]robloxservices.UserPresence, len(presences))
	for _, presence := range presences {
		presenceByUser[presence.UserID] = presence
	}
	people := make([]SocialUser, 0, len(users))
	for index, user := range users {
		person := SocialUser{UserID: user.UserID, Username: user.Username, DisplayName: cmp.Or(user.DisplayName, user.Username)}
		if withPresence {
			presence, found := presenceByUser[user.UserID]
			if !found {
				presence = robloxservices.UserPresence{UserID: user.UserID, UserPresenceType: robloxservices.PresenceTypeOffline}
			}
			person.Presence = &presence
		}
		if index < len(requests) {
			request := requests[index]
			person.Request = &FriendRequestInfo{Origin: request.OriginSourceType, MutualFriends: request.MutualFriends}
			if !request.SentAt.IsZero() {
				person.Request.SentAtMs = request.SentAt.UnixMilli()
			}
		}
		people = append(people, person)
	}
	return people
}

func compareFriends(left, right SocialUser) int {
	return cmp.Or(
		cmp.Compare(friendPresenceRank(left.Presence), friendPresenceRank(right.Presence)),
		strings.Compare(strings.ToLower(left.DisplayName), strings.ToLower(right.DisplayName)),
	)
}

func (service *Service) GetSocialProfile(ctx context.Context, accountID, userID int64) (SocialProfile, error) {
	ctx, version, logger, done, err := service.beginSocial(ctx, accountID, "profile")
	if err != nil {
		return SocialProfile{}, err
	}
	defer done()
	account, err := service.repo.GetView(ctx, accountID)
	if err != nil {
		return SocialProfile{}, mapRepositoryError(err)
	}
	profile := SocialProfile{UserID: userID, Relationship: SocialRelationship{Self: userID == account.RobloxUserID, Friend: robloxservices.FriendStatusNone}}
	reads := []accountRead{
		{"about", func() error {
			about, readErr := service.users.About(ctx, accountID, version, userID)
			if readErr == nil {
				profile.Description, profile.Verified, profile.Banned = about.Description, about.VerifiedBadge, about.Banned
				if !about.CreatedAt.IsZero() {
					profile.CreatedAtMs = about.CreatedAt.UnixMilli()
				}
			}
			return readErr
		}},
		{"friends", func() error {
			var readErr error
			profile.FriendCount, readErr = service.users.SocialCount(ctx, accountID, version, userID, "friends")
			return readErr
		}},
		{"followers", func() error {
			var readErr error
			profile.FollowerCount, readErr = service.users.SocialCount(ctx, accountID, version, userID, "followers")
			return readErr
		}},
		{"following", func() error {
			var readErr error
			profile.FollowingCount, readErr = service.users.SocialCount(ctx, accountID, version, userID, "followings")
			return readErr
		}},
		{"previousNames", func() error {
			var readErr error
			profile.PreviousNames, readErr = service.users.PreviousNames(ctx, accountID, version, userID)
			return readErr
		}},
		{"primaryGroup", func() error {
			var readErr error
			profile.PrimaryGroup, readErr = service.users.PrimaryGroup(ctx, accountID, version, userID)
			return readErr
		}},
		{"robloxBadges", func() error {
			var readErr error
			profile.RobloxBadges, readErr = service.users.RobloxBadges(ctx, accountID, version, userID)
			return readErr
		}},
		{"socialLinks", func() error {
			var readErr error
			profile.SocialLinks, readErr = service.users.UserSocialLinks(ctx, accountID, version, userID)
			return readErr
		}},
		{"premium", func() error {
			var readErr error
			profile.Premium, readErr = service.users.Premium(ctx, accountID, version, userID)
			return readErr
		}},
		{"inventory", func() error {
			var readErr error
			profile.InventoryVisible, readErr = service.users.InventoryVisible(ctx, accountID, version, userID)
			return readErr
		}},
		{"connections", func() error {
			var readErr error
			profile.SavedAccountID, profile.Connections, readErr = service.friendConnections(ctx, logger, accountID, userID)
			return readErr
		}},
	}
	if !profile.Relationship.Self {
		reads = append(reads, service.relationshipReads(ctx, accountID, version, account.RobloxUserID, userID, &profile.Relationship)...)
	}
	if profile.Unavailable, err = service.runAccountReads(ctx, accountID, version, "social-profile", logger, reads); err != nil {
		return SocialProfile{}, err
	}
	return profile, nil
}

func (service *Service) relationshipReads(ctx context.Context, accountID, version, selfID, userID int64, relationship *SocialRelationship) []accountRead {
	return []accountRead{
		{"friendship", func() error {
			var readErr error
			relationship.Friend, readErr = service.users.FriendStatus(ctx, accountID, version, selfID, userID)
			return readErr
		}},
		{"trusted", func() error {
			var readErr error
			relationship.Trusted, readErr = service.users.TrustedFriend(ctx, accountID, version, userID)
			return readErr
		}},
		{"follow", func() error {
			state, readErr := service.users.FollowState(ctx, accountID, version, userID)
			relationship.Following, relationship.FollowedBy = state.Following, state.FollowedBy
			return readErr
		}},
		{"nickname", func() error {
			nicknames, readErr := service.users.Nicknames(ctx, accountID, version, []int64{userID})
			relationship.Nickname = nicknames[userID]
			return readErr
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
		case view.ID != viewerID && view.State != accounts.StateReauthRequired && view.State != accounts.StateDisabled:
			candidates = append(candidates, view)
		}
	}
	statuses, failures := service.friendStatuses(ctx, candidates, userID)
	connections := []FriendConnection{}
	failed := 0
	for index, view := range candidates {
		if failures[index] != nil {
			failed++
			continue
		}
		if statuses[index] != robloxservices.FriendStatusNone {
			connections = append(connections, FriendConnection{AccountID: view.ID, Username: view.Username, DisplayName: cmp.Or(view.DisplayName, view.Username), Status: statuses[index]})
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

func (service *Service) friendStatuses(ctx context.Context, views []accounts.AccountView, userID int64) ([]robloxservices.FriendStatus, []error) {
	statuses := make([]robloxservices.FriendStatus, len(views))
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
				view := views[index]
				statuses[index], failures[index] = service.users.FriendStatus(ctx, view.ID, 0, view.RobloxUserID, userID)
			}
		})
	}
	workers.Wait()
	return statuses, failures
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

func (service *Service) beginSocial(parent context.Context, accountID int64, operation string) (context.Context, int64, *slog.Logger, func(), error) {
	ctx, version, done, err := service.beginAccountSession(parent, accountID, "social")
	if err != nil {
		return nil, 0, nil, nil, err
	}
	logger := service.logs.Module("application.social").With("operation_id", fmt.Sprintf("social-%d", socialOperationID.Add(1)), "operation", operation, "account_id", accountID)
	return ctx, version, logger, done, nil
}

func (service *Service) headshotImages(ctx context.Context, userIDs []int64) (map[int64]string, error) {
	headshots, err := service.avatars.AvatarHeadshots(ctx, userIDs)
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

func socialFailure(logger *slog.Logger, err error) error {
	var remote *roblox.Error
	if !errors.As(err, &remote) {
		logger.Warn("social operation failed", "error", err)
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "social", Message: "The request failed. Try again.", Cause: err}
	}
	logger.Warn("social operation failed", "endpoint", remote.Endpoint, "error_kind", remote.Kind, "status", remote.Status, "roblox_code", remote.RobloxCode)
	return err
}

func socialInputError(message string) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "social", Message: message}
}
