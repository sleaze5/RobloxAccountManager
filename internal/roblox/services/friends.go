package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	friendsPageSize        = 50
	maxFriendsPages        = 40
	friendRequestsPageSize = 50
	followsPageSize        = 50
	userSearchPageSize     = 25
	nicknameBatchSize      = 100
)

type FriendRequest struct {
	UserID           int64
	SentAt           time.Time
	OriginSourceType string
	MutualFriends    []string
}

// Shape verified against signed-in responses on 2026-11-01. The older
// /friends endpoint returns empty names, so names come from Users.Names.
func (service *Users) FriendIDs(ctx context.Context, accountID, version, userID int64) ([]int64, error) {
	return service.findFriendIDs(ctx, accountID, version, userID, "friends", "")
}

// TrustedFriendIDs reads the account's own trusted friends. Roblox rejects the
// request for any other user. Shape verified on 2026-11-09.
func (service *Users) TrustedFriendIDs(ctx context.Context, accountID, version, selfID int64) ([]int64, error) {
	return service.findFriendIDs(ctx, accountID, version, selfID, "trusted-friends", "FindTrustedFriends")
}

func (service *Users) findFriendIDs(ctx context.Context, accountID, version, userID int64, endpoint, findFriendsType string) ([]int64, error) {
	ids := []int64{}
	seen, seenCursors := map[int64]bool{}, map[string]bool{}
	cursor := ""
	for range maxFriendsPages {
		var payload struct {
			PageItems *[]struct {
				ID int64 `json:"id"`
			} `json:"PageItems"`
			NextCursor *string `json:"NextCursor"`
		}
		query := url.Values{"limit": {strconv.Itoa(friendsPageSize)}}
		if cursor != "" {
			query.Set("cursor", cursor)
		}
		if findFriendsType != "" {
			query.Set("findFriendsType", findFriendsType)
		}
		address := "https://friends.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/friends/find?" + query.Encode()
		if err := service.readAccountDetails(ctx, accountID, version, endpoint, address, &payload); err != nil {
			return nil, err
		}
		if payload.PageItems == nil {
			return nil, invalidProfileDetails(endpoint)
		}
		for _, friend := range *payload.PageItems {
			if friend.ID > 0 && !seen[friend.ID] {
				seen[friend.ID] = true
				ids = append(ids, friend.ID)
			}
		}
		if payload.NextCursor == nil || *payload.NextCursor == "" {
			return ids, nil
		}
		cursor = *payload.NextCursor
		if len(cursor) > 4096 || seenCursors[cursor] {
			return nil, invalidProfileDetails(endpoint)
		}
		seenCursors[cursor] = true
	}
	return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Roblox returned too many friends to list."}
}

func (service *Users) FriendRequestCount(ctx context.Context, accountID, version int64) (*int64, error) {
	var payload struct {
		Count *int64 `json:"count"`
	}
	err := service.readAccountDetails(ctx, accountID, version, "friend-request-count", "https://friends.roblox.com/v1/user/friend-requests/count", &payload)
	if err == nil && (payload.Count == nil || *payload.Count < 0) {
		err = invalidProfileDetails("friend-request-count")
	}
	return payload.Count, err
}

func (service *Users) FriendRequests(ctx context.Context, accountID, version int64, cursor string) ([]FriendRequest, string, error) {
	var payload struct {
		Data *[]struct {
			ID            int64 `json:"id"`
			FriendRequest struct {
				SentAt           string `json:"sentAt"`
				OriginSourceType string `json:"originSourceType"`
			} `json:"friendRequest"`
			MutualFriendsList []string `json:"mutualFriendsList"`
		} `json:"data"`
		NextPageCursor *string `json:"nextPageCursor"`
	}
	query := url.Values{"limit": {strconv.Itoa(friendRequestsPageSize)}}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	if err := service.readAccountDetails(ctx, accountID, version, "friend-requests", "https://friends.roblox.com/v1/my/friends/requests?"+query.Encode(), &payload); err != nil {
		return nil, "", err
	}
	if payload.Data == nil || len(*payload.Data) > friendRequestsPageSize {
		return nil, "", invalidProfileDetails("friend-requests")
	}
	requests := make([]FriendRequest, 0, len(*payload.Data))
	for _, requester := range *payload.Data {
		if requester.ID <= 0 {
			return nil, "", invalidProfileDetails("friend-requests")
		}
		request := FriendRequest{UserID: requester.ID, OriginSourceType: strings.TrimSpace(requester.FriendRequest.OriginSourceType), MutualFriends: []string{}}
		if sentAt, err := time.Parse(time.RFC3339, strings.TrimSpace(requester.FriendRequest.SentAt)); err == nil {
			request.SentAt = sentAt.UTC()
		}
		for _, name := range requester.MutualFriendsList {
			if name = strings.TrimSpace(name); name != "" && len(name) <= 64 {
				request.MutualFriends = append(request.MutualFriends, name)
			}
		}
		requests = append(requests, request)
	}
	next := ""
	if payload.NextPageCursor != nil {
		next = *payload.NextPageCursor
	}
	return requests, next, nil
}

