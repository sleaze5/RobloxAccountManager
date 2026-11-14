package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/appservice"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

func (service *Service) GetSocialPage(ctx context.Context, accountID, userID int64, list appservice.SocialList, cursor string) (appservice.SocialPage, error) {
	if err := socialTarget(accountID, userID, "social"); err != nil {
		return appservice.SocialPage{}, err
	}
	return service.core.GetSocialPage(ctx, accountID, userID, list, cursor)
}

func (service *Service) SearchPlayers(ctx context.Context, accountID int64, keyword, cursor string) (appservice.SocialPage, error) {
	if err := positiveID(accountID, "player-search"); err != nil {
		return appservice.SocialPage{}, err
	}
	return service.core.SearchPlayers(ctx, accountID, keyword, cursor)
}

func (service *Service) GetSocialPresences(ctx context.Context, accountID int64, userIDs []int64) ([]robloxservices.UserPresence, error) {
	if err := positiveID(accountID, "social-presence"); err != nil {
		return nil, err
	}
	for _, userID := range userIDs {
		if err := positiveID(userID, "social-presence"); err != nil {
			return nil, err
		}
	}
	return service.core.GetSocialPresences(ctx, accountID, userIDs)
}

func (service *Service) GetSocialProfile(ctx context.Context, accountID, userID int64, section appservice.SocialProfileSection) (appservice.SocialProfile, error) {
	if err := socialTarget(accountID, userID, "social-profile"); err != nil {
		return appservice.SocialProfile{}, err
	}
	return service.core.GetSocialProfile(ctx, accountID, userID, section)
}

func (service *Service) GetSocialRelationship(ctx context.Context, accountID, userID int64) (appservice.SocialRelationshipSnapshot, error) {
	if err := socialTarget(accountID, userID, "social-relationship"); err != nil {
		return appservice.SocialRelationshipSnapshot{}, err
	}
	return service.core.GetSocialRelationship(ctx, accountID, userID)
}

func (service *Service) SendFriendRequest(ctx context.Context, accountID, userID int64) error {
	if err := socialTarget(accountID, userID, "friend-request-send"); err != nil {
		return err
	}
	return service.core.SendFriendRequest(ctx, accountID, userID)
}

func (service *Service) SendTrustedFriendRequest(ctx context.Context, accountID, userID int64) error {
	if err := socialTarget(accountID, userID, "trusted-friend-request-send"); err != nil {
		return err
	}
	return service.core.SendTrustedFriendRequest(ctx, accountID, userID)
}

func (service *Service) AcceptTrustedFriendRequest(ctx context.Context, accountID, userID int64) error {
	if err := socialTarget(accountID, userID, "trusted-friend-request-accept"); err != nil {
		return err
	}
	return service.core.AcceptTrustedFriendRequest(ctx, accountID, userID)
}

func (service *Service) DeclineTrustedFriendRequest(ctx context.Context, accountID, userID int64) error {
	if err := socialTarget(accountID, userID, "trusted-friend-request-decline"); err != nil {
		return err
	}
	return service.core.DeclineTrustedFriendRequest(ctx, accountID, userID)
}

func (service *Service) RemoveTrustedFriend(ctx context.Context, accountID, userID int64) error {
	if err := socialTarget(accountID, userID, "trusted-friend-remove"); err != nil {
		return err
	}
	return service.core.RemoveTrustedFriend(ctx, accountID, userID)
}

func (service *Service) AcceptFriendRequest(ctx context.Context, accountID, userID int64) error {
	if err := socialTarget(accountID, userID, "friend-request-accept"); err != nil {
		return err
	}
	return service.core.AcceptFriendRequest(ctx, accountID, userID)
}

func (service *Service) DeclineFriendRequest(ctx context.Context, accountID, userID int64) error {
	if err := socialTarget(accountID, userID, "friend-request-decline"); err != nil {
		return err
	}
	return service.core.DeclineFriendRequest(ctx, accountID, userID)
}

func (service *Service) Unfriend(ctx context.Context, accountID, userID int64) error {
	if err := socialTarget(accountID, userID, "unfriend"); err != nil {
		return err
	}
	return service.core.Unfriend(ctx, accountID, userID)
}

func (service *Service) Follow(ctx context.Context, accountID, userID int64) error {
	if err := socialTarget(accountID, userID, "follow"); err != nil {
		return err
	}
	return service.core.Follow(ctx, accountID, userID)
}

func (service *Service) Unfollow(ctx context.Context, accountID, userID int64) error {
	if err := socialTarget(accountID, userID, "unfollow"); err != nil {
		return err
	}
	return service.core.Unfollow(ctx, accountID, userID)
}

func (service *Service) SetNickname(ctx context.Context, accountID, userID int64, nickname string) (string, error) {
	if err := socialTarget(accountID, userID, "nickname-update"); err != nil {
		return "", err
	}
	return service.core.SetNickname(ctx, accountID, userID, nickname)
}

func socialTarget(accountID, userID int64, endpoint string) error {
	if err := positiveID(accountID, endpoint); err != nil {
		return err
	}
	return positiveID(userID, endpoint)
}

func (service *Service) RemoveSocialConnections(ctx context.Context, accountID int64, list appservice.SocialList, userIDs []int64) (appservice.SocialRemoval, error) {
	if err := positiveID(accountID, "social-bulk-remove"); err != nil {
		return appservice.SocialRemoval{}, err
	}
	return service.core.RemoveSocialConnections(ctx, accountID, list, userIDs)
}

func (service *Service) FindPlayerByUsername(ctx context.Context, username string) (*appservice.SocialUser, error) {
	return service.core.FindPlayerByUsername(ctx, username)
}

func (service *Service) GetInventoryPage(ctx context.Context, accountID, userID int64, category robloxservices.InventoryCategory, cursor string) (robloxservices.InventoryPage, error) {
	if err := socialTarget(accountID, userID, "inventory"); err != nil {
		return robloxservices.InventoryPage{}, err
	}
	return service.core.GetInventoryPage(ctx, accountID, userID, category, cursor)
}

func (service *Service) IsInventoryItemFavorite(ctx context.Context, accountID int64, itemType robloxservices.CatalogItemType, itemID int64) (bool, error) {
	if err := socialTarget(accountID, itemID, "item-favorite"); err != nil {
		return false, err
	}
	return service.core.IsInventoryItemFavorite(ctx, accountID, itemType, itemID)
}

func (service *Service) SetInventoryItemFavorite(ctx context.Context, accountID int64, itemType robloxservices.CatalogItemType, itemID int64, isFavorite bool) error {
	if err := socialTarget(accountID, itemID, "item-favorite-update"); err != nil {
		return err
	}
	return service.core.SetInventoryItemFavorite(ctx, accountID, itemType, itemID, isFavorite)
}

func (service *Service) DeleteInventoryAsset(ctx context.Context, accountID, assetID int64) error {
	if err := socialTarget(accountID, assetID, "inventory-asset-delete"); err != nil {
		return err
	}
	return service.core.DeleteInventoryAsset(ctx, accountID, assetID)
}

func (service *Service) GetWornAssetIDs(ctx context.Context, accountID, userID int64) ([]int64, error) {
	if err := socialTarget(accountID, userID, "inventory-worn-assets"); err != nil {
		return nil, err
	}
	return service.core.GetWornAssetIDs(ctx, accountID, userID)
}
