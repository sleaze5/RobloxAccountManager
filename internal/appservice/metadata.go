package appservice

import (
	"context"
	"fmt"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
)

func (service *Service) refreshMetadata(key string, refresh func(context.Context)) {
	service.mu.Lock()
	parent := service.gamesContext
	service.mu.Unlock()
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
		if service.users.CachedProfileFresh(ctx, userID) {
			return
		}
		if _, err := service.users.Profile(ctx, userID); err != nil {
			if ctx.Err() == nil {
				service.logger.WarnContext(ctx, "public account metadata refresh unavailable; keeping cached names", "operation", "account-metadata-refresh", "account_id", accountID)
			}
			return
		}
		if ctx.Err() == nil {
			service.events.SessionChanged(accountID)
		}
	})
}
