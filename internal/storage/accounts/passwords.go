package accountstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	accountdomain "github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/vault"
)

func (repository *Repository) GetPassword(ctx context.Context, accountID int64) (string, error) {
	database, release, err := repository.leaseDatabase(ctx)
	if err != nil {
		return "", fmt.Errorf("open vault for saved password: %w", err)
	}
	defer release()
	var password string
	if err := database.QueryRowContext(ctx, `SELECT password FROM account_sessions WHERE account_id = ?`, accountID).Scan(&password); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", accountdomain.ErrNotFound
		}
		return "", fmt.Errorf("read saved account password: %w", err)
	}
	return password, nil
}

func (repository *Repository) SetPassword(ctx context.Context, accountID int64, password string) error {
	if password == "" {
		return errors.New("saved account password cannot be empty")
	}
	database, release, err := repository.leaseDatabase(ctx)
	if err != nil {
		return fmt.Errorf("open vault to save password: %w", err)
	}
	defer release()
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin account password update: %w", err)
	}
	defer transaction.Rollback()
	update, err := transaction.ExecContext(ctx, `UPDATE account_sessions SET password = ?, updated_at_ms = ? WHERE account_id = ?`, password, time.Now().UTC().UnixMilli(), accountID)
	if err != nil {
		return fmt.Errorf("save account password: %w", err)
	}
	changed, err := update.RowsAffected()
	if err != nil {
		return fmt.Errorf("check saved account password update: %w", err)
	}
	if changed == 0 {
		return accountdomain.ErrNotFound
	}
	if err := vault.BumpRevision(ctx, transaction); err != nil {
		return fmt.Errorf("record account password revision: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit account password update: %w", err)
	}
	return nil
}
