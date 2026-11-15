package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/appservice"
)

func (service *Service) GetRobloxNotifications(ctx context.Context, accountID int64, startIndex int) (appservice.RobloxNotificationPage, error) {
	if err := positiveID(accountID, "roblox-notifications"); err != nil {
		return appservice.RobloxNotificationPage{}, err
	}
	return service.core.GetRobloxNotifications(ctx, accountID, startIndex)
}

func (service *Service) MarkRobloxNotificationsSeen(ctx context.Context, accountID int64) error {
	if err := positiveID(accountID, "roblox-notifications"); err != nil {
		return err
	}
	return service.core.MarkRobloxNotificationsSeen(ctx, accountID)
}

func (service *Service) DismissRobloxNotification(ctx context.Context, accountID int64, notificationID, dismissAction string) error {
	if err := positiveID(accountID, "roblox-notifications"); err != nil {
		return err
	}
	return service.core.DismissRobloxNotification(ctx, accountID, notificationID, dismissAction)
}

func (service *Service) DismissAllRobloxNotifications(ctx context.Context, accountID int64, category appservice.RobloxNotificationCategory) (int, error) {
	if err := positiveID(accountID, "roblox-notifications"); err != nil {
		return 0, err
	}
	return service.core.DismissAllRobloxNotifications(ctx, accountID, category)
}
