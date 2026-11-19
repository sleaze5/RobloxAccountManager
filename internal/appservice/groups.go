package appservice

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// GroupSource names a part of a group profile that can fail without failing
// the profile.
type GroupSource string

const (
	GroupSourceRoles         GroupSource = "roles"
	GroupSourceSocialLinks   GroupSource = "socialLinks"
	GroupSourceAllies        GroupSource = "allies"
	GroupSourceEnemies       GroupSource = "enemies"
	GroupSourcePreviousNames GroupSource = "previousNames"
	GroupSourceFunds         GroupSource = "funds"
)

// UserGroupList is every group of a user, with the user's primary group.
type UserGroupList struct {
	Groups []robloxservices.UserGroup `json:"groups"`
	// PrimaryGroupID is 0 when the user has no primary group.
	PrimaryGroupID int64 `json:"primaryGroupId"`
}

// GroupProfile is everything that the group page shows at first.
type GroupProfile struct {
	Details       robloxservices.GroupDetails      `json:"details"`
	Membership    robloxservices.GroupMembership   `json:"membership"`
	Roles         []robloxservices.GroupRole       `json:"roles"`
	SocialLinks   []robloxservices.GroupSocialLink `json:"socialLinks"`
	Allies        []robloxservices.GroupSummary    `json:"allies"`
	Enemies       []robloxservices.GroupSummary    `json:"enemies"`
	PreviousNames []robloxservices.GroupNameChange `json:"previousNames"`
	// FundsInRobux is nil when the account may not see the funds.
	FundsInRobux *int64 `json:"fundsInRobux"`
	// Unavailable lists the parts whose read failed.
	Unavailable []GroupSource `json:"unavailable"`
}

func (service *Service) groupLogger(operation string, accountID int64) *slog.Logger {
	return service.logs.Module("application.groups").With("operation", operation, "account_id", accountID)
}

// GetUserGroups lists the groups of a user, as the account sees them, with
// their icons.
func (service *Service) GetUserGroups(ctx context.Context, accountID, userID int64) (UserGroupList, error) {
	const operation = "groups-user"
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, operation)
	if err != nil {
		return UserGroupList{}, err
	}
	defer done()
	logger := service.groupLogger(operation, accountID)
	list := UserGroupList{}
	reads := []accountRead{
		applyRead("groups", func() ([]robloxservices.UserGroup, error) {
			return service.users.UserGroups(ctx, accountID, version, userID)
		}, func(groups []robloxservices.UserGroup) { list.Groups = groups }),
		applyRead("primary", func() (*robloxservices.UserPrimaryGroup, error) {
			return service.users.PrimaryGroup(ctx, accountID, version, userID)
		}, func(primaryGroup *robloxservices.UserPrimaryGroup) {
			if primaryGroup != nil {
				list.PrimaryGroupID = primaryGroup.ID
			}
		}),
	}
	unavailable, err := service.runAccountReads(ctx, accountID, version, operation, logger, reads)
	if err != nil {
		return UserGroupList{}, err
	}
	if slices.Contains(unavailable, "groups") {
		return UserGroupList{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: operation, Message: "Could not load the groups. Try again."}
	}
	summaries := make([]*robloxservices.GroupSummary, len(list.Groups))
	for index := range list.Groups {
		summaries[index] = &list.Groups[index].Group
	}
	service.addGroupIcons(ctx, summaries, logger)
	return list, nil
}

func (service *Service) SearchGroups(ctx context.Context, accountID int64, keyword, cursor string) (robloxservices.GroupSearchPage, error) {
	const operation = "groups-search"
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, operation)
	if err != nil {
		return robloxservices.GroupSearchPage{}, err
	}
	defer done()
	logger := service.groupLogger(operation, accountID)
	page, err := service.users.SearchGroups(ctx, accountID, version, keyword, cursor)
	if err != nil {
		logger.WarnContext(ctx, "group search failed", "error", err)
		return robloxservices.GroupSearchPage{}, err
	}
	summaries := make([]*robloxservices.GroupSummary, len(page.Groups))
	for index := range page.Groups {
		summaries[index] = &page.Groups[index]
	}
	service.addGroupIcons(ctx, summaries, logger)
	return page, nil
}

