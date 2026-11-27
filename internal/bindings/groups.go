package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/appservice"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// maxGroupIconCount is the most icons that one thumbnails request reads.
const maxGroupIconCount = 100

func (service *Service) GetUserGroups(ctx context.Context, accountID, userID int64) (appservice.UserGroupList, error) {
	if err := validateAccountAndTargetIDs(accountID, userID, "groups-user"); err != nil {
		return appservice.UserGroupList{}, err
	}
	return service.core.GetUserGroups(ctx, accountID, userID)
}

func (service *Service) SearchGroups(ctx context.Context, accountID int64, keyword, cursor string) (robloxservices.GroupSearchPage, error) {
	if err := positiveID(accountID, "groups-search"); err != nil {
		return robloxservices.GroupSearchPage{}, err
	}
	return service.core.SearchGroups(ctx, accountID, keyword, cursor)
}

func (service *Service) GetGroupProfile(ctx context.Context, accountID, groupID int64) (appservice.GroupProfile, error) {
	if err := validateAccountAndTargetIDs(accountID, groupID, "group-profile"); err != nil {
		return appservice.GroupProfile{}, err
	}
	return service.core.GetGroupProfile(ctx, accountID, groupID)
}

func (service *Service) GetGroupMembers(ctx context.Context, accountID, groupID int64, role *robloxservices.GroupRole, cursor string) (robloxservices.GroupMemberPage, error) {
	const operation = "group-members"
	if err := validateAccountAndTargetIDs(accountID, groupID, operation); err != nil {
		return robloxservices.GroupMemberPage{}, err
	}
	if role != nil {
		if err := positiveID(role.RoleID, operation); err != nil {
			return robloxservices.GroupMemberPage{}, err
		}
	}
	return service.core.GetGroupMembers(ctx, accountID, groupID, role, cursor)
}

func (service *Service) GetGroupJoinRequests(ctx context.Context, accountID, groupID int64, cursor string) (robloxservices.GroupJoinRequestPage, error) {
	if err := validateAccountAndTargetIDs(accountID, groupID, "group-join-requests"); err != nil {
		return robloxservices.GroupJoinRequestPage{}, err
	}
	return service.core.GetGroupJoinRequests(ctx, accountID, groupID, cursor)
}

func (service *Service) GetGroupGames(ctx context.Context, accountID, groupID int64, groupName, cursor string) (robloxservices.GroupGamePage, error) {
	if err := validateAccountAndTargetIDs(accountID, groupID, "group-games"); err != nil {
		return robloxservices.GroupGamePage{}, err
	}
	return service.core.GetGroupGames(ctx, accountID, groupID, groupName, cursor)
}

func (service *Service) JoinGroup(ctx context.Context, accountID, groupID int64) error {
	if err := validateAccountAndTargetIDs(accountID, groupID, "group-join"); err != nil {
		return err
	}
	return service.core.JoinGroup(ctx, accountID, groupID)
}

func (service *Service) LeaveGroup(ctx context.Context, accountID, groupID int64) error {
	if err := validateAccountAndTargetIDs(accountID, groupID, "group-leave"); err != nil {
		return err
	}
	return service.core.LeaveGroup(ctx, accountID, groupID)
}

func (service *Service) SetPrimaryGroup(ctx context.Context, accountID, groupID int64) error {
	if err := validateAccountAndTargetIDs(accountID, groupID, "group-primary"); err != nil {
		return err
	}
	return service.core.SetPrimaryGroup(ctx, accountID, groupID)
}

func (service *Service) RemovePrimaryGroup(ctx context.Context, accountID int64) error {
	if err := positiveID(accountID, "group-primary-remove"); err != nil {
		return err
	}
	return service.core.RemovePrimaryGroup(ctx, accountID)
}

func (service *Service) PostGroupShout(ctx context.Context, accountID, groupID int64, message string) (string, error) {
	if err := validateAccountAndTargetIDs(accountID, groupID, "group-shout"); err != nil {
		return "", err
	}
	return service.core.PostGroupShout(ctx, accountID, groupID, message)
}

func (service *Service) SetGroupDescription(ctx context.Context, accountID, groupID int64, description string) (string, error) {
	if err := validateAccountAndTargetIDs(accountID, groupID, "group-description"); err != nil {
		return "", err
	}
	return service.core.SetGroupDescription(ctx, accountID, groupID, description)
}

func (service *Service) SetGroupMemberRole(ctx context.Context, accountID, groupID, userID, roleID int64) error {
	const operation = "group-member-role"
	if err := validateAccountAndTargetIDs(accountID, groupID, operation); err != nil {
		return err
	}
	if err := validateAccountAndTargetIDs(userID, roleID, operation); err != nil {
		return err
	}
	return service.core.SetGroupMemberRole(ctx, accountID, groupID, userID, roleID)
}

func (service *Service) RemoveGroupMember(ctx context.Context, accountID, groupID, userID int64) error {
	const operation = "group-member-remove"
	if err := validateAccountAndTargetIDs(accountID, groupID, operation); err != nil {
		return err
	}
	if err := positiveID(userID, operation); err != nil {
		return err
	}
	return service.core.RemoveGroupMember(ctx, accountID, groupID, userID)
}

func (service *Service) AnswerGroupJoinRequest(ctx context.Context, accountID, groupID, userID int64, isAccepted bool) error {
	const operation = "group-join-request"
	if err := validateAccountAndTargetIDs(accountID, groupID, operation); err != nil {
		return err
	}
	if err := positiveID(userID, operation); err != nil {
		return err
	}
	return service.core.AnswerGroupJoinRequest(ctx, accountID, groupID, userID, isAccepted)
}

func (service *Service) GetGroupIcons(ctx context.Context, groupIDs []int64) ([]appservice.GroupIconView, error) {
	if len(groupIDs) > maxGroupIconCount {
		return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "group-icons", Message: "Load fewer group icons at once."}
	}
	for _, groupID := range groupIDs {
		if err := positiveID(groupID, "group-icons"); err != nil {
			return nil, err
		}
	}
	return service.core.GetGroupIcons(ctx, groupIDs)
}

func (service *Service) GetGroupSettings(ctx context.Context, accountID, groupID int64) (robloxservices.GroupSettings, error) {
	if err := validateAccountAndTargetIDs(accountID, groupID, "group-settings"); err != nil {
		return robloxservices.GroupSettings{}, err
	}
	return service.core.GetGroupSettings(ctx, accountID, groupID)
}

func (service *Service) SaveGroupSettings(ctx context.Context, accountID, groupID int64, change robloxservices.GroupSettingsChange) error {
	if err := validateAccountAndTargetIDs(accountID, groupID, "group-settings-update"); err != nil {
		return err
	}
	return service.core.SaveGroupSettings(ctx, accountID, groupID, change)
}
