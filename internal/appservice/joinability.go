package appservice

import (
	"context"
	"errors"

	"github.com/sleaze5/RobloxAccountManager/internal/gamelaunch"
	"github.com/sleaze5/RobloxAccountManager/internal/games"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	maxJoinCheckAccounts = 20
	// Each account's check costs up to three Roblox requests.
	joinCheckWorkerLimit = 4
)

// AccountJoinCheck is how joining the launch target would likely end for one
// account.
type AccountJoinCheck struct {
	AccountID int64           `json:"accountId"`
	Check     games.JoinCheck `json:"check"`
}

// CheckLaunchJoins asks Roblox how joining the launch target would end for the
// first accounts, so that the launch panel can warn before a launch. It never
// stops a launch. Input that opens Roblox without joining a game has nothing to
// check.
func (service *Service) CheckLaunchJoins(ctx context.Context, accountIDs []int64, input gamelaunch.Input) ([]AccountJoinCheck, error) {
	accountIDs = accountIDs[:min(len(accountIDs), maxJoinCheckAccounts)]
	target, err := gamelaunch.Parse(input)
	if err != nil {
		return nil, launchPreparationError(err)
	}
	if !target.HasDestination() {
		return []AccountJoinCheck{}, nil
	}
	checks := make([]AccountJoinCheck, len(accountIDs))
	forEachIndexConcurrently(len(accountIDs), joinCheckWorkerLimit, func(index int) {
		checks[index] = AccountJoinCheck{AccountID: accountIDs[index], Check: service.checkAccountJoin(ctx, accountIDs[index], target, input.ResolveCurrentGame)}
	})
	return checks, nil
}

// checkAccountJoin reports a failed check as an unknown outcome, because a
// check that fails says nothing about the join.
func (service *Service) checkAccountJoin(parent context.Context, accountID int64, target gamelaunch.Request, resolveCurrentGame bool) games.JoinCheck {
	unknown := games.JoinCheck{Outcome: games.JoinOutcomeUnknown}
	ctx, version, logger, done, err := service.beginFeatureOperation(parent, accountID, "launch", "join-check")
	if err != nil {
		return unknown
	}
	defer done()
	if target.UserID > 0 || target.Username != "" {
		target, err = service.resolveLaunchUser(ctx, accountID, target, resolveCurrentGame)
	} else {
		target, err = service.gameLinks.ResolveShare(ctx, accountID, version, target)
	}
	if err != nil {
		// A target that does not resolve makes the launch fail the same way.
		if failure, ok := errors.AsType[*roblox.Error](err); ok && failure.Kind == roblox.KindProtocol {
			return games.JoinCheck{Outcome: games.JoinOutcomeRestricted, Message: failure.Message}
		}
		if !errors.Is(err, context.Canceled) {
			logger.WarnContext(ctx, "could not resolve the join target", "error", err)
		}
		return unknown
	}
	check, err := service.games.CheckJoin(ctx, accountID, version, target)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			logger.WarnContext(ctx, "could not check the join", "error", err)
		}
		return unknown
	}
	logger.DebugContext(ctx, "join checked", "outcome", check.Outcome)
	return check
}

// GetUserJoinability reads whether the account can join the user, as the
// user's Roblox profile shows it.
func (service *Service) GetUserJoinability(ctx context.Context, accountID, userID int64) (games.UserJoinability, error) {
	ctx, version, logger, done, err := service.beginFeatureOperation(ctx, accountID, "social", "user-joinability")
	if err != nil {
		return games.UserJoinabilityUnknown, err
	}
	defer done()
	joinability, err := service.users.UserJoinability(ctx, accountID, version, userID)
	if err != nil {
		return games.UserJoinabilityUnknown, socialFailure(logger, err)
	}
	return joinability, nil
}
