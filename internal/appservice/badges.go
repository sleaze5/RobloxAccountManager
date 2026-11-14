package appservice

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// ListGameBadges lists the badges of a universe with their icons. A missing
// icon leaves the badge without one.
func (service *Service) ListGameBadges(ctx context.Context, universeID int64) ([]games.Badge, error) {
	if err := validGameID(universeID); err != nil {
		return nil, err
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return nil, err
	}
	defer done()
	badges, err := service.badges.ForUniverse(ctx, universeID)
	if err != nil {
		return nil, service.gamesError(ctx, "game-badges", err)
	}
	targets := make([]robloxservices.ThumbnailTarget, len(badges))
	for index, badge := range badges {
		targets[index] = robloxservices.ThumbnailTarget{Type: "BadgeIcon", TargetID: badge.BadgeID}
	}
	icons, err := service.thumbnails.Images(ctx, targets, robloxservices.SmallThumbnailSize)
	if ctx.Err() != nil {
		return nil, service.gamesError(ctx, "game-badges", ctx.Err())
	}
	if err != nil {
		service.logs.Module("application.games").WarnContext(ctx, "badge icons unavailable", "operation", "game-badge-icons", "universe_id", universeID, "error", err)
	}
	for index := range badges {
		badges[index].IconURL = icons[targets[index]]
	}
	return badges, nil
}

// ListBadgeAwards reads which of the badges the account earned, and when.
func (service *Service) ListBadgeAwards(ctx context.Context, query games.BadgeAwardQuery) ([]games.BadgeAward, error) {
	if len(query.BadgeIDs) > robloxservices.MaxUniverseBadges {
		return nil, gamesInputError("Too many badges were requested. Refresh the badges.")
	}
	for _, badgeID := range query.BadgeIDs {
		if !games.ValidID(badgeID) {
			return nil, gamesInputError("A badge is invalid. Refresh the badges.")
		}
	}
	ctx, version, done, err := service.beginAccountSession(ctx, query.AccountID, "badge-awards")
	if err != nil {
		return nil, err
	}
	defer done()
	account, err := service.accountRepository.GetView(ctx, query.AccountID)
	if err != nil {
		return nil, mapRepositoryError(err)
	}
	awards, err := service.badges.Awards(ctx, query.AccountID, version, account.RobloxUserID, query.BadgeIDs)
	if err != nil {
		return nil, service.gamesError(ctx, "badge-awards", err)
	}
	return awards, nil
}
