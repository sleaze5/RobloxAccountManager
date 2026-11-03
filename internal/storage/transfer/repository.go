package transferstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	accountdomain "github.com/sleaze5/RobloxAccountManager/internal/accounts"
)

type DatabaseProvider interface {
	Lease(context.Context) (*sql.DB, func(), error)
}

type Repository struct {
	provider DatabaseProvider
	now      func() time.Time
	logger   *slog.Logger
}

type Replacement struct {
	Identity  accountdomain.Identity
	Cookie    string
	ExpiresAt *time.Time
}

type MergeInput struct {
	Accounts       []Account
	Replacements   map[int64]Replacement
	FavoritePlaces []FavoritePlace
	Settings       *PendingSettings
}

type PendingSettings struct {
	Data     json.RawMessage
	Sections []string
}

type MergeResult struct {
	Added          []int64
	Updated        []int64
	Replaced       []accountdomain.SessionRecord
	FavoritePlaces int
}

func NewRepository(provider DatabaseProvider, logger *slog.Logger) *Repository {
	return &Repository{provider: provider, now: time.Now, logger: logger}
}

func (repository *Repository) Read(ctx context.Context) (Payload, error) {
	db, release, err := repository.provider.Lease(ctx)
	if err != nil {
		return Payload{}, err
	}
	defer release()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Payload{}, fmt.Errorf("begin export read: %w", err)
	}
	defer tx.Rollback()
	payload := Payload{}
	if payload.Accounts, err = readAccounts(ctx, tx); err != nil {
		return Payload{}, err
	}
	if payload.Tags, err = readCustomTags(ctx, tx); err != nil {
		return Payload{}, err
	}
	if payload.FavoritePlaces, err = readFavoritePlaces(ctx, tx); err != nil {
		return Payload{}, err
	}
	if err := tx.Commit(); err != nil {
		return Payload{}, fmt.Errorf("finish export read: %w", err)
	}
	return payload, nil
}

func readAccounts(ctx context.Context, tx *sql.Tx) ([]Account, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT a.id, a.roblox_user_id, a.username, a.display_name, a.created_at_ms,
       s.roblosecurity, s.cookie_expires_at_ms, s.password
FROM accounts a JOIN account_sessions s ON s.account_id = a.id
ORDER BY a.display_order, a.id`)
	if err != nil {
		return nil, fmt.Errorf("list exported accounts: %w", err)
	}
	defer rows.Close()
	accounts := []Account{}
	indices := map[int64]int{}
	for rows.Next() {
		var id int64
		var expires sql.NullInt64
		account := Account{Tags: []string{}}
		if err := rows.Scan(&id, &account.RobloxUserID, &account.Username, &account.DisplayName, &account.CreatedAtMS, &account.Cookie, &expires, &account.Password); err != nil {
			return nil, fmt.Errorf("read exported account: %w", err)
		}
		if expires.Valid {
			account.CookieExpiresAtMS = &expires.Int64
		}
		indices[id] = len(accounts)
		accounts = append(accounts, account)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read exported accounts: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close exported accounts: %w", err)
	}
	tagRows, err := tx.QueryContext(ctx, `SELECT account_tags.account_id, tags.name, tags.kind FROM account_tags JOIN tags ON tags.id = account_tags.tag_id ORDER BY tags.name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("list exported account tags: %w", err)
	}
	defer tagRows.Close()
	for tagRows.Next() {
		var accountID int64
		var name string
		var kind accountdomain.TagKind
		if err := tagRows.Scan(&accountID, &name, &kind); err != nil {
			return nil, fmt.Errorf("read exported account tag: %w", err)
		}
		index, ok := indices[accountID]
		switch {
		case !ok:
		case kind == accountdomain.TagKindFavorite:
			accounts[index].Favorite = true
		default:
			accounts[index].Tags = append(accounts[index].Tags, name)
		}
	}
	if err := tagRows.Err(); err != nil {
		return nil, fmt.Errorf("read exported account tags: %w", err)
	}
	return accounts, nil
}

func readCustomTags(ctx context.Context, tx *sql.Tx) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT name FROM tags WHERE kind = 'custom' ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("list exported tags: %w", err)
	}
	defer rows.Close()
	tags := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("read exported tag: %w", err)
		}
		tags = append(tags, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read exported tags: %w", err)
	}
	return tags, nil
}

func readFavoritePlaces(ctx context.Context, tx *sql.Tx) ([]FavoritePlace, error) {
	rows, err := tx.QueryContext(ctx, `SELECT place_id, nickname FROM favorite_places ORDER BY display_order, place_id`)
	if err != nil {
		return nil, fmt.Errorf("list exported favorite places: %w", err)
	}
	defer rows.Close()
	places := []FavoritePlace{}
	for rows.Next() {
		var place FavoritePlace
		if err := rows.Scan(&place.PlaceID, &place.Nickname); err != nil {
			return nil, fmt.Errorf("read exported favorite place: %w", err)
		}
		places = append(places, place)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read exported favorite places: %w", err)
	}
	return places, nil
}

