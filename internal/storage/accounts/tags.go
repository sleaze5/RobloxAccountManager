package accountstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	accountdomain "github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/vault"
)

func (repository *Repository) CreateTag(ctx context.Context, input string) (accountdomain.TagView, error) {
	name, err := accountdomain.ValidateTagName(input)
	if err != nil {
		return accountdomain.TagView{}, err
	}
	database, release, err := repository.leaseDatabase(ctx)
	if err != nil {
		return accountdomain.TagView{}, err
	}
	defer release()
	now := time.Now().UTC().UnixMilli()
	var id int64
	err = vault.RunWriteTransaction(ctx, database, "tag creation", func(transaction *sql.Tx) (bool, error) {
		insertion, err := transaction.ExecContext(ctx, `INSERT INTO tags(name, kind, created_at_ms, updated_at_ms) VALUES(?, 'custom', ?, ?)`, name, now, now)
		if err != nil {
			if isConstraint(err) {
				return false, accountdomain.ErrDuplicateTag
			}
			return false, fmt.Errorf("create account tag: %w", err)
		}
		if id, err = insertion.LastInsertId(); err != nil {
			return false, fmt.Errorf("read created tag ID: %w", err)
		}
		return true, nil
	})
	if err != nil {
		return accountdomain.TagView{}, err
	}
	repository.logger.InfoContext(ctx, "account tag created", "operation", "tag-create", "tag_id", id)
	return accountdomain.TagView{ID: id, Name: name, Kind: accountdomain.TagKindCustom}, nil
}

func (repository *Repository) RemoveTag(ctx context.Context, tagID int64) error {
	if tagID <= 0 {
		return accountdomain.ErrTagNotFound
	}
	database, release, err := repository.leaseDatabase(ctx)
	if err != nil {
		return err
	}
	defer release()
	err = vault.RunWriteTransaction(ctx, database, "tag removal", func(transaction *sql.Tx) (bool, error) {
		deletion, err := transaction.ExecContext(ctx, `DELETE FROM tags WHERE id = ? AND kind = 'custom'`, tagID)
		if err != nil {
			return false, fmt.Errorf("remove account tag: %w", err)
		}
		changed, err := deletion.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("count removed tags: %w", err)
		}
		if changed == 0 {
			return false, accountdomain.ErrTagNotFound
		}
		return true, nil
	})
	if err != nil {
		return err
	}
	repository.logger.InfoContext(ctx, "account tag removed", "operation", "tag-remove", "tag_id", tagID)
	return nil
}

