package services

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

// The group API shapes below were verified against signed-in responses on
// 2026-11-19. Roblox calls groups communities on its website, but its APIs
// keep the group names.
const (
	groupMemberPageSize       = 100
	groupSearchPageSize       = 25
	groupGamePageSize         = 50
	groupRelationshipPageSize = 50
	groupNameHistoryPageSize  = 10
	maxGroupNameLength        = 100
	// MaxGroupShoutLength is the limit that the groups metadata API reports
	// as groupStatusMaxLength.
	MaxGroupShoutLength = 255
	// MaxGroupDescriptionLength is the longest description that Roblox saves.
	MaxGroupDescriptionLength   = 1000
	maxGroupSearchKeywordLength = 50
	minGroupSearchKeywordLength = 2
)

// GroupSummary is what a list of groups shows about each group.
type GroupSummary struct {
	GroupID              int64  `json:"groupId"`
	Name                 string `json:"name"`
	Description          string `json:"description"`
	MemberCount          int64  `json:"memberCount"`
	HasVerifiedBadge     bool   `json:"hasVerifiedBadge"`
	IsPublicEntryAllowed bool   `json:"isPublicEntryAllowed"`
	IconURL              string `json:"iconUrl"`
}

// GroupUser is a Roblox user as a group API lists one, such as the owner, a
// member, or the poster of a shout.
type GroupUser struct {
	UserID           int64  `json:"userId"`
	Username         string `json:"username"`
	DisplayName      string `json:"displayName"`
	HasVerifiedBadge bool   `json:"hasVerifiedBadge"`
}

type GroupShout struct {
	Body   string    `json:"body"`
	Poster GroupUser `json:"poster"`
	// UpdatedAtMs is when the shout was posted, or 0 when Roblox leaves it
	// out.
	UpdatedAtMs int64 `json:"updatedAtMs"`
}

type GroupDetails struct {
	Group GroupSummary `json:"group"`
	// Owner is nil for a group that nobody owns.
	Owner    *GroupUser  `json:"owner"`
	Shout    *GroupShout `json:"shout"`
	IsLocked bool        `json:"isLocked"`
}

type GroupRole struct {
	RoleID      int64  `json:"roleId"`
	Name        string `json:"name"`
	Rank        int64  `json:"rank"`
	MemberCount int64  `json:"memberCount"`
}

// UserGroup is a group that a user belongs to, with the user's role.
type UserGroup struct {
	Group GroupSummary `json:"group"`
	Role  GroupRole    `json:"role"`
}

// GroupPermissions are the actions that the account may take in a group.
type GroupPermissions struct {
	CanViewShout          bool `json:"canViewShout"`
	CanPostShout          bool `json:"canPostShout"`
	CanChangeRank         bool `json:"canChangeRank"`
	CanAcceptJoinRequests bool `json:"canAcceptJoinRequests"`
	CanRemoveMembers      bool `json:"canRemoveMembers"`
	CanSpendFunds         bool `json:"canSpendFunds"`
	CanManageGames        bool `json:"canManageGames"`
	// CanConfigure allows editing the description and the other settings.
	CanConfigure bool `json:"canConfigure"`
}

// GroupMembership is how the account belongs to a group. A rank of 0 means
// the account is a guest, not a member.
type GroupMembership struct {
	Role              GroupRole        `json:"role"`
	IsPrimary         bool             `json:"isPrimary"`
	IsPendingJoin     bool             `json:"isPendingJoin"`
	IsOwner           bool             `json:"isOwner"`
	CanViewMemberList bool             `json:"canViewMemberList"`
	Permissions       GroupPermissions `json:"permissions"`
}

type GroupMember struct {
	User GroupUser `json:"user"`
	Role GroupRole `json:"role"`
}

type GroupMemberPage struct {
	Members    []GroupMember `json:"members"`
	NextCursor string        `json:"nextCursor"`
}

type GroupJoinRequest struct {
	User          GroupUser `json:"user"`
	RequestedAtMs int64     `json:"requestedAtMs"`
}

type GroupJoinRequestPage struct {
	Requests   []GroupJoinRequest `json:"requests"`
	NextCursor string             `json:"nextCursor"`
}

