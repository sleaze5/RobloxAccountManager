package accountstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	accountdomain "github.com/sleaze5/RobloxAccountManager/internal/accounts"
)

func (repository *Repository) CreateTag(ctx context.Context, input string) (TagView, error) {
	name, err := accountdomain.ValidateTagName(input)
	if err != nil {
		return TagView{}, err
	}
	db, release, err := repository.database(ctx)
	if err != nil {
		return TagView{}, err
	}
	defer release()
	now := repository.now().UTC().UnixMilli()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return TagView{}, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO tags(name, kind, created_at_ms, updated_at_ms) VALUES(?, 'custom', ?, ?)`, name, now, now)
	if err != nil {
		if isConstraint(err) {
			return TagView{}, ErrDuplicateTag
		}
		return TagView{}, fmt.Errorf("create account tag: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return TagView{}, err
	}
	if err := bumpRevision(ctx, tx); err != nil {
		return TagView{}, err
	}
	if err := tx.Commit(); err != nil {
		return TagView{}, err
	}
	repository.logger.Info("account tag created", "tag_id", id)
	return TagView{ID: id, Name: name, Kind: TagKindCustom}, nil
}

func (repository *Repository) RemoveTag(ctx context.Context, tagID int64) error {
	if tagID <= 0 {
		return ErrTagNotFound
	}
	db, release, err := repository.database(ctx)
	if err != nil {
		return err
	}
	defer release()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE id = ? AND kind = 'custom'`, tagID)
	if err != nil {
		return fmt.Errorf("remove account tag: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		return ErrTagNotFound
	}
	if err := bumpRevision(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	repository.logger.Info("account tag removed", "tag_id", tagID)
	return nil
}

func (repository *Repository) ListTags(ctx context.Context) ([]TagView, error) {
	db, release, err := repository.database(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	rows, err := db.QueryContext(ctx, `SELECT id, name, kind FROM tags ORDER BY CASE kind WHEN 'favorite' THEN 0 ELSE 1 END, name COLLATE NOCASE, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := []TagView{}
	for rows.Next() {
		var tag TagView
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Kind); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

// SetAccountsTag adds or removes one tag on every listed account in a single transaction.
func (repository *Repository) SetAccountsTag(ctx context.Context, accountIDs []int64, tagID int64, selected bool) ([]AccountView, error) {
	if len(accountIDs) == 0 {
		return nil, ErrNotFound
	}
	for _, accountID := range accountIDs {
		if accountID <= 0 {
			return nil, ErrNotFound
		}
	}
	if tagID <= 0 {
		return nil, ErrTagNotFound
	}
	db, release, err := repository.database(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM tags WHERE id = ?`, tagID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTagNotFound
	} else if err != nil {
		return nil, err
	}
	now := repository.now().UTC().UnixMilli()
	anyChanged := false
	for _, accountID := range accountIDs {
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM accounts WHERE id = ?`, accountID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		} else if err != nil {
			return nil, err
		}
		var result sql.Result
		if selected {
			result, err = tx.ExecContext(ctx, `INSERT INTO account_tags(account_id, tag_id, created_at_ms) VALUES(?, ?, ?) ON CONFLICT(account_id, tag_id) DO NOTHING`, accountID, tagID, now)
		} else {
			result, err = tx.ExecContext(ctx, `DELETE FROM account_tags WHERE account_id = ? AND tag_id = ?`, accountID, tagID)
		}
		if err != nil {
			return nil, err
		}
		if changed, _ := result.RowsAffected(); changed > 0 {
			anyChanged = true
			if _, err := tx.ExecContext(ctx, `UPDATE accounts SET updated_at_ms = ? WHERE id = ?`, now, accountID); err != nil {
				return nil, err
			}
		}
	}
	if anyChanged {
		if err := bumpRevision(ctx, tx); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	views := make([]AccountView, 0, len(accountIDs))
	for _, accountID := range accountIDs {
		view, err := repository.GetView(ctx, accountID)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func loadAccountTags(ctx context.Context, db *sql.DB, views []AccountView) error {
	if len(views) == 0 {
		return nil
	}
	indices := make(map[int64]int, len(views))
	placeholders := make([]string, len(views))
	arguments := make([]any, len(views))
	for index := range views {
		views[index].Tags = []TagView{}
		indices[views[index].ID] = index
		placeholders[index] = "?"
		arguments[index] = views[index].ID
	}
	rows, err := db.QueryContext(ctx, `
SELECT account_tags.account_id, tags.id, tags.name, tags.kind
FROM account_tags JOIN tags ON tags.id = account_tags.tag_id
WHERE account_tags.account_id IN (`+strings.Join(placeholders, ",")+`)
ORDER BY account_tags.account_id, CASE tags.kind WHEN 'favorite' THEN 0 ELSE 1 END, tags.name COLLATE NOCASE, tags.id`, arguments...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var accountID int64
		var tag TagView
		if err := rows.Scan(&accountID, &tag.ID, &tag.Name, &tag.Kind); err != nil {
			return err
		}
		if index, ok := indices[accountID]; ok {
			views[index].Tags = append(views[index].Tags, tag)
		}
	}
	return rows.Err()
}

func listAccountTags(ctx context.Context, db *sql.DB, accountID int64) ([]TagView, error) {
	rows, err := db.QueryContext(ctx, `SELECT tags.id, tags.name, tags.kind FROM account_tags JOIN tags ON tags.id = account_tags.tag_id WHERE account_tags.account_id = ? ORDER BY CASE tags.kind WHEN 'favorite' THEN 0 ELSE 1 END, tags.name COLLATE NOCASE, tags.id`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := []TagView{}
	for rows.Next() {
		var tag TagView
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Kind); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}
