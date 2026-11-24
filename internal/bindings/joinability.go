package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/appservice"
	"github.com/sleaze5/RobloxAccountManager/internal/gamelaunch"
	"github.com/sleaze5/RobloxAccountManager/internal/games"
)

func (service *Service) CheckLaunchJoins(ctx context.Context, accountIDs []int64, input gamelaunch.Input) ([]appservice.AccountJoinCheck, error) {
	for _, accountID := range accountIDs {
		if err := positiveID(accountID, "join-check"); err != nil {
			return nil, err
		}
	}
	return service.core.CheckLaunchJoins(ctx, accountIDs, input)
}

func (service *Service) GetUserJoinability(ctx context.Context, accountID, userID int64) (games.UserJoinability, error) {
	if err := validateAccountAndTargetIDs(accountID, userID, "user-joinability"); err != nil {
		return games.UserJoinabilityUnknown, err
	}
	return service.core.GetUserJoinability(ctx, accountID, userID)
}
