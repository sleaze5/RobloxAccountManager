package bindings

import (
	"context"

	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

func (service *Service) GetAvatarBackground(ctx context.Context, accountID, userID int64) (*robloxservices.AvatarBackground, error) {
	if err := validateAccountAndTargetIDs(accountID, userID, "avatar-background"); err != nil {
		return nil, err
	}
	return service.core.GetAvatarBackground(ctx, accountID, userID)
}
