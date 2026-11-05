package gamestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
)

var (
	ErrPrivateServerSaved    = errors.New("private server link is already saved")
	ErrPrivateServerNotSaved = errors.New("private server link is not saved")
)

func (repository *Repository) PrivateServers(ctx context.Context, placeID int64) ([]games.SavedPrivateServer, error) {
	return repository.listPrivateServers(ctx, `SELECT id, place_id, name, access_code, link_code, created_at_ms FROM saved_private_servers WHERE place_id = ? ORDER BY id`, placeID)
}

func (repository *Repository) AllPrivateServers(ctx context.Context) ([]games.SavedPrivateServer, error) {
	return repository.listPrivateServers(ctx, `SELECT id, place_id, name, access_code, link_code, created_at_ms FROM saved_private_servers ORDER BY id`)
}

func (repository *Repository) listPrivateServers(ctx context.Context, query string, args ...any) ([]games.SavedPrivateServer, error) {
	db, release, err := repository.provider.Lease(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list saved private servers: %w", err)
	}
	defer rows.Close()
	servers := []games.SavedPrivateServer{}
	for rows.Next() {
		var server games.SavedPrivateServer
		if err := rows.Scan(&server.ID, &server.PlaceID, &server.Name, &server.AccessCode, &server.LinkCode, &server.CreatedAtMs); err != nil {
			return nil, fmt.Errorf("read saved private server: %w", err)
		}
		server.LaunchLink = games.PrivateServerLaunchLink(server.PlaceID, server.AccessCode, server.LinkCode)
		servers = append(servers, server)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read saved private servers: %w", err)
	}
	return servers, nil
}

func (repository *Repository) SavePrivateServer(ctx context.Context, server games.SavedPrivateServer) (games.SavedPrivateServer, error) {
	db, release, err := repository.provider.Lease(ctx)
	if err != nil {
		return games.SavedPrivateServer{}, err
	}
	defer release()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return games.SavedPrivateServer{}, fmt.Errorf("begin saved private server insert: %w", err)
	}
	defer tx.Rollback()
	server.CreatedAtMs = time.Now().UnixMilli()
	result, err := tx.ExecContext(ctx, `INSERT INTO saved_private_servers(place_id, name, access_code, link_code, created_at_ms) VALUES(?, ?, ?, ?, ?) ON CONFLICT(place_id, access_code) DO NOTHING`, server.PlaceID, server.Name, server.AccessCode, server.LinkCode, server.CreatedAtMs)
	if err != nil {
		return games.SavedPrivateServer{}, fmt.Errorf("insert saved private server: %w", err)
	}
	if changed, err := result.RowsAffected(); err != nil {
		return games.SavedPrivateServer{}, fmt.Errorf("count saved private servers: %w", err)
	} else if changed == 0 {
		return games.SavedPrivateServer{}, ErrPrivateServerSaved
	}
	if server.ID, err = result.LastInsertId(); err != nil {
		return games.SavedPrivateServer{}, fmt.Errorf("read saved private server ID: %w", err)
	}
	if err := commitPrivateServers(ctx, tx); err != nil {
		return games.SavedPrivateServer{}, err
	}
	server.LaunchLink = games.PrivateServerLaunchLink(server.PlaceID, server.AccessCode, server.LinkCode)
	repository.logger.InfoContext(ctx, "private server link saved", "operation", "private-server-link-save", "place_id", server.PlaceID)
	return server, nil
}

func (repository *Repository) RemovePrivateServer(ctx context.Context, id int64) error {
	db, release, err := repository.provider.Lease(ctx)
	if err != nil {
		return err
	}
	defer release()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin saved private server removal: %w", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `DELETE FROM saved_private_servers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("remove saved private server: %w", err)
	}
	if changed, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("count removed private servers: %w", err)
	} else if changed == 0 {
		return ErrPrivateServerNotSaved
	}
	if err := commitPrivateServers(ctx, tx); err != nil {
		return err
	}
	repository.logger.InfoContext(ctx, "private server link removed", "operation", "private-server-link-remove", "saved_server_id", id)
	return nil
}

func commitPrivateServers(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `UPDATE vault_state SET storage_revision = storage_revision + 1 WHERE singleton = 1`); err != nil {
		return fmt.Errorf("update storage revision: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit saved private servers: %w", err)
	}
	return nil
}