func (service *Users) AcceptFriendRequest(ctx context.Context, accountID, version, userID int64) error {
	return service.friendAction(ctx, accountID, version, userID, "friend-request-accept", "accept-friend-request")
}

func (service *Users) DeclineFriendRequest(ctx context.Context, accountID, version, userID int64) error {
	return service.friendAction(ctx, accountID, version, userID, "friend-request-decline", "decline-friend-request")
}

// Unfriend ends a friendship, or cancels a friend request that the account
// sent. Roblox uses the same request for both, as RoSeal does.
func (service *Users) Unfriend(ctx context.Context, accountID, version, userID int64) error {
	return service.friendAction(ctx, accountID, version, userID, "unfriend", "unfriend")
}

func (service *Users) SendFriendRequest(ctx context.Context, accountID, version, userID int64) error {
	address := "https://friends.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/request-friendship"
	_, err := service.writeAccount(ctx, accountID, version, "friend-request-send", http.MethodPost, address, map[string]int{"friendshipOriginSourceType": 0})
	return err
}

func (service *Users) SendTrustedFriendRequest(ctx context.Context, accountID, version, userID int64) error {
	return service.trustedFriendAction(ctx, accountID, version, userID, "trusted-friend-request-send", "send-trusted-friend-request")
}

func (service *Users) AcceptTrustedFriendRequest(ctx context.Context, accountID, version, userID int64) error {
	return service.trustedFriendAction(ctx, accountID, version, userID, "trusted-friend-request-accept", "accept-trusted-friend-request")
}

// DeclineTrustedFriendRequest ignores a trusted friend request that the user sent.
func (service *Users) DeclineTrustedFriendRequest(ctx context.Context, accountID, version, userID int64) error {
	return service.trustedFriendAction(ctx, accountID, version, userID, "trusted-friend-request-decline", "ignore-trusted-friend-request")
}

// RemoveTrustedFriend ends a trusted friendship, or cancels a trusted friend
// request that the account sent. Roblox uses the same request for both.
func (service *Users) RemoveTrustedFriend(ctx context.Context, accountID, version, userID int64) error {
	return service.trustedFriendAction(ctx, accountID, version, userID, "trusted-friend-remove", "remove-trusted-friend")
}

func (service *Users) trustedFriendAction(ctx context.Context, accountID, version, userID int64, endpoint, action string) error {
	address := "https://friends.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/" + action
	_, err := service.writeAccount(ctx, accountID, version, endpoint, http.MethodPost, address, struct{}{})
	return err
}

func (service *Users) friendAction(ctx context.Context, accountID, version, userID int64, endpoint, action string) error {
	address := "https://friends.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/" + action
	_, err := service.writeAccount(ctx, accountID, version, endpoint, http.MethodPost, address, nil)
	return err
}

type FriendStatus string

const (
	FriendStatusNone            FriendStatus = "none"
	FriendStatusFriends         FriendStatus = "friends"
	FriendStatusRequestSent     FriendStatus = "requestSent"
	FriendStatusRequestReceived FriendStatus = "requestReceived"
)