// Replace keeps new row IDs above the removed ones, so account and tag IDs that the
// application still holds for removed rows never point at imported rows.
func (repository *Repository) Replace(ctx context.Context, payload Payload, settings *PendingSettings) ([]int64, error) {
	db, release, err := repository.provider.Lease(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin vault replacement: %w", err)
	}
	defer tx.Rollback()
	removed, err := accountIDs(ctx, tx)
	if err != nil {
		return nil, err
	}
	var nextAccountID, nextTagID int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) + 1 FROM accounts`).Scan(&nextAccountID); err != nil {
		return nil, fmt.Errorf("read account ID range: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) + 1 FROM tags`).Scan(&nextTagID); err != nil {
		return nil, fmt.Errorf("read tag ID range: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM accounts; DELETE FROM tags WHERE kind = 'custom'; DELETE FROM favorite_places;`); err != nil {
		return nil, fmt.Errorf("clear vault data: %w", err)
	}
	if err := insertPayload(ctx, tx, payload, nextAccountID, nextTagID, repository.now().UTC().UnixMilli()); err != nil {
		return nil, err
	}
	if err := storePendingSettings(ctx, tx, settings); err != nil {
		return nil, err
	}
	if err := bumpRevision(ctx, tx); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit vault replacement: %w", err)
	}
	repository.logger.Info("vault data replaced", "operation", "vault-import", "removed_accounts", len(removed), "accounts", len(payload.Accounts), "favorite_places", len(payload.FavoritePlaces))
	return removed, nil
}

func insertPayload(ctx context.Context, tx *sql.Tx, payload Payload, nextAccountID, nextTagID, now int64) error {
	tagIDs, err := favoriteTagIDs(ctx, tx)
	if err != nil {
		return err
	}
	for _, name := range payload.Tags {
		if _, err := tx.ExecContext(ctx, `INSERT INTO tags(id, name, kind, created_at_ms, updated_at_ms) VALUES(?, ?, 'custom', ?, ?)`, nextTagID, name, now, now); err != nil {
			return fmt.Errorf("insert imported tag: %w", err)
		}
		tagIDs[name] = nextTagID
		nextTagID++
	}
	for order, account := range payload.Accounts {
		if err := insertAccount(ctx, tx, nextAccountID, int64(order), account, now); err != nil {
			return err
		}
		if _, err := assignTags(ctx, tx, nextAccountID, account, tagIDs, now); err != nil {
			return err
		}
		nextAccountID++
	}
	for order, place := range payload.FavoritePlaces {
		if _, err := tx.ExecContext(ctx, `INSERT INTO favorite_places(place_id, nickname, display_order, created_at_ms) VALUES(?, ?, ?, ?)`, place.PlaceID, place.Nickname, order, now); err != nil {
			return fmt.Errorf("insert imported favorite place: %w", err)
		}
	}
	return nil
}

func (repository *Repository) Merge(ctx context.Context, input MergeInput) (MergeResult, error) {
	db, release, err := repository.provider.Lease(ctx)
	if err != nil {
		return MergeResult{}, err
	}
	defer release()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return MergeResult{}, fmt.Errorf("begin vault merge: %w", err)
	}
	defer tx.Rollback()
	now := repository.now().UTC().UnixMilli()
	tagIDs, err := favoriteTagIDs(ctx, tx)
	if err != nil {
		return MergeResult{}, err
	}
	result := MergeResult{}
	for _, account := range input.Accounts {
		for _, name := range account.Tags {
			if tagIDs[name], err = ensureTag(ctx, tx, name, now); err != nil {
				return MergeResult{}, err
			}
		}
		if err := mergeAccount(ctx, tx, account, input.Replacements, tagIDs, now, &result); err != nil {
			return MergeResult{}, err
		}
	}
	for _, place := range input.FavoritePlaces {
		inserted, err := tx.ExecContext(ctx, `INSERT INTO favorite_places(place_id, nickname, display_order, created_at_ms) SELECT ?, ?, COALESCE(MAX(display_order) + 1, 0), ? FROM favorite_places WHERE true ON CONFLICT(place_id) DO NOTHING`, place.PlaceID, place.Nickname, now)
		if err != nil {
			return MergeResult{}, fmt.Errorf("insert imported favorite place: %w", err)
		}
		if changed, err := inserted.RowsAffected(); err != nil {
			return MergeResult{}, fmt.Errorf("count imported favorite places: %w", err)
		} else if changed > 0 {
			result.FavoritePlaces++
		}
	}
	if err := storePendingSettings(ctx, tx, input.Settings); err != nil {
		return MergeResult{}, err
	}
	if err := bumpRevision(ctx, tx); err != nil {
		return MergeResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return MergeResult{}, fmt.Errorf("commit vault merge: %w", err)
	}
	repository.logger.Info("vault data merged", "operation", "vault-import", "added_accounts", len(result.Added), "replaced_cookies", len(result.Replaced), "updated_accounts", len(result.Updated), "favorite_places", result.FavoritePlaces)
	return result, nil
}

