package gamestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
)

var ErrNotFavorite = errors.New("place is not a favorite")

type DatabaseProvider interface {
	Lease(context.Context) (*sql.DB, func(), error)
}

type Repository struct {
	provider DatabaseProvider
	logger   *slog.Logger
}

func NewRepository(provider DatabaseProvider, logger *slog.Logger) *Repository {
	return &Repository{provider: provider, logger: logger}
}

func (repository *Repository) List(ctx context.Context) ([]games.Place, error) {
	db, release, err := repository.provider.Lease(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	rows, err := db.QueryContext(ctx, `SELECT place_id, universe_id, name, creator_id, creator_name, creator_type, creator_verified, icon_url, nickname FROM favorite_places ORDER BY display_order, place_id`)
	if err != nil {
		return nil, fmt.Errorf("list favorite places: %w", err)
	}
	defer rows.Close()
	places := make([]games.Place, 0)
	for rows.Next() {
		var place games.Place
		if err := rows.Scan(&place.PlaceID, &place.UniverseID, &place.Name, &place.CreatorID, &place.CreatorName, &place.CreatorType, &place.CreatorVerified, &place.IconURL, &place.Nickname); err != nil {
			return nil, fmt.Errorf("read favorite place: %w", err)
		}
		places = append(places, place)
	}
	return places, rows.Err()
}

func (repository *Repository) Save(ctx context.Context, place games.Place) error {
	if place.PlaceID <= 0 || place.UniverseID <= 0 || place.Name == "" {
		return fmt.Errorf("invalid favorite place")
	}
	return repository.write(ctx, place.PlaceID, "favorite-place-save", false, `INSERT INTO favorite_places(place_id, universe_id, name, creator_id, creator_name, creator_type, creator_verified, icon_url, display_order, created_at_ms)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, (SELECT COALESCE(MIN(display_order) - 1, 0) FROM favorite_places), ?) ON CONFLICT(place_id) DO UPDATE SET
		universe_id = excluded.universe_id, name = excluded.name, creator_id = excluded.creator_id,
		creator_name = excluded.creator_name, creator_type = excluded.creator_type, creator_verified = excluded.creator_verified, icon_url = excluded.icon_url`,
		place.PlaceID, place.UniverseID, place.Name, place.CreatorID, place.CreatorName, place.CreatorType, place.CreatorVerified, place.IconURL, time.Now().UnixMilli())
}

func (repository *Repository) SetNickname(ctx context.Context, placeID int64, nickname string) error {
	return repository.write(ctx, placeID, "favorite-place-nickname", true, `UPDATE favorite_places SET nickname = ? WHERE place_id = ?`, nickname, placeID)
}

func (repository *Repository) Reorder(ctx context.Context, placeIDs []int64) error {
	db, release, err := repository.provider.Lease(ctx)
	if err != nil {
		return err
	}
	defer release()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM favorite_places`).Scan(&count); err != nil {
		return err
	}
	if count != len(placeIDs) {
		return ErrNotFavorite
	}
	for order, placeID := range placeIDs {
		result, err := tx.ExecContext(ctx, `UPDATE favorite_places SET display_order = ? WHERE place_id = ?`, order, placeID)
		if err != nil {
			return err
		}
		if changed, err := result.RowsAffected(); err != nil {
			return err
		} else if changed == 0 {
			return ErrNotFavorite
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE vault_state SET storage_revision = storage_revision + 1 WHERE singleton = 1`); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	repository.logger.InfoContext(ctx, "favorite places reordered", "operation", "favorite-places-reorder", "count", len(placeIDs))
	return nil
}

func (repository *Repository) Remove(ctx context.Context, placeID int64) error {
	return repository.write(ctx, placeID, "favorite-place-remove", false, `DELETE FROM favorite_places WHERE place_id = ?`, placeID)
}

func (repository *Repository) write(ctx context.Context, placeID int64, operation string, existing bool, query string, args ...any) error {
	db, release, err := repository.provider.Lease(ctx)
	if err != nil {
		return err
	}
	defer release()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	if existing {
		if changed, err := result.RowsAffected(); err != nil {
			return err
		} else if changed == 0 {
			return ErrNotFavorite
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE vault_state SET storage_revision = storage_revision + 1 WHERE singleton = 1`); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	repository.logger.InfoContext(ctx, "favorite places updated", "operation", operation, "place_id", placeID)
	return nil
}