func (repository *Repository) ListTags(ctx context.Context) ([]accountdomain.TagView, error) {
	database, release, err := repository.leaseDatabase(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	rows, err := database.QueryContext(ctx, `SELECT id, name, kind FROM tags ORDER BY CASE kind WHEN 'favorite' THEN 0 ELSE 1 END, name COLLATE NOCASE, id`)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()
	tags := []accountdomain.TagView{}
	for rows.Next() {
		var tag accountdomain.TagView
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Kind); err != nil {
			return nil, fmt.Errorf("read tag: %w", err)
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read tags: %w", err)
	}
	return tags, nil
}

func (repository *Repository) SetAccountsTag(ctx context.Context, accountIDs []int64, tagID int64, selected bool) ([]accountdomain.AccountView, error) {
	if len(accountIDs) == 0 {
		return nil, accountdomain.ErrNotFound
	}
	for _, accountID := range accountIDs {
		if accountID <= 0 {
			return nil, accountdomain.ErrNotFound
		}
	}
	if tagID <= 0 {
		return nil, accountdomain.ErrTagNotFound
	}
	database, release, err := repository.leaseDatabase(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	err = vault.RunWriteTransaction(ctx, database, "tag assignment", func(transaction *sql.Tx) (bool, error) {
		var exists int
		if err := transaction.QueryRowContext(ctx, `SELECT 1 FROM tags WHERE id = ?`, tagID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
			return false, accountdomain.ErrTagNotFound
		} else if err != nil {
			return false, fmt.Errorf("find tag: %w", err)
		}
		return assignAccountsTag(ctx, transaction, accountIDs, tagID, selected, time.Now().UTC().UnixMilli())
	})
	if err != nil {
		return nil, err
	}
	views := make([]accountdomain.AccountView, 0, len(accountIDs))
	for _, accountID := range accountIDs {
		view, err := repository.GetView(ctx, accountID)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func loadAccountTags(ctx context.Context, database *sql.DB, views []accountdomain.AccountView) error {
	if len(views) == 0 {
		return nil
	}
	indices := make(map[int64]int, len(views))
	placeholders := make([]string, len(views))
	arguments := make([]any, len(views))
	for index := range views {
		views[index].Tags = []accountdomain.TagView{}
		indices[views[index].ID] = index
		placeholders[index] = "?"
		arguments[index] = views[index].ID
	}
	rows, err := database.QueryContext(ctx, accountTagsQuery+`
WHERE account_tags.account_id IN (`+strings.Join(placeholders, ",")+`)
ORDER BY account_tags.account_id, CASE tags.kind WHEN 'favorite' THEN 0 ELSE 1 END, tags.name COLLATE NOCASE, tags.id`, arguments...)
	if err != nil {
		return fmt.Errorf("list account tags: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var accountID int64
		var tag accountdomain.TagView
		if err := rows.Scan(&accountID, &tag.ID, &tag.Name, &tag.Kind); err != nil {
			return fmt.Errorf("read account tag: %w", err)
		}
		if index, ok := indices[accountID]; ok {
			views[index].Tags = append(views[index].Tags, tag)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read account tags: %w", err)
	}
	return nil
}

func listAccountTags(ctx context.Context, database interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, accountID int64) ([]accountdomain.TagView, error) {
	rows, err := database.QueryContext(ctx, `SELECT tags.id, tags.name, tags.kind FROM account_tags JOIN tags ON tags.id = account_tags.tag_id WHERE account_tags.account_id = ? ORDER BY CASE tags.kind WHEN 'favorite' THEN 0 ELSE 1 END, tags.name COLLATE NOCASE, tags.id`, accountID)
	if err != nil {
		return nil, fmt.Errorf("list account tags: %w", err)
	}
	defer rows.Close()
	tags := []accountdomain.TagView{}
	for rows.Next() {
		var tag accountdomain.TagView
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Kind); err != nil {
			return nil, fmt.Errorf("read account tag: %w", err)
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read account tags: %w", err)
	}
	return tags, nil
}

const accountTagsQuery = `
SELECT account_tags.account_id, tags.id, tags.name, tags.kind
FROM account_tags JOIN tags ON tags.id = account_tags.tag_id`

func assignAccountsTag(ctx context.Context, transaction *sql.Tx, accountIDs []int64, tagID int64, selected bool, now int64) (bool, error) {
	var exists int
	var err error
	anyChanged := false
	for _, accountID := range accountIDs {
		if err := transaction.QueryRowContext(ctx, `SELECT 1 FROM accounts WHERE id = ?`, accountID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
			return false, accountdomain.ErrNotFound
		} else if err != nil {
			return false, fmt.Errorf("find tagged account: %w", err)
		}
		var change sql.Result
		if selected {
			change, err = transaction.ExecContext(ctx, `INSERT INTO account_tags(account_id, tag_id, created_at_ms) VALUES(?, ?, ?) ON CONFLICT(account_id, tag_id) DO NOTHING`, accountID, tagID, now)
		} else {
			change, err = transaction.ExecContext(ctx, `DELETE FROM account_tags WHERE account_id = ? AND tag_id = ?`, accountID, tagID)
		}
		if err != nil {
			return false, fmt.Errorf("update account tag: %w", err)
		}
		changed, err := change.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("count updated account tags: %w", err)
		}
		if changed > 0 {
			anyChanged = true
			if _, err := transaction.ExecContext(ctx, `UPDATE accounts SET updated_at_ms = ? WHERE id = ?`, now, accountID); err != nil {
				return false, fmt.Errorf("update tagged account: %w", err)
			}
		}
	}
	return anyChanged, nil
}