func mergeAccount(ctx context.Context, tx *sql.Tx, account Account, replacements map[int64]Replacement, tagIDs map[string]int64, now int64, result *MergeResult) error {
	var accountID int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM accounts WHERE roblox_user_id = ?`, account.RobloxUserID).Scan(&accountID)
	if errors.Is(err, sql.ErrNoRows) {
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) + 1 FROM accounts`).Scan(&accountID); err != nil {
			return fmt.Errorf("read next account ID: %w", err)
		}
		var order int64
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(display_order) + 1, 0) FROM accounts`).Scan(&order); err != nil {
			return fmt.Errorf("read next account position: %w", err)
		}
		if err := insertAccount(ctx, tx, accountID, order, account, now); err != nil {
			return err
		}
		if _, err := assignTags(ctx, tx, accountID, account, tagIDs, now); err != nil {
			return err
		}
		result.Added = append(result.Added, accountID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("find imported account: %w", err)
	}
	changed, err := assignTags(ctx, tx, accountID, account, tagIDs, now)
	if err != nil {
		return err
	}
	passwordChanged, err := importPassword(ctx, tx, accountID, account.Password, now)
	if err != nil {
		return err
	}
	if replacement, ok := replacements[account.RobloxUserID]; ok {
		record, err := replaceSession(ctx, tx, accountID, replacement, now)
		if err != nil {
			return err
		}
		result.Replaced = append(result.Replaced, record)
		return nil
	}
	if changed || passwordChanged {
		result.Updated = append(result.Updated, accountID)
	}
	return nil
}

func importPassword(ctx context.Context, tx *sql.Tx, accountID int64, password string, now int64) (bool, error) {
	if password == "" {
		return false, nil
	}
	updated, err := tx.ExecContext(ctx, `UPDATE account_sessions SET password = ?, updated_at_ms = ? WHERE account_id = ? AND password <> ?`, password, now, accountID, password)
	if err != nil {
		return false, fmt.Errorf("import saved account password: %w", err)
	}
	count, err := updated.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("count imported saved passwords: %w", err)
	}
	return count > 0, nil
}

func replaceSession(ctx context.Context, tx *sql.Tx, accountID int64, replacement Replacement, now int64) (accountdomain.SessionRecord, error) {
	identity := replacement.Identity
	if _, err := tx.ExecContext(ctx, `UPDATE accounts SET username = ?, display_name = ?, created_at_ms = ?, updated_at_ms = ? WHERE id = ?`, identity.Username, identity.DisplayName, identity.CreatedAt.UTC().UnixMilli(), now, accountID); err != nil {
		return accountdomain.SessionRecord{}, fmt.Errorf("update imported account identity: %w", err)
	}
	var expires any
	if replacement.ExpiresAt != nil {
		expires = replacement.ExpiresAt.UTC().UnixMilli()
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE account_sessions SET roblosecurity = ?, cookie_expires_at_ms = ?, state = ?, state_reason = '', secret_version = secret_version + 1,
replaced_at_ms = ?, last_validated_at_ms = ?, last_success_at_ms = ?, last_auth_failure_at_ms = NULL, updated_at_ms = ? WHERE account_id = ?`,
		replacement.Cookie, expires, accountdomain.StateActive, now, now, now, now, accountID); err != nil {
		return accountdomain.SessionRecord{}, fmt.Errorf("replace imported account cookie: %w", err)
	}
	record := accountdomain.SessionRecord{AccountID: accountID, State: accountdomain.StateActive}
	if err := tx.QueryRowContext(ctx, `SELECT secret_version FROM account_sessions WHERE account_id = ?`, accountID).Scan(&record.SecretVersion); err != nil {
		return accountdomain.SessionRecord{}, fmt.Errorf("read replaced cookie version: %w", err)
	}
	return record, nil
}

