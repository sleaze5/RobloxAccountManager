package bindings

import (
	"context"

	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

func (service *Service) GetAvatarPortrait(ctx context.Context, accountID, userID int64) (robloxservices.AvatarPortrait, error) {
	if err := validateAccountAndTargetIDs(accountID, userID, "avatar-portrait"); err != nil {
		return robloxservices.AvatarPortrait{}, err
	}
	return service.core.GetAvatarPortrait(ctx, accountID, userID)
}