// GetGroupProfile reads the details of the group, how the account belongs
// to it, and the parts that the group page shows at first. The details and
// the membership must load, and the other parts are listed in Unavailable
// when they fail.
func (service *Service) GetGroupProfile(ctx context.Context, accountID, groupID int64) (GroupProfile, error) {
	const operation = "group-profile"
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, operation)
	if err != nil {
		return GroupProfile{}, err
	}
	defer done()
	logger := service.groupLogger(operation, accountID).With("group_id", groupID)
	users := service.users
	profile := GroupProfile{Roles: []robloxservices.GroupRole{}, SocialLinks: []robloxservices.GroupSocialLink{}, Allies: []robloxservices.GroupSummary{}, Enemies: []robloxservices.GroupSummary{}, PreviousNames: []robloxservices.GroupNameChange{}}
	reads := []accountRead{
		applyRead("details", func() (robloxservices.GroupDetails, error) {
			return users.GroupDetails(ctx, accountID, version, groupID)
		}, func(details robloxservices.GroupDetails) { profile.Details = details }),
		applyRead("membership", func() (robloxservices.GroupMembership, error) {
			return users.GroupMembership(ctx, accountID, version, groupID)
		}, func(membership robloxservices.GroupMembership) { profile.Membership = membership }),
		applyRead(string(GroupSourceRoles), func() ([]robloxservices.GroupRole, error) { return users.GroupRoles(ctx, accountID, version, groupID) }, func(roles []robloxservices.GroupRole) { profile.Roles = roles }),
		applyRead(string(GroupSourceSocialLinks), func() ([]robloxservices.GroupSocialLink, error) {
			return users.GroupSocialLinks(ctx, accountID, version, groupID)
		}, func(links []robloxservices.GroupSocialLink) { profile.SocialLinks = links }),
		applyRead(string(GroupSourceAllies), func() ([]robloxservices.GroupSummary, error) {
			return users.RelatedGroups(ctx, accountID, version, groupID, robloxservices.GroupAllies)
		}, func(allies []robloxservices.GroupSummary) { profile.Allies = allies }),
		applyRead(string(GroupSourceEnemies), func() ([]robloxservices.GroupSummary, error) {
			return users.RelatedGroups(ctx, accountID, version, groupID, robloxservices.GroupEnemies)
		}, func(enemies []robloxservices.GroupSummary) { profile.Enemies = enemies }),
		applyRead(string(GroupSourcePreviousNames), func() ([]robloxservices.GroupNameChange, error) {
			return users.GroupNameHistory(ctx, accountID, version, groupID)
		}, func(nameChanges []robloxservices.GroupNameChange) { profile.PreviousNames = nameChanges }),
	}
	unavailable, err := service.runAccountReads(ctx, accountID, version, operation, logger, reads)
	if err != nil {
		return GroupProfile{}, err
	}
	if slices.Contains(unavailable, "details") || slices.Contains(unavailable, "membership") {
		return GroupProfile{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: operation, Message: "Could not load the group. Try again."}
	}
	profile.Unavailable = make([]GroupSource, len(unavailable))
	for index, source := range unavailable {
		profile.Unavailable[index] = GroupSource(source)
	}
	// Roblox shows the funds only to members who may spend them.
	if profile.Membership.Permissions.CanSpendFunds {
		funds, err := users.GroupFunds(ctx, accountID, version, groupID)
		if err != nil {
			logger.WarnContext(ctx, "group funds unavailable", "error", err)
			profile.Unavailable = append(profile.Unavailable, GroupSourceFunds)
		} else {
			profile.FundsInRobux = &funds
		}
	}
	summaries := []*robloxservices.GroupSummary{&profile.Details.Group}
	for index := range profile.Allies {
		summaries = append(summaries, &profile.Allies[index])
	}
	for index := range profile.Enemies {
		summaries = append(summaries, &profile.Enemies[index])
	}
	service.addGroupIcons(ctx, summaries, logger)
	return profile, nil
}

// GroupIconView is the icon of one group.
type GroupIconView struct {
	GroupID  int64  `json:"groupId"`
	ImageURL string `json:"imageUrl"`
}

