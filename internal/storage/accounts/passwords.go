package accountstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func (repository *Repository) GetPassword(ctx context.Context, accountID int64) (string, error) {
	db, release, err := repository.database(ctx)
	if err != nil {
		return "", fmt.Errorf("open vault for saved password: %w", err)
	}
	defer release()
	var password string
	if err := db.QueryRowContext(ctx, `SELECT password FROM account_sessions WHERE account_id = ?`, accountID).Scan(&password); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("read saved account password: %w", err)
	}
	return password, nil
}

func (repository *Repository) SetPassword(ctx context.Context, accountID int64, password string) error {
	if password == "" {
		return errors.New("saved account password cannot be empty")
	}
	db, release, err := repository.database(ctx)
	if err != nil {
		return fmt.Errorf("open vault to save password: %w", err)
	}
	defer release()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin account password update: %w", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE account_sessions SET password = ?, updated_at_ms = ? WHERE account_id = ?`, password, repository.now().UTC().UnixMilli(), accountID)
	if err != nil {
		return fmt.Errorf("save account password: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check saved account password update: %w", err)
	}
	if changed == 0 {
		return ErrNotFound
	}
	if err := bumpRevision(ctx, tx); err != nil {
		return fmt.Errorf("record account password revision: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit account password update: %w", err)
	}
	return nil
}