type GroupSocialLink struct {
	Type  string `json:"type"`
	URL   string `json:"url"`
	Title string `json:"title"`
}

type GroupNameChange struct {
	Name        string `json:"name"`
	ChangedAtMs int64  `json:"changedAtMs"`
}

type GroupSearchPage struct {
	Groups     []GroupSummary `json:"groups"`
	NextCursor string         `json:"nextCursor"`
}

// GroupGame is a game that a group owns. IsPrivate marks a game that only
// people with access can play, which the public game list of the group
// leaves out.
type GroupGame struct {
	Place       games.Place `json:"place"`
	IsPrivate   bool        `json:"isPrivate"`
	Visits      int64       `json:"visits"`
	UpdatedAtMs int64       `json:"updatedAtMs"`
}

type GroupGamePage struct {
	Games      []GroupGame `json:"games"`
	NextCursor string      `json:"nextCursor"`
}

// GroupRelationship is whether a related group is an ally or an enemy.
type GroupRelationship string

const (
	GroupAllies  GroupRelationship = "allies"
	GroupEnemies GroupRelationship = "enemies"
)

type listedGroupUser struct {
	UserID           int64  `json:"userId"`
	Username         string `json:"username"`
	DisplayName      string `json:"displayName"`
	HasVerifiedBadge bool   `json:"hasVerifiedBadge"`
}

func (listedUser listedGroupUser) user() GroupUser {
	return GroupUser{
		UserID: listedUser.UserID, Username: boundedText(listedUser.Username, maxGroupNameLength),
		DisplayName:      boundedText(listedUser.DisplayName, maxGroupNameLength),
		HasVerifiedBadge: listedUser.HasVerifiedBadge,
	}
}

type listedGroupRole struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Rank        int64  `json:"rank"`
	MemberCount int64  `json:"memberCount"`
}

func (listedRole listedGroupRole) role() GroupRole {
	return GroupRole{RoleID: listedRole.ID, Name: boundedText(listedRole.Name, maxGroupNameLength), Rank: min(max(listedRole.Rank, 0), 255), MemberCount: max(listedRole.MemberCount, 0)}
}

type listedGroup struct {
	ID                 int64            `json:"id"`
	Name               string           `json:"name"`
	Description        string           `json:"description"`
	Owner              *listedGroupUser `json:"owner"`
	Shout              *listedShout     `json:"shout"`
	MemberCount        int64            `json:"memberCount"`
	PublicEntryAllowed bool             `json:"publicEntryAllowed"`
	HasVerifiedBadge   bool             `json:"hasVerifiedBadge"`
	IsLocked           bool             `json:"isLocked"`
}

type listedShout struct {
	Body    string           `json:"body"`
	Poster  *listedGroupUser `json:"poster"`
	Updated string           `json:"updated"`
}

func (group listedGroup) summary() GroupSummary {
	return GroupSummary{
		GroupID: group.ID, Name: boundedText(group.Name, maxGroupNameLength),
		Description: boundedText(group.Description, MaxGroupDescriptionLength),
		MemberCount: max(group.MemberCount, 0), HasVerifiedBadge: group.HasVerifiedBadge,
		IsPublicEntryAllowed: group.PublicEntryAllowed,
	}
}

func (group listedGroup) details() GroupDetails {
	details := GroupDetails{Group: group.summary(), IsLocked: group.IsLocked}
	if group.Owner != nil && group.Owner.UserID > 0 {
		owner := group.Owner.user()
		details.Owner = &owner
	}
	if group.Shout != nil && strings.TrimSpace(group.Shout.Body) != "" {
		details.Shout = &GroupShout{Body: boundedText(group.Shout.Body, MaxGroupShoutLength), UpdatedAtMs: robloxTimeMs(group.Shout.Updated)}
		if group.Shout.Poster != nil {
			details.Shout.Poster = group.Shout.Poster.user()
		}
	}
	return details
}

func groupsAddress(path string, query url.Values) string {
	address := url.URL{Scheme: "https", Host: "groups.roblox.com", Path: path, RawQuery: query.Encode()}
	return address.String()
}