// FriendStatus reads how the account is connected to another user. Roblox
// only answers for the signed-in user, so selfID must be the account's user.
func (service *Users) FriendStatus(ctx context.Context, accountID, version, selfID, userID int64) (FriendStatus, error) {
	var payload struct {
		Data *[]struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	address := "https://friends.roblox.com/v1/users/" + strconv.FormatInt(selfID, 10) + "/friends/statuses?userIds=" + strconv.FormatInt(userID, 10)
	if err := service.readAccountDetails(ctx, accountID, version, "friend-status", address, &payload); err != nil {
		return "", err
	}
	if payload.Data == nil {
		return "", invalidProfileDetails("friend-status")
	}
	for _, friend := range *payload.Data {
		if friend.ID != userID {
			continue
		}
		switch friend.Status {
		case "Friends":
			return FriendStatusFriends, nil
		case "RequestSent":
			return FriendStatusRequestSent, nil
		case "RequestReceived":
			return FriendStatusRequestReceived, nil
		default:
			return FriendStatusNone, nil
		}
	}
	return "", invalidProfileDetails("friend-status")
}

func (service *Users) RobloxBadges(ctx context.Context, accountID, version, userID int64) ([]string, error) {
	var payload *[]struct {
		Name string `json:"name"`
	}
	address := "https://accountinformation.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/roblox-badges"
	if err := service.readAccountDetails(ctx, accountID, version, "friend-roblox-badges", address, &payload); err != nil {
		return nil, err
	}
	if payload == nil {
		return nil, invalidProfileDetails("friend-roblox-badges")
	}
	names := make([]string, 0, len(*payload))
	for _, badge := range *payload {
		if name := strings.TrimSpace(badge.Name); name != "" && len(name) <= 64 {
			names = append(names, name)
		}
	}
	return names, nil
}

func (service *Users) Premium(ctx context.Context, accountID, version, userID int64) (*bool, error) {
	var isPremium *bool
	address := "https://premiumfeatures.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/validate-membership"
	err := service.readAccountDetails(ctx, accountID, version, "friend-premium", address, &isPremium)
	if err == nil && isPremium == nil {
		err = invalidProfileDetails("friend-premium")
	}
	return isPremium, err
}

func (service *Users) InventoryVisible(ctx context.Context, accountID, version, userID int64) (*bool, error) {
	var payload struct {
		CanView *bool `json:"canView"`
	}
	address := "https://inventory.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/can-view-inventory"
	err := service.readAccountDetails(ctx, accountID, version, "friend-inventory", address, &payload)
	if err == nil && payload.CanView == nil {
		err = invalidProfileDetails("friend-inventory")
	}
	return payload.CanView, err
}

// Follows reads one page of a user's followers or followings, newest first.
func (service *Users) Follows(ctx context.Context, accountID, version, userID int64, counter, cursor string) ([]int64, string, error) {
	switch counter {
	case "followers", "followings":
	default:
		return nil, "", invalidProfileDetails("follows")
	}
	query := url.Values{"limit": {strconv.Itoa(followsPageSize)}, "sortOrder": {"Desc"}}
	address := "https://friends.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/" + counter
	return service.userIDPage(ctx, accountID, version, "social-"+counter, address, query, cursor, followsPageSize)
}

// SearchUsers reads one page of Roblox users whose username or display name
// matches the keyword. Roblox rejects keywords shorter than 3 characters.
func (service *Users) SearchUsers(ctx context.Context, accountID, version int64, keyword, cursor string) ([]int64, string, error) {
	query := url.Values{"keyword": {keyword}, "limit": {strconv.Itoa(userSearchPageSize)}}
	return service.userIDPage(ctx, accountID, version, "user-search", "https://users.roblox.com/v1/users/search", query, cursor, userSearchPageSize)
}

func (service *Users) userIDPage(ctx context.Context, accountID, version int64, endpoint, address string, query url.Values, cursor string, limit int) ([]int64, string, error) {
	var payload struct {
		Data *[]struct {
			ID int64 `json:"id"`
		} `json:"data"`
		NextPageCursor *string `json:"nextPageCursor"`
	}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	if err := service.readAccountDetails(ctx, accountID, version, endpoint, address+"?"+query.Encode(), &payload); err != nil {
		return nil, "", err
	}
	if payload.Data == nil || len(*payload.Data) > limit {
		return nil, "", invalidProfileDetails(endpoint)
	}
	ids := make([]int64, 0, len(*payload.Data))
	for _, user := range *payload.Data {
		if user.ID > 0 {
			ids = append(ids, user.ID)
		}
	}
	next := ""
	if payload.NextPageCursor != nil {
		next = *payload.NextPageCursor
	}
	return ids, next, nil
}

type FollowState struct {
	Following  bool
	FollowedBy bool
}

// FollowState reads whether the account follows the user and whether the user
// follows the account.
func (service *Users) FollowState(ctx context.Context, accountID, version, userID int64) (FollowState, error) {
	var followingResponse struct {
		Followings *[]struct {
			UserID      int64 `json:"userId"`
			IsFollowing bool  `json:"isFollowing"`
			IsFollowed  bool  `json:"isFollowed"`
		} `json:"followings"`
	}
	payload := map[string][]int64{"targetUserIds": {userID}}
	if err := service.postRead(ctx, accountID, version, "follow-state", "https://friends.roblox.com/v1/user/following-exists", payload, &followingResponse); err != nil {
		return FollowState{}, err
	}
	if followingResponse.Followings == nil {
		return FollowState{}, invalidProfileDetails("follow-state")
	}
	for _, following := range *followingResponse.Followings {
		if following.UserID == userID {
			return FollowState{Following: following.IsFollowing, FollowedBy: following.IsFollowed}, nil
		}
	}
	return FollowState{}, invalidProfileDetails("follow-state")
}

type TrustedStatus string

const (
	TrustedStatusNone            TrustedStatus = "none"
	TrustedStatusTrusted         TrustedStatus = "trusted"
	TrustedStatusRequestSent     TrustedStatus = "requestSent"
	TrustedStatusRequestReceived TrustedStatus = "requestReceived"
)

func (service *Users) TrustedStatus(ctx context.Context, accountID, version, userID int64) (TrustedStatus, error) {
	var payload struct {
		Status *string `json:"status"`
	}
	address := "https://friends.roblox.com/v1/my/trusted-friends/" + strconv.FormatInt(userID, 10) + "/status"
	if err := service.readAccountDetails(ctx, accountID, version, "trusted-friend-status", address, &payload); err != nil {
		return "", err
	}
	if payload.Status == nil {
		return "", invalidProfileDetails("trusted-friend-status")
	}
	switch *payload.Status {
	case "TrustedFriends":
		return TrustedStatusTrusted, nil
	case "RequestSent":
		return TrustedStatusRequestSent, nil
	case "RequestReceived":
		return TrustedStatusRequestReceived, nil
	default:
		return TrustedStatusNone, nil
	}
}

// Nicknames reads the private names the account gave to users. Users without
// a nickname are omitted.
func (service *Users) Nicknames(ctx context.Context, accountID, version int64, userIDs []int64) (map[int64]string, error) {
	nicknames := map[int64]string{}
	for batch := range slices.Chunk(userIDs, nicknameBatchSize) {
		var tagsResponse *[]struct {
			TargetUserID  int64  `json:"targetUserId"`
			TargetUserTag string `json:"targetUserTag"`
		}
		payload := map[string][]int64{"targetUserIds": batch}
		if err := service.postRead(ctx, accountID, version, "nicknames", "https://contacts.roblox.com/v1/user/get-tags", payload, &tagsResponse); err != nil {
			return nil, err
		}
		if tagsResponse == nil {
			return nil, invalidProfileDetails("nicknames")
		}
		for _, nickname := range *tagsResponse {
			if tag := strings.TrimSpace(nickname.TargetUserTag); nickname.TargetUserID > 0 && tag != "" {
				nicknames[nickname.TargetUserID] = tag
			}
		}
	}
	return nicknames, nil
}

// SetNickname saves the private name the account shows for the user. An empty
// nickname removes it.
func (service *Users) SetNickname(ctx context.Context, accountID, version, userID int64, nickname string) error {
	response, err := service.writeAccount(ctx, accountID, version, "nickname-update", http.MethodPost, "https://contacts.roblox.com/v1/user/tag", map[string]any{"targetUserId": userID, "userTag": nickname})
	if err != nil {
		return err
	}
	var payload struct {
		Status string `json:"status"`
	}
	if json.Unmarshal(response.Body, &payload) != nil || !strings.EqualFold(payload.Status, "Success") {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "nickname-update", Status: response.Status, Message: "Roblox did not save the nickname. Try a different one."}
	}
	return nil
}

func (service *Users) Follow(ctx context.Context, accountID, version, userID int64) error {
	return service.friendAction(ctx, accountID, version, userID, "follow", "follow")
}

func (service *Users) Unfollow(ctx context.Context, accountID, version, userID int64) error {
	return service.friendAction(ctx, accountID, version, userID, "unfollow", "unfollow")
}

func (service *Users) postRead(ctx context.Context, accountID, version int64, endpoint, address string, payload, result any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "The request could not be encoded.", Cause: err}
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	endpointURL, _ := url.Parse(address)
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: endpoint, AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodPost, URL: endpointURL, Body: body,
		ContentType: "application/json", RequiresCSRF: true, MaxResponseSize: 64 << 10,
		Retry: roblox.RetryPolicy{MaxAttempts: 2, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
	})
	if err != nil {
		return err
	}
	if err := json.Unmarshal(response.Body, result); err != nil {
		return invalidProfileDetails(endpoint)
	}
	return nil
}
