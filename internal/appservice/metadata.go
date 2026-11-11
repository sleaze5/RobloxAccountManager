package appservice

import (
	"context"
	"fmt"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
)

func (service *Service) refreshMetadata(key string, refresh func(context.Context)) {
	service.mutex.Lock()
	parent := service.gamesContext
	service.mutex.Unlock()
	if parent == nil || parent.Err() != nil {
		return
	}
	if _, loaded := service.metadataRefresh.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	go func() {
		defer service.metadataRefresh.Delete(key)
		ctx, cancel := context.WithTimeout(parent, 45*time.Second)
		defer cancel()
		select {
		case service.metadataSlots <- struct{}{}:
		case <-ctx.Done():
			return
		}
		if ctx.Err() == nil {
			refresh(ctx)
		}
		<-service.metadataSlots
		timer := time.NewTimer(time.Minute)
		defer timer.Stop()
		select {
		case <-parent.Done():
		case <-timer.C:
		}
	}()
}

func (service *Service) refreshFavoriteMetadata(places []games.Place) {
	service.refreshMetadata("favorites", func(ctx context.Context) {
		for _, place := range places {
			if ctx.Err() != nil {
				return
			}
			if _, fresh := service.games.CachedPlace(ctx, place.PlaceID); fresh {
				continue
			}
			updated, found, err := service.games.PlaceByID(ctx, place.PlaceID)
			if err != nil {
				if ctx.Err() == nil {
					service.logs.Module("application.games").WarnContext(ctx, "favorite metadata refresh unavailable; keeping cached details", "operation", "favorite-metadata-refresh", "place_id", place.PlaceID)
				}
				continue
			}
			if found && ctx.Err() == nil {
				service.events.send("game:metadata-changed", updated)
			}
		}
	})
}

func (service *Service) refreshAccountMetadata(accountID, userID int64) {
	service.refreshMetadata(fmt.Sprintf("user-%d", userID), func(ctx context.Context) {
		identity, err := service.users.Profile(ctx, userID)
		if err != nil {
			if ctx.Err() == nil {
				service.logger.WarnContext(ctx, "saved account names unavailable; retry the account list later", "operation", "account-name-recovery", "account_id", accountID)
			}
			return
		}
		if err := service.accountRepository.RestoreNames(ctx, accountID, identity); err != nil {
			service.logger.WarnContext(ctx, "saved account names could not be restored", "operation", "account-name-recovery", "account_id", accountID, "error", err)
			return
		}
		if ctx.Err() == nil {
			service.events.SessionChanged(accountID)
		}
	})
}

// refreshAccountModeration validates a moderated account whose punishment is
// missing from the cache, which reads and caches the punishment again.
func (service *Service) refreshAccountModeration(accountID, userID int64) {
	service.refreshMetadata(fmt.Sprintf("moderation-%d", userID), func(ctx context.Context) {
		if _, err := service.ValidateAccount(ctx, accountID); err != nil && ctx.Err() == nil {
			service.logger.WarnContext(ctx, "account moderation could not be refreshed; retry the account list later", "operation", "account-moderation-refresh", "account_id", accountID, "error", err)
		}
	})
}
