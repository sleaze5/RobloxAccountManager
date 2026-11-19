package bindings

import (
	"context"

	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

func (service *Service) GetAvatarAnimation(ctx context.Context, accountID int64, reference string) (robloxservices.InventoryItem, error) {
	if err := positiveID(accountID, "avatar-animation"); err != nil {
		return robloxservices.InventoryItem{}, err
	}
	return service.core.GetAvatarAnimation(ctx, accountID, reference)
}