// GetGroupIcons reads the icons of the groups that other pages name, such as
// the group that created an item. A group without an icon is left out.
func (service *Service) GetGroupIcons(ctx context.Context, groupIDs []int64) ([]GroupIconView, error) {
	targets := make([]robloxservices.ThumbnailTarget, len(groupIDs))
	for index, groupID := range groupIDs {
		targets[index] = robloxservices.ThumbnailTarget{Type: "GroupIcon", TargetID: groupID}
	}
	images, err := service.thumbnails.Images(ctx, targets, robloxservices.SmallThumbnailSize)
	if err != nil {
		service.logs.Module("application.groups").WarnContext(ctx, "group icons unavailable", "operation", "group-icons", "groups", len(groupIDs), "error", err)
		return nil, err
	}
	icons := make([]GroupIconView, 0, len(groupIDs))
	for index, groupID := range groupIDs {
		if imageURL := images[targets[index]]; imageURL != "" {
			icons = append(icons, GroupIconView{GroupID: groupID, ImageURL: imageURL})
		}
	}
	return icons, nil
}

// addGroupIcons adds the icon of each group. A missing icon leaves the group
// without it.
func (service *Service) addGroupIcons(ctx context.Context, summaries []*robloxservices.GroupSummary, logger *slog.Logger) {
	if len(summaries) == 0 {
		return
	}
	targets := make([]robloxservices.ThumbnailTarget, len(summaries))
	for index, summary := range summaries {
		targets[index] = robloxservices.ThumbnailTarget{Type: "GroupIcon", TargetID: summary.GroupID}
	}
	images, err := service.thumbnails.Images(ctx, targets, robloxservices.SmallThumbnailSize)
	if err != nil {
		logger.WarnContext(ctx, "group icons unavailable", "error", err)
	}
	for index, summary := range summaries {
		summary.IconURL = images[targets[index]]
	}
}

// GetGroupMembers reads one page of the members of the group, or of one role
// when role is not nil.
func (service *Service) GetGroupMembers(ctx context.Context, accountID, groupID int64, role *robloxservices.GroupRole, cursor string) (robloxservices.GroupMemberPage, error) {
	const operation = "group-members"
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, operation)
	if err != nil {
		return robloxservices.GroupMemberPage{}, err
	}
	defer done()
	page, err := service.users.GroupMembers(ctx, accountID, version, groupID, role, cursor)
	if err != nil {
		service.groupLogger(operation, accountID).WarnContext(ctx, "group members unavailable", "group_id", groupID, "error", err)
		return robloxservices.GroupMemberPage{}, err
	}
	return page, nil
}

func (service *Service) GetGroupJoinRequests(ctx context.Context, accountID, groupID int64, cursor string) (robloxservices.GroupJoinRequestPage, error) {
	const operation = "group-join-requests"
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, operation)
	if err != nil {
		return robloxservices.GroupJoinRequestPage{}, err
	}
	defer done()
	page, err := service.users.GroupJoinRequests(ctx, accountID, version, groupID, cursor)
	if err != nil {
		service.groupLogger(operation, accountID).WarnContext(ctx, "group join requests unavailable", "group_id", groupID, "error", err)
		return robloxservices.GroupJoinRequestPage{}, err
	}
	return page, nil
}

// GetGroupGames reads one page of every game of the group, private games
// included, with their icons.
func (service *Service) GetGroupGames(ctx context.Context, accountID, groupID int64, groupName, cursor string) (robloxservices.GroupGamePage, error) {
	const operation = "group-games"
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, operation)
	if err != nil {
		return robloxservices.GroupGamePage{}, err
	}
	defer done()
	logger := service.groupLogger(operation, accountID).With("group_id", groupID)
	page, err := service.users.GroupGames(ctx, accountID, version, groupID, groupName, cursor)
	if err != nil {
		logger.WarnContext(ctx, "group games unavailable", "error", err)
		return robloxservices.GroupGamePage{}, err
	}
	targets := make([]robloxservices.ThumbnailTarget, len(page.Games))
	for index, game := range page.Games {
		targets[index] = robloxservices.ThumbnailTarget{Type: "GameIcon", TargetID: game.Place.UniverseID}
	}
	images, err := service.thumbnails.Images(ctx, targets, robloxservices.SmallThumbnailSize)
	if err != nil {
		logger.WarnContext(ctx, "group game icons unavailable", "error", err)
	}
	for index := range page.Games {
		page.Games[index].Place.IconURL = images[targets[index]]
	}
	return page, nil
}

