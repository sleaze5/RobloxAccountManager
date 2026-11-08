package vault

import (
	"context"
	"database/sql"
	"fmt"
)

// RunWriteTransaction runs write in one transaction and commits it. When
// write reports that it changed vault data, the storage revision increases in
// the same transaction, because MaybeCreateBackup compares that revision with
// the newest backup. An error from write rolls the transaction back and is
// returned unchanged, so callers can match sentinel errors.
func RunWriteTransaction(ctx context.Context, database *sql.DB, operation string, write func(*sql.Tx) (bool, error)) error {
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin %s: %w", operation, err)
	}
	defer transaction.Rollback()
	changed, err := write(transaction)
	if err != nil {
		return err
	}
	if changed {
		if err := bumpRevision(ctx, transaction); err != nil {
			return err
		}
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit %s: %w", operation, err)
	}
	return nil
}