func groupPath(groupID int64, suffix string) string {
	return "/v1/groups/" + strconv.FormatInt(groupID, 10) + suffix
}

func invalidGroupData(endpoint string) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Roblox returned invalid group data."}
}

// readGroupData reads a group response, which can be larger than a profile
// detail, such as a list of groups with their descriptions.
func (service *Users) readGroupData(ctx context.Context, accountID, version int64, endpoint, address string, target any) error {
	parsedAddress, err := url.Parse(address)
	if err != nil {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Could not create the group request.", Cause: err}
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return service.readLargeAccountDetails(ctx, accountID, version, endpoint, parsedAddress, target, "Roblox returned invalid group data.")
}

func validCursor(cursor, endpoint string) error {
	if len(cursor) > maxCursorLength {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "This page has expired. Refresh the group."}
	}
	return nil
}

// UserGroups lists the groups that the user belongs to, with the user's role
// in each.
func (service *Users) UserGroups(ctx context.Context, accountID, version, userID int64) ([]UserGroup, error) {
	const endpoint = "groups-user"
	var payload struct {
		Data *[]struct {
			Group listedGroup     `json:"group"`
			Role  listedGroupRole `json:"role"`
		} `json:"data"`
	}
	address := groupsAddress("/v1/users/"+strconv.FormatInt(userID, 10)+"/groups/roles", url.Values{"includeLocked": {"true"}})
	if err := service.readGroupData(ctx, accountID, version, endpoint, address, &payload); err != nil {
		return nil, err
	}
	if payload.Data == nil {
		return nil, invalidGroupData(endpoint)
	}
	userGroups := make([]UserGroup, 0, len(*payload.Data))
	for _, listedMembership := range *payload.Data {
		if listedMembership.Group.ID > 0 {
			userGroups = append(userGroups, UserGroup{Group: listedMembership.Group.summary(), Role: listedMembership.Role.role()})
		}
	}
	return userGroups, nil
}