func insertAccount(ctx context.Context, tx *sql.Tx, accountID, order int64, account Account, now int64) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO accounts(id, roblox_user_id, username, display_name, display_order, created_at_ms, updated_at_ms) VALUES(?, ?, ?, ?, ?, ?, ?)`,
		accountID, account.RobloxUserID, account.Username, account.DisplayName, order, account.CreatedAtMS, now); err != nil {
		return fmt.Errorf("insert imported account: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO account_sessions(account_id, roblosecurity, cookie_expires_at_ms, state, secret_version, imported_at_ms, updated_at_ms, password)
VALUES(?, ?, ?, ?, 1, ?, ?, ?)`, accountID, account.Cookie, account.CookieExpiresAtMS, accountdomain.StateActive, now, now, account.Password); err != nil {
		return fmt.Errorf("insert imported account session: %w", err)
	}
	return nil
}

func assignTags(ctx context.Context, tx *sql.Tx, accountID int64, account Account, tagIDs map[string]int64, now int64) (bool, error) {
	names := account.Tags
	if account.Favorite {
		names = append([]string{favoriteTagKey}, names...)
	}
	changed := false
	for _, name := range names {
		result, err := tx.ExecContext(ctx, `INSERT INTO account_tags(account_id, tag_id, created_at_ms) VALUES(?, ?, ?) ON CONFLICT(account_id, tag_id) DO NOTHING`, accountID, tagIDs[name], now)
		if err != nil {
			return false, fmt.Errorf("assign imported tag: %w", err)
		}
		if count, err := result.RowsAffected(); err != nil {
			return false, fmt.Errorf("count imported tags: %w", err)
		} else if count > 0 {
			changed = true
		}
	}
	if changed {
		if _, err := tx.ExecContext(ctx, `UPDATE accounts SET updated_at_ms = ? WHERE id = ?`, now, accountID); err != nil {
			return false, fmt.Errorf("update tagged imported account: %w", err)
		}
	}
	return changed, nil
}

// favoriteTagKey cannot collide with a custom tag, because custom tag names never contain spaces.
const favoriteTagKey = " favorite"

func favoriteTagIDs(ctx context.Context, tx *sql.Tx) (map[string]int64, error) {
	var id int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM tags WHERE kind = 'favorite'`).Scan(&id); err != nil {
		return nil, fmt.Errorf("find favorite tag: %w", err)
	}
	return map[string]int64{favoriteTagKey: id}, nil
}

func ensureTag(ctx context.Context, tx *sql.Tx, name string, now int64) (int64, error) {
	if _, err := tx.ExecContext(ctx, `INSERT INTO tags(name, kind, created_at_ms, updated_at_ms) VALUES(?, 'custom', ?, ?) ON CONFLICT(name) DO NOTHING`, name, now, now); err != nil {
		return 0, fmt.Errorf("create imported tag: %w", err)
	}
	var id int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM tags WHERE name = ?`, name).Scan(&id); err != nil {
		return 0, fmt.Errorf("find imported tag: %w", err)
	}
	return id, nil
}

func accountIDs(ctx context.Context, tx *sql.Tx) ([]int64, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM accounts`)
	if err != nil {
		return nil, fmt.Errorf("list replaced accounts: %w", err)
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("read replaced account: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read replaced accounts: %w", err)
	}
	return ids, nil
}

func storePendingSettings(ctx context.Context, tx *sql.Tx, settings *PendingSettings) error {
	if settings == nil {
		return nil
	}
	sections, err := json.Marshal(settings.Sections)
	if err != nil {
		return fmt.Errorf("encode pending settings sections: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO pending_settings_import(singleton, settings, sections) VALUES(1, ?, ?)`, string(settings.Data), string(sections)); err != nil {
		return fmt.Errorf("store pending settings import: %w", err)
	}
	return nil
}

func (repository *Repository) PendingSettings(ctx context.Context) (*PendingSettings, error) {
	db, release, err := repository.provider.Lease(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	var data, sections string
	err = db.QueryRowContext(ctx, `SELECT settings, sections FROM pending_settings_import WHERE singleton = 1`).Scan(&data, &sections)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read pending settings import: %w", err)
	}
	pending := &PendingSettings{Data: json.RawMessage(data)}
	if err := json.Unmarshal([]byte(sections), &pending.Sections); err != nil {
		return nil, fmt.Errorf("decode pending settings sections: %w", err)
	}
	return pending, nil
}

func (repository *Repository) ClearPendingSettings(ctx context.Context) error {
	db, release, err := repository.provider.Lease(ctx)
	if err != nil {
		return err
	}
	defer release()
	if _, err := db.ExecContext(ctx, `DELETE FROM pending_settings_import`); err != nil {
		return fmt.Errorf("clear pending settings import: %w", err)
	}
	return nil
}

func bumpRevision(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `UPDATE vault_state SET storage_revision = storage_revision + 1 WHERE singleton = 1`); err != nil {
		return fmt.Errorf("update storage revision: %w", err)
	}
	return nil
}
