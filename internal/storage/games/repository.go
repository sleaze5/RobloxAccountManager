package gamestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/caches"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/vault"
)

var ErrNotBookmarked = errors.New("place is not bookmarked")

// Repository stores the bookmarks and the saved private servers. The vault
// keeps bookmarks in the favorite_places table, the name of bookmarks in
// earlier versions.
type Repository struct {
	vault  *vault.Manager
	cache  *caches.Store
	logger *slog.Logger
}

func NewRepository(vaultManager *vault.Manager, logger *slog.Logger, cache *caches.Store) *Repository {
	return &Repository{vault: vaultManager, logger: logger, cache: cache}
}

func (repository *Repository) List(ctx context.Context) ([]games.Place, error) {
	database, release, err := repository.vault.Lease(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	rows, err := database.QueryContext(ctx, `SELECT place_id, nickname FROM favorite_places ORDER BY display_order, place_id`)
	if err != nil {
		return nil, fmt.Errorf("list bookmarks: %w", err)
	}
	defer rows.Close()
	places := make([]games.Place, 0)
	for rows.Next() {
		var place games.Place
		if err := rows.Scan(&place.PlaceID, &place.Nickname); err != nil {
			return nil, fmt.Errorf("read bookmark: %w", err)
		}
		metadata, _ := repository.cache.Place(ctx, place.PlaceID)
		metadata.Nickname = place.Nickname
		places = append(places, metadata)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read bookmarks: %w", err)
	}
	return places, nil
}

func (repository *Repository) Save(ctx context.Context, place games.Place) error {
	if place.PlaceID <= 0 || place.UniverseID <= 0 || place.Name == "" {
		return errors.New("invalid bookmark")
	}
	return repository.write(ctx, place.PlaceID, "bookmark-add", false, `INSERT INTO favorite_places(place_id, display_order, created_at_ms) VALUES(?, (SELECT COALESCE(MIN(display_order) - 1, 0) FROM favorite_places), ?) ON CONFLICT(place_id) DO NOTHING`, place.PlaceID, time.Now().UnixMilli())
}

func (repository *Repository) SetNickname(ctx context.Context, placeID int64, nickname string) error {
	return repository.write(ctx, placeID, "bookmark-nickname", true, `UPDATE favorite_places SET nickname = ? WHERE place_id = ?`, nickname, placeID)
}

func (repository *Repository) Reorder(ctx context.Context, placeIDs []int64) error {
	database, release, err := repository.vault.Lease(ctx)
	if err != nil {
		return err
	}
	defer release()
	err = vault.RunWriteTransaction(ctx, database, "bookmark reorder", func(transaction *sql.Tx) (bool, error) {
		var count int
		if err := transaction.QueryRowContext(ctx, `SELECT COUNT(*) FROM favorite_places`).Scan(&count); err != nil {
			return false, fmt.Errorf("count bookmarks: %w", err)
		}
		if count != len(placeIDs) {
			return false, ErrNotBookmarked
		}
		for order, placeID := range placeIDs {
			update, err := transaction.ExecContext(ctx, `UPDATE favorite_places SET display_order = ? WHERE place_id = ?`, order, placeID)
			if err != nil {
				return false, fmt.Errorf("reorder bookmark: %w", err)
			}
			if changed, err := update.RowsAffected(); err != nil {
				return false, fmt.Errorf("count reordered bookmarks: %w", err)
			} else if changed == 0 {
				return false, ErrNotBookmarked
			}
		}
		return true, nil
	})
	if err != nil {
		return err
	}
	repository.logger.InfoContext(ctx, "bookmarks reordered", "operation", "bookmarks-reorder", "count", len(placeIDs))
	return nil
}

func (repository *Repository) Remove(ctx context.Context, placeID int64) error {
	return repository.write(ctx, placeID, "bookmark-remove", false, `DELETE FROM favorite_places WHERE place_id = ?`, placeID)
}

func (repository *Repository) write(ctx context.Context, placeID int64, operation string, existing bool, query string, arguments ...any) error {
	database, release, err := repository.vault.Lease(ctx)
	if err != nil {
		return err
	}
	defer release()
	var wasChanged bool
	err = vault.RunWriteTransaction(ctx, database, "bookmark update", func(transaction *sql.Tx) (bool, error) {
		change, err := transaction.ExecContext(ctx, query, arguments...)
		if err != nil {
			return false, fmt.Errorf("update bookmark: %w", err)
		}
		changed, err := change.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("count updated bookmarks: %w", err)
		}
		if changed == 0 && existing {
			return false, ErrNotBookmarked
		}
		wasChanged = changed > 0
		return wasChanged, nil
	})
	if err != nil || !wasChanged {
		return err
	}
	repository.logger.InfoContext(ctx, "bookmarks updated", "operation", operation, "place_id", placeID)
	return nil
}