// SearchGroups finds groups by name. Roblox rejects keywords shorter than 2
// characters.
func (service *Users) SearchGroups(ctx context.Context, accountID, version int64, keyword, cursor string) (GroupSearchPage, error) {
	const endpoint = "groups-search"
	keyword = strings.TrimSpace(keyword)
	if length := utf8.RuneCountInString(keyword); length < minGroupSearchKeywordLength || length > maxGroupSearchKeywordLength {
		return GroupSearchPage{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Enter 2 to 50 characters to search groups."}
	}
	if err := validCursor(cursor, endpoint); err != nil {
		return GroupSearchPage{}, err
	}
	query := url.Values{"keyword": {keyword}, "limit": {strconv.Itoa(groupSearchPageSize)}}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	var payload struct {
		NextPageCursor *string        `json:"nextPageCursor"`
		Data           *[]listedGroup `json:"data"`
	}
	if err := service.readGroupData(ctx, accountID, version, endpoint, groupsAddress("/v1/groups/search", query), &payload); err != nil {
		return GroupSearchPage{}, err
	}
	if payload.Data == nil || len(*payload.Data) > groupSearchPageSize {
		return GroupSearchPage{}, invalidGroupData(endpoint)
	}
	page := GroupSearchPage{Groups: make([]GroupSummary, 0, len(*payload.Data))}
	for _, listedGroup := range *payload.Data {
		if listedGroup.ID > 0 {
			page.Groups = append(page.Groups, listedGroup.summary())
		}
	}
	if payload.NextPageCursor != nil && len(*payload.NextPageCursor) <= maxCursorLength {
		page.NextCursor = *payload.NextPageCursor
	}
	return page, nil
}

func (service *Users) GroupDetails(ctx context.Context, accountID, version, groupID int64) (GroupDetails, error) {
	const endpoint = "group-details"
	var payload listedGroup
	if err := service.readGroupData(ctx, accountID, version, endpoint, groupsAddress(groupPath(groupID, ""), nil), &payload); err != nil {
		return GroupDetails{}, err
	}
	if payload.ID != groupID {
		return GroupDetails{}, invalidGroupData(endpoint)
	}
	return payload.details(), nil
}

func (service *Users) GroupMembership(ctx context.Context, accountID, version, groupID int64) (GroupMembership, error) {
	const endpoint = "group-membership"
	var payload struct {
		IsPrimary     bool `json:"isPrimary"`
		IsPendingJoin bool `json:"isPendingJoin"`
		UserRole      *struct {
			Role listedGroupRole `json:"role"`
		} `json:"userRole"`
		Permissions struct {
			GroupPostsPermissions struct {
				ViewStatus   bool `json:"viewStatus"`
				PostToStatus bool `json:"postToStatus"`
			} `json:"groupPostsPermissions"`
			GroupMembershipPermissions struct {
				ChangeRank    bool `json:"changeRank"`
				InviteMembers bool `json:"inviteMembers"`
				RemoveMembers bool `json:"removeMembers"`
			} `json:"groupMembershipPermissions"`
			GroupEconomyPermissions struct {
				SpendGroupFunds  bool `json:"spendGroupFunds"`
				ManageGroupGames bool `json:"manageGroupGames"`
			} `json:"groupEconomyPermissions"`
		} `json:"permissions"`
		CanConfigure      bool `json:"canConfigure"`
		CanViewMemberList bool `json:"canViewMemberList"`
		IsOwner           bool `json:"isOwner"`
	}
	if err := service.readGroupData(ctx, accountID, version, endpoint, groupsAddress(groupPath(groupID, "/membership"), nil), &payload); err != nil {
		return GroupMembership{}, err
	}
	if payload.UserRole == nil {
		return GroupMembership{}, invalidGroupData(endpoint)
	}
	permissions := payload.Permissions
	return GroupMembership{
		Role: payload.UserRole.Role.role(), IsPrimary: payload.IsPrimary, IsPendingJoin: payload.IsPendingJoin,
		IsOwner: payload.IsOwner, CanViewMemberList: payload.CanViewMemberList,
		Permissions: GroupPermissions{
			CanViewShout: permissions.GroupPostsPermissions.ViewStatus, CanPostShout: permissions.GroupPostsPermissions.PostToStatus,
			CanChangeRank: permissions.GroupMembershipPermissions.ChangeRank, CanAcceptJoinRequests: permissions.GroupMembershipPermissions.InviteMembers,
			CanRemoveMembers: permissions.GroupMembershipPermissions.RemoveMembers, CanSpendFunds: permissions.GroupEconomyPermissions.SpendGroupFunds,
			CanManageGames: permissions.GroupEconomyPermissions.ManageGroupGames, CanConfigure: payload.CanConfigure,
		},
	}, nil
}

// GroupRoles lists the roles of the group from the lowest rank, without the
// guest role, which no member holds.
func (service *Users) GroupRoles(ctx context.Context, accountID, version, groupID int64) ([]GroupRole, error) {
	const endpoint = "group-roles"
	var payload struct {
		Roles *[]listedGroupRole `json:"roles"`
	}
	if err := service.readGroupData(ctx, accountID, version, endpoint, groupsAddress(groupPath(groupID, "/roles"), nil), &payload); err != nil {
		return nil, err
	}
	if payload.Roles == nil {
		return nil, invalidGroupData(endpoint)
	}
	roles := make([]GroupRole, 0, len(*payload.Roles))
	for _, listedRole := range *payload.Roles {
		if listedRole.ID > 0 && listedRole.Rank > 0 {
			roles = append(roles, listedRole.role())
		}
	}
	slices.SortStableFunc(roles, func(first, second GroupRole) int { return int(first.Rank - second.Rank) })
	return roles, nil
}

// GroupMembers reads one page of the members, newest first, or of the
// members of the role when role is not nil.
func (service *Users) GroupMembers(ctx context.Context, accountID, version, groupID int64, role *GroupRole, cursor string) (GroupMemberPage, error) {
	const endpoint = "group-members"
	if err := validCursor(cursor, endpoint); err != nil {
		return GroupMemberPage{}, err
	}
	query := url.Values{"limit": {strconv.Itoa(groupMemberPageSize)}, "sortOrder": {"Desc"}}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	var (
		page GroupMemberPage
		err  error
	)
	if role == nil {
		page, err = service.everyGroupMember(ctx, accountID, version, groupID, query)
	} else {
		page, err = service.groupRoleMembers(ctx, accountID, version, groupID, *role, query)
	}
	if err != nil {
		return GroupMemberPage{}, groupMemberListError(err, endpoint)
	}
	if len(page.NextCursor) > maxCursorLength {
		page.NextCursor = ""
	}
	return page, nil
}

func (service *Users) everyGroupMember(ctx context.Context, accountID, version, groupID int64, query url.Values) (GroupMemberPage, error) {
	const endpoint = "group-members"
	var payload struct {
		NextPageCursor *string `json:"nextPageCursor"`
		Data           *[]struct {
			User listedGroupUser `json:"user"`
			Role listedGroupRole `json:"role"`
		} `json:"data"`
	}
	if err := service.readGroupData(ctx, accountID, version, endpoint, groupsAddress(groupPath(groupID, "/users"), query), &payload); err != nil {
		return GroupMemberPage{}, err
	}
	if payload.Data == nil || len(*payload.Data) > groupMemberPageSize {
		return GroupMemberPage{}, invalidGroupData(endpoint)
	}
	page := GroupMemberPage{Members: make([]GroupMember, 0, len(*payload.Data))}
	for _, listedMember := range *payload.Data {
		if listedMember.User.UserID > 0 {
			page.Members = append(page.Members, GroupMember{User: listedMember.User.user(), Role: listedMember.Role.role()})
		}
	}
	if payload.NextPageCursor != nil {
		page.NextCursor = *payload.NextPageCursor
	}
	return page, nil
}

func (service *Users) groupRoleMembers(ctx context.Context, accountID, version, groupID int64, role GroupRole, query url.Values) (GroupMemberPage, error) {
	const endpoint = "group-members"
	var payload struct {
		NextPageCursor *string            `json:"nextPageCursor"`
		Data           *[]listedGroupUser `json:"data"`
	}
	address := groupsAddress(groupPath(groupID, "/roles/"+strconv.FormatInt(role.RoleID, 10)+"/users"), query)
	if err := service.readGroupData(ctx, accountID, version, endpoint, address, &payload); err != nil {
		return GroupMemberPage{}, err
	}
	if payload.Data == nil || len(*payload.Data) > groupMemberPageSize {
		return GroupMemberPage{}, invalidGroupData(endpoint)
	}
	page := GroupMemberPage{Members: make([]GroupMember, 0, len(*payload.Data))}
	for _, listedUser := range *payload.Data {
		if listedUser.UserID > 0 {
			page.Members = append(page.Members, GroupMember{User: listedUser.user(), Role: role})
		}
	}
	if payload.NextPageCursor != nil {
		page.NextCursor = *payload.NextPageCursor
	}
	return page, nil
}

// groupMemberListError explains a refusal, because a group can hide its
// member list from guests.
func groupMemberListError(err error, endpoint string) error {
	if remote, ok := errors.AsType[*roblox.Error](err); ok && remote.Kind == roblox.KindForbidden {
		return &roblox.Error{Kind: remote.Kind, Endpoint: endpoint, Status: remote.Status, RobloxCode: remote.RobloxCode, Message: "This group hides its member list.", Cause: err}
	}
	return err
}

func (service *Users) GroupJoinRequests(ctx context.Context, accountID, version, groupID int64, cursor string) (GroupJoinRequestPage, error) {
	const endpoint = "group-join-requests"
	if err := validCursor(cursor, endpoint); err != nil {
		return GroupJoinRequestPage{}, err
	}
	query := url.Values{"limit": {strconv.Itoa(groupMemberPageSize)}, "sortOrder": {"Desc"}}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	var payload struct {
		NextPageCursor *string `json:"nextPageCursor"`
		Data           *[]struct {
			Requester listedGroupUser `json:"requester"`
			Created   string          `json:"created"`
		} `json:"data"`
	}
	if err := service.readGroupData(ctx, accountID, version, endpoint, groupsAddress(groupPath(groupID, "/join-requests"), query), &payload); err != nil {
		return GroupJoinRequestPage{}, err
	}
	if payload.Data == nil || len(*payload.Data) > groupMemberPageSize {
		return GroupJoinRequestPage{}, invalidGroupData(endpoint)
	}
	page := GroupJoinRequestPage{Requests: make([]GroupJoinRequest, 0, len(*payload.Data))}
	for _, listedRequest := range *payload.Data {
		if listedRequest.Requester.UserID > 0 {
			page.Requests = append(page.Requests, GroupJoinRequest{User: listedRequest.Requester.user(), RequestedAtMs: robloxTimeMs(listedRequest.Created)})
		}
	}
	if payload.NextPageCursor != nil && len(*payload.NextPageCursor) <= maxCursorLength {
		page.NextCursor = *payload.NextPageCursor
	}
	return page, nil
}

func (service *Users) GroupSocialLinks(ctx context.Context, accountID, version, groupID int64) ([]GroupSocialLink, error) {
	const endpoint = "group-social-links"
	var payload struct {
		Data *[]struct {
			Type  string `json:"type"`
			URL   string `json:"url"`
			Title string `json:"title"`
		} `json:"data"`
	}
	if err := service.readGroupData(ctx, accountID, version, endpoint, groupsAddress(groupPath(groupID, "/social-links"), nil), &payload); err != nil {
		return nil, err
	}
	if payload.Data == nil {
		return nil, invalidGroupData(endpoint)
	}
	links := make([]GroupSocialLink, 0, len(*payload.Data))
	for _, listedLink := range *payload.Data {
		// Only web links open, so that a link never starts another program.
		if parsedLink, err := url.Parse(listedLink.URL); err == nil && parsedLink.Scheme == "https" && parsedLink.Host != "" {
			links = append(links, GroupSocialLink{Type: boundedText(listedLink.Type, maxGroupNameLength), URL: parsedLink.String(), Title: boundedText(listedLink.Title, maxGroupNameLength)})
		}
	}
	return links, nil
}

// RelatedGroups lists the first allies or enemies of the group.
func (service *Users) RelatedGroups(ctx context.Context, accountID, version, groupID int64, relationship GroupRelationship) ([]GroupSummary, error) {
	endpoint := "group-" + string(relationship)
	if relationship != GroupAllies && relationship != GroupEnemies {
		return nil, invalidGroupData(endpoint)
	}
	var payload struct {
		RelatedGroups *[]listedGroup `json:"relatedGroups"`
	}
	query := url.Values{"model.startRowIndex": {"0"}, "model.maxRows": {strconv.Itoa(groupRelationshipPageSize)}}
	if err := service.readGroupData(ctx, accountID, version, endpoint, groupsAddress(groupPath(groupID, "/relationships/"+string(relationship)), query), &payload); err != nil {
		return nil, err
	}
	if payload.RelatedGroups == nil {
		return nil, invalidGroupData(endpoint)
	}
	relatedGroups := make([]GroupSummary, 0, len(*payload.RelatedGroups))
	for _, listedGroup := range *payload.RelatedGroups {
		if listedGroup.ID > 0 {
			relatedGroups = append(relatedGroups, listedGroup.summary())
		}
	}
	return relatedGroups, nil
}

// GroupNameHistory lists the previous names of the group, newest first.
func (service *Users) GroupNameHistory(ctx context.Context, accountID, version, groupID int64) ([]GroupNameChange, error) {
	const endpoint = "group-name-history"
	var payload struct {
		Data *[]struct {
			Name    string `json:"name"`
			Created string `json:"created"`
		} `json:"data"`
	}
	query := url.Values{"limit": {strconv.Itoa(groupNameHistoryPageSize)}, "sortOrder": {"Desc"}}
	if err := service.readGroupData(ctx, accountID, version, endpoint, groupsAddress(groupPath(groupID, "/name-history"), query), &payload); err != nil {
		return nil, err
	}
	if payload.Data == nil {
		return nil, invalidGroupData(endpoint)
	}
	nameChanges := make([]GroupNameChange, 0, len(*payload.Data))
	for _, listedChange := range *payload.Data {
		if name := boundedText(listedChange.Name, maxGroupNameLength); name != "" {
			nameChanges = append(nameChanges, GroupNameChange{Name: name, ChangedAtMs: robloxTimeMs(listedChange.Created)})
		}
	}
	return nameChanges, nil
}

// GroupFunds reads the Robux of the group, which Roblox shows only to members
// who may spend them.
func (service *Users) GroupFunds(ctx context.Context, accountID, version, groupID int64) (int64, error) {
	const endpoint = "group-funds"
	var payload struct {
		Robux *int64 `json:"robux"`
	}
	address := "https://economy.roblox.com/v1/groups/" + strconv.FormatInt(groupID, 10) + "/currency"
	if err := service.readGroupData(ctx, accountID, version, endpoint, address, &payload); err != nil {
		return 0, err
	}
	if payload.Robux == nil || *payload.Robux < 0 {
		return 0, invalidGroupData(endpoint)
	}
	return *payload.Robux, nil
}

// GroupGames reads one page of every game of the group, newest first,
// including private games. The games API lists every game of a group to
// anyone with the All filter, and the develop API reports which ones are
// private. Verified on 2026-11-19.
func (service *Users) GroupGames(ctx context.Context, accountID, version, groupID int64, groupName, cursor string) (GroupGamePage, error) {
	const endpoint = "group-games"
	if err := validCursor(cursor, endpoint); err != nil {
		return GroupGamePage{}, err
	}
	query := url.Values{"accessFilter": {"All"}, "limit": {strconv.Itoa(groupGamePageSize)}, "sortOrder": {"Desc"}}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	var payload struct {
		NextPageCursor *string `json:"nextPageCursor"`
		Data           *[]struct {
			ID        int64  `json:"id"`
			Name      string `json:"name"`
			Updated   string `json:"updated"`
			Visits    int64  `json:"placeVisits"`
			RootPlace struct {
				ID int64 `json:"id"`
			} `json:"rootPlace"`
		} `json:"data"`
	}
	address := (&url.URL{Scheme: "https", Host: "games.roblox.com", Path: "/v2/groups/" + strconv.FormatInt(groupID, 10) + "/games", RawQuery: query.Encode()}).String()
	if err := service.readGroupData(ctx, accountID, version, endpoint, address, &payload); err != nil {
		return GroupGamePage{}, err
	}
	if payload.Data == nil || len(*payload.Data) > groupGamePageSize {
		return GroupGamePage{}, invalidGroupData(endpoint)
	}
	page := GroupGamePage{Games: make([]GroupGame, 0, len(*payload.Data))}
	universeIDs := make([]int64, 0, len(*payload.Data))
	for _, listedGame := range *payload.Data {
		if !games.ValidID(listedGame.ID) || !games.ValidID(listedGame.RootPlace.ID) {
			continue
		}
		universeIDs = append(universeIDs, listedGame.ID)
		page.Games = append(page.Games, GroupGame{
			Place: games.Place{
				PlaceID: listedGame.RootPlace.ID, UniverseID: listedGame.ID, Name: boundedText(listedGame.Name, maxGroupNameLength),
				CreatorID: groupID, CreatorName: groupName, CreatorType: "Group",
			},
			Visits: max(listedGame.Visits, 0), UpdatedAtMs: robloxTimeMs(listedGame.Updated),
		})
	}
	privateUniverseIDs, err := service.privateUniverseIDs(ctx, accountID, version, universeIDs)
	if err != nil {
		return GroupGamePage{}, err
	}
	for index := range page.Games {
		page.Games[index].IsPrivate = slices.Contains(privateUniverseIDs, page.Games[index].Place.UniverseID)
	}
	if payload.NextPageCursor != nil && len(*payload.NextPageCursor) <= maxCursorLength {
		page.NextCursor = *payload.NextPageCursor
	}
	return page, nil
}

// privateUniverseIDs lists the universes whose privacy type is Private.
func (service *Users) privateUniverseIDs(ctx context.Context, accountID, version int64, universeIDs []int64) ([]int64, error) {
	const endpoint = "group-games-privacy"
	if len(universeIDs) == 0 {
		return nil, nil
	}
	query := url.Values{}
	for _, universeID := range universeIDs {
		query.Add("ids", strconv.FormatInt(universeID, 10))
	}
	var payload struct {
		Data *[]struct {
			ID          int64  `json:"id"`
			PrivacyType string `json:"privacyType"`
		} `json:"data"`
	}
	address := (&url.URL{Scheme: "https", Host: "develop.roblox.com", Path: "/v1/universes/multiget", RawQuery: query.Encode()}).String()
	if err := service.readGroupData(ctx, accountID, version, endpoint, address, &payload); err != nil {
		return nil, err
	}
	if payload.Data == nil {
		return nil, invalidGroupData(endpoint)
	}
	privateIDs := []int64{}
	for _, listedUniverse := range *payload.Data {
		if listedUniverse.PrivacyType == "Private" {
			privateIDs = append(privateIDs, listedUniverse.ID)
		}
	}
	return privateIDs, nil
}

// JoinGroup joins the group, or asks to join a group that accepts members by
// request. The membership tells which one happened.
func (service *Users) JoinGroup(ctx context.Context, accountID, version, groupID int64) error {
	_, err := service.writeAccount(ctx, accountID, version, "group-join", http.MethodPost, groupsAddress(groupPath(groupID, "/users"), nil), map[string]any{})
	return err
}

// RemoveGroupMember removes a user from the group. It leaves the group when
// the user is the account itself.
func (service *Users) RemoveGroupMember(ctx context.Context, accountID, version, groupID, userID int64) error {
	_, err := service.writeAccount(ctx, accountID, version, "group-member-remove", http.MethodDelete, groupsAddress(groupPath(groupID, "/users/"+strconv.FormatInt(userID, 10)), nil), nil)
	return err
}

func (service *Users) SetGroupMemberRole(ctx context.Context, accountID, version, groupID, userID, roleID int64) error {
	_, err := service.writeAccount(ctx, accountID, version, "group-member-role", http.MethodPatch, groupsAddress(groupPath(groupID, "/users/"+strconv.FormatInt(userID, 10)), nil), map[string]int64{"roleId": roleID})
	return err
}

func (service *Users) SetPrimaryGroup(ctx context.Context, accountID, version, groupID int64) error {
	_, err := service.writeAccount(ctx, accountID, version, "group-primary", http.MethodPost, groupsAddress("/v1/user/groups/primary", nil), map[string]int64{"groupId": groupID})
	return err
}

func (service *Users) RemovePrimaryGroup(ctx context.Context, accountID, version int64) error {
	_, err := service.writeAccount(ctx, accountID, version, "group-primary-remove", http.MethodDelete, groupsAddress("/v1/user/groups/primary", nil), nil)
	return err
}

// PostGroupShout replaces the shout of the group. An empty message clears it.
func (service *Users) PostGroupShout(ctx context.Context, accountID, version, groupID int64, message string) error {
	_, err := service.writeAccount(ctx, accountID, version, "group-shout", http.MethodPatch, groupsAddress(groupPath(groupID, "/status"), nil), map[string]string{"message": message})
	return err
}

func (service *Users) SetGroupDescription(ctx context.Context, accountID, version, groupID int64, description string) error {
	_, err := service.writeAccount(ctx, accountID, version, "group-description", http.MethodPatch, groupsAddress(groupPath(groupID, "/description"), nil), map[string]string{"description": description})
	return err
}

// AnswerGroupJoinRequest accepts or declines the request of the user to join
// the group.
func (service *Users) AnswerGroupJoinRequest(ctx context.Context, accountID, version, groupID, userID int64, isAccepted bool) error {
	method := http.MethodDelete
	if isAccepted {
		method = http.MethodPost
	}
	_, err := service.writeAccount(ctx, accountID, version, "group-join-request", method, groupsAddress(groupPath(groupID, "/join-requests/users/"+strconv.FormatInt(userID, 10)), nil), nil)
	return err
}
