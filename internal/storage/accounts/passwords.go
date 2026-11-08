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
	return vault.RunWriteTransaction(ctx, database, "account password update", func(transaction *sql.Tx) (bool, error) {
		update, err := transaction.ExecContext(ctx, `UPDATE account_sessions SET password = ?, updated_at_ms = ? WHERE account_id = ?`, password, time.Now().UTC().UnixMilli(), accountID)
		if err != nil {
			return false, fmt.Errorf("save account password: %w", err)
		}
		changed, err := update.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("check saved account password update: %w", err)
		}
		if changed == 0 {
			return false, accountdomain.ErrNotFound
		}
		return true, nil
	})
}