func (service *Service) JoinGroup(ctx context.Context, accountID, groupID int64) error {
	return service.editAccount(ctx, accountID, "group-join", "Roblox did not let this account join the group.", func(ctx context.Context, version int64) error {
		return service.users.JoinGroup(ctx, accountID, version, groupID)
	})
}

func (service *Service) LeaveGroup(ctx context.Context, accountID, groupID int64) error {
	account, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return mapRepositoryError(err)
	}
	return service.editAccount(ctx, accountID, "group-leave", "Roblox did not let this account leave the group. The owner cannot leave.", func(ctx context.Context, version int64) error {
		return service.users.RemoveGroupMember(ctx, accountID, version, groupID, account.RobloxUserID)
	})
}

func (service *Service) SetPrimaryGroup(ctx context.Context, accountID, groupID int64) error {
	return service.editAccount(ctx, accountID, "group-primary", "Only a member can make the group primary.", func(ctx context.Context, version int64) error {
		return service.users.SetPrimaryGroup(ctx, accountID, version, groupID)
	})
}

func (service *Service) RemovePrimaryGroup(ctx context.Context, accountID int64) error {
	return service.editAccount(ctx, accountID, "group-primary-remove", "", func(ctx context.Context, version int64) error {
		return service.users.RemovePrimaryGroup(ctx, accountID, version)
	})
}

// PostGroupShout replaces the shout of the group and returns the saved text.
// An empty message clears the shout.
func (service *Service) PostGroupShout(ctx context.Context, accountID, groupID int64, message string) (string, error) {
	const operation = "group-shout"
	message = strings.TrimSpace(message)
	if utf8.RuneCountInString(message) > robloxservices.MaxGroupShoutLength || !utf8.ValidString(message) {
		return "", accountEditError(operation, fmt.Sprintf("Shorten the shout to %d characters or fewer.", robloxservices.MaxGroupShoutLength))
	}
	err := service.editAccount(ctx, accountID, operation, "This role cannot post the group shout.", func(ctx context.Context, version int64) error {
		return service.users.PostGroupShout(ctx, accountID, version, groupID, message)
	})
	return message, err
}

// SetGroupDescription saves the description of the group and returns the
// saved text.
func (service *Service) SetGroupDescription(ctx context.Context, accountID, groupID int64, description string) (string, error) {
	const operation = "group-description"
	description = strings.TrimSpace(description)
	if utf8.RuneCountInString(description) > robloxservices.MaxGroupDescriptionLength || !utf8.ValidString(description) {
		return "", accountEditError(operation, fmt.Sprintf("Shorten the description to %d characters or fewer.", robloxservices.MaxGroupDescriptionLength))
	}
	err := service.editAccount(ctx, accountID, operation, "This role cannot change the group description.", func(ctx context.Context, version int64) error {
		return service.users.SetGroupDescription(ctx, accountID, version, groupID, description)
	})
	return description, err
}

func (service *Service) SetGroupMemberRole(ctx context.Context, accountID, groupID, userID, roleID int64) error {
	return service.editAccount(ctx, accountID, "group-member-role", "This role cannot give that role.", func(ctx context.Context, version int64) error {
		return service.users.SetGroupMemberRole(ctx, accountID, version, groupID, userID, roleID)
	})
}

func (service *Service) RemoveGroupMember(ctx context.Context, accountID, groupID, userID int64) error {
	return service.editAccount(ctx, accountID, "group-member-remove", "This role cannot remove that member.", func(ctx context.Context, version int64) error {
		return service.users.RemoveGroupMember(ctx, accountID, version, groupID, userID)
	})
}

// AnswerGroupJoinRequest accepts or declines the request of a user to join
// the group.
func (service *Service) AnswerGroupJoinRequest(ctx context.Context, accountID, groupID, userID int64, isAccepted bool) error {
	return service.editAccount(ctx, accountID, "group-join-request", "This role cannot answer join requests.", func(ctx context.Context, version int64) error {
		return service.users.AnswerGroupJoinRequest(ctx, accountID, version, groupID, userID, isAccepted)
	})
}
