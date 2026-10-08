package accountstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	sqlite3 "github.com/mutecomm/go-sqlcipher/v4"
	accountdomain "github.com/sleaze5/RobloxAccountManager/internal/accounts"
)

type (
	Identity      = accountdomain.Identity
	SessionState  = accountdomain.SessionState
	SessionRecord = accountdomain.SessionRecord
	AccountView   = accountdomain.AccountView
	AccountPage   = accountdomain.AccountPage
	AccountQuery  = accountdomain.AccountQuery
	TagKind       = accountdomain.TagKind
	TagView       = accountdomain.TagView
)

const (
	StateActive         = accountdomain.StateActive
	StateUnknown        = accountdomain.StateUnknown
	StateReauthRequired = accountdomain.StateReauthRequired
	StateChallenged     = accountdomain.StateChallenged
	StateDisabled       = accountdomain.StateDisabled
	TagKindCustom       = accountdomain.TagKindCustom
	TagKindFavorite     = accountdomain.TagKindFavorite
	accountPageSize     = 10
)

var (
	ErrNotFound         = accountdomain.ErrNotFound
	ErrDuplicateAccount = accountdomain.ErrDuplicateAccount
	ErrTagNotFound      = accountdomain.ErrTagNotFound
	ErrDuplicateTag     = accountdomain.ErrDuplicateTag
	ErrInvalidTagName   = accountdomain.ErrInvalidTagName
	ErrInvalidOrder     = accountdomain.ErrInvalidOrder
	ErrStaleSecret      = accountdomain.ErrStaleSecret
)

type DatabaseProvider interface {
	Lease(context.Context) (*sql.DB, func(), error)
}

type Repository struct {
	provider DatabaseProvider
	now      func() time.Time
	logger   *slog.Logger
}

func NewRepository(provider DatabaseProvider, loggers ...*slog.Logger) *Repository {
	logger := slog.New(slog.DiscardHandler)
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	}
	return &Repository{provider: provider, now: time.Now, logger: logger}
}

func (repository *Repository) database(ctx context.Context) (*sql.DB, func(), error) {
	return repository.provider.Lease(ctx)
}

func (repository *Repository) Create(ctx context.Context, identity Identity, cookie string, expiresAt *time.Time) (AccountView, SessionRecord, error) {
	if identity.RobloxUserID <= 0 || identity.Username == "" || cookie == "" {
		return AccountView{}, SessionRecord{}, fmt.Errorf("invalid account record")
	}
	db, release, err := repository.database(ctx)
	if err != nil {
		return AccountView{}, SessionRecord{}, err
	}
	defer release()
	now := repository.now().UTC().Truncate(time.Millisecond)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return AccountView{}, SessionRecord{}, fmt.Errorf("begin account insert: %w", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `
INSERT INTO accounts(roblox_user_id, username, display_name, display_order, created_at_ms, updated_at_ms)
SELECT ?, ?, ?, COALESCE(MAX(display_order) + 1, 0), ?, ? FROM accounts`, identity.RobloxUserID, identity.Username, identity.DisplayName, identity.CreatedAt.UTC().UnixMilli(), now.UnixMilli())
	if err != nil {
		if isConstraint(err) {
			return AccountView{}, SessionRecord{}, ErrDuplicateAccount
		}
		return AccountView{}, SessionRecord{}, fmt.Errorf("insert account: %w", err)
	}
	accountID, err := result.LastInsertId()
	if err != nil {
		return AccountView{}, SessionRecord{}, fmt.Errorf("read inserted account ID: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO account_sessions(account_id, roblosecurity, cookie_expires_at_ms, state, secret_version, imported_at_ms, last_validated_at_ms, last_success_at_ms, updated_at_ms)
VALUES(?, ?, ?, ?, 1, ?, ?, ?, ?)`, accountID, cookie, nullableMillis(expiresAt), StateActive, now.UnixMilli(), now.UnixMilli(), now.UnixMilli(), now.UnixMilli()); err != nil {
		return AccountView{}, SessionRecord{}, fmt.Errorf("insert account session: %w", err)
	}
	if err := bumpRevision(ctx, tx); err != nil {
		return AccountView{}, SessionRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return AccountView{}, SessionRecord{}, fmt.Errorf("commit account insert: %w", err)
	}
	session := SessionRecord{AccountID: accountID, Roblosecurity: cookie, CookieExpiresAt: copyTime(expiresAt), State: StateActive, SecretVersion: 1, ImportedAt: now, LastValidatedAt: &now, LastSuccessAt: &now, UpdatedAt: now}
	repository.logger.Info("account stored", "account_id", accountID, "roblox_user_id", identity.RobloxUserID)
	return viewFrom(identity, accountID, session, now), session, nil
}

func (repository *Repository) FindByRobloxUserID(ctx context.Context, userID int64) (AccountView, error) {
	db, release, err := repository.database(ctx)
	if err != nil {
		return AccountView{}, err
	}
	defer release()
	view, err := scanView(db.QueryRowContext(ctx, viewQuery+` WHERE a.roblox_user_id = ?`, userID))
	if err != nil {
		return AccountView{}, err
	}
	view.Tags, err = listAccountTags(ctx, db, view.ID)
	return view, err
}

func (repository *Repository) ReplaceCookie(ctx context.Context, accountID int64, identity Identity, cookie string, expiresAt *time.Time) (AccountView, SessionRecord, error) {
	if accountID <= 0 || identity.RobloxUserID <= 0 || identity.Username == "" || cookie == "" {
		return AccountView{}, SessionRecord{}, fmt.Errorf("invalid replacement account record")
	}
	db, release, err := repository.database(ctx)
	if err != nil {
		return AccountView{}, SessionRecord{}, err
	}
	defer release()
	now := repository.now().UTC().Truncate(time.Millisecond)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return AccountView{}, SessionRecord{}, err
	}
	defer tx.Rollback()
	var storedUserID int64
	if err := tx.QueryRowContext(ctx, `SELECT roblox_user_id FROM accounts WHERE id = ?`, accountID).Scan(&storedUserID); errors.Is(err, sql.ErrNoRows) {
		return AccountView{}, SessionRecord{}, ErrNotFound
	} else if err != nil {
		return AccountView{}, SessionRecord{}, err
	}
	if storedUserID != identity.RobloxUserID {
		return AccountView{}, SessionRecord{}, fmt.Errorf("replacement cookie belongs to a different Roblox account")
	}
	if _, err := tx.ExecContext(ctx, `UPDATE accounts SET username = ?, display_name = ?, created_at_ms = ?, updated_at_ms = ? WHERE id = ?`, identity.Username, identity.DisplayName, identity.CreatedAt.UTC().UnixMilli(), now.UnixMilli(), accountID); err != nil {
		return AccountView{}, SessionRecord{}, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE account_sessions SET roblosecurity = ?, cookie_expires_at_ms = ?, state = ?, state_reason = '', secret_version = secret_version + 1,
replaced_at_ms = ?, last_validated_at_ms = ?, last_success_at_ms = ?, last_auth_failure_at_ms = NULL, updated_at_ms = ? WHERE account_id = ?`, cookie, nullableMillis(expiresAt), StateActive, now.UnixMilli(), now.UnixMilli(), now.UnixMilli(), now.UnixMilli(), accountID); err != nil {
		return AccountView{}, SessionRecord{}, err
	}
	if err := bumpRevision(ctx, tx); err != nil {
		return AccountView{}, SessionRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return AccountView{}, SessionRecord{}, err
	}
	view, err := repository.GetView(ctx, accountID)
	if err != nil {
		return AccountView{}, SessionRecord{}, err
	}
	session, err := repository.GetSession(ctx, accountID)
	return view, session, err
}

func (repository *Repository) RotateCookie(ctx context.Context, accountID, expectedVersion int64, cookie string, expiresAt *time.Time) (SessionRecord, bool, error) {
	if cookie == "" {
		return SessionRecord{}, false, fmt.Errorf("rotated cookie is empty")
	}
	db, release, err := repository.database(ctx)
	if err != nil {
		return SessionRecord{}, false, err
	}
	defer release()
	now := repository.now().UTC().Truncate(time.Millisecond)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return SessionRecord{}, false, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE account_sessions SET roblosecurity = ?, cookie_expires_at_ms = ?, secret_version = secret_version + 1, rotated_at_ms = ?, updated_at_ms = ? WHERE account_id = ? AND secret_version = ?`, cookie, nullableMillis(expiresAt), now.UnixMilli(), now.UnixMilli(), accountID, expectedVersion)
	if err != nil {
		return SessionRecord{}, false, fmt.Errorf("rotate account cookie: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return SessionRecord{}, false, err
	}
	if changed == 0 {
		return SessionRecord{}, false, ErrStaleSecret
	}
	current, err := scanSession(tx.QueryRowContext(ctx, sessionQuery+` WHERE account_id = ?`, accountID))
	if err != nil {
		return SessionRecord{}, false, err
	}
	if err := bumpRevision(ctx, tx); err != nil {
		return SessionRecord{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return SessionRecord{}, false, err
	}
	return current, true, nil
}

func (repository *Repository) Remove(ctx context.Context, accountID int64) error {
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
	var order int64
	if err := tx.QueryRowContext(ctx, `SELECT display_order FROM accounts WHERE id = ?`, accountID).Scan(&order); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM accounts WHERE id = ?`, accountID); err != nil {
		return err
	}
	var remaining int64
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM accounts`).Scan(&remaining); err != nil {
		return err
	}
	// Two shifts keep display_order unique: move later rows above every existing value, then back down to one below their old value.
	if _, err := tx.ExecContext(ctx, `UPDATE accounts SET display_order = display_order + ? WHERE display_order > ?`, remaining+2, order); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE accounts SET display_order = display_order - ? WHERE display_order > ?`, remaining+3, remaining); err != nil {
		return err
	}
	if err := bumpRevision(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	repository.logger.Info("account removed", "account_id", accountID)
	return nil
}

func (repository *Repository) Page(ctx context.Context, query AccountQuery) (AccountPage, error) {
	if query.Cursor != nil && (query.Cursor.DisplayOrder < 0 || query.Cursor.AccountID <= 0) {
		return AccountPage{}, ErrInvalidOrder
	}
	for _, tagID := range query.TagIDs {
		if tagID <= 0 {
			return AccountPage{}, ErrTagNotFound
		}
	}
	db, release, err := repository.database(ctx)
	if err != nil {
		return AccountPage{}, err
	}
	defer release()
	where := []string{"1 = 1"}
	arguments := make([]any, 0, 6)
	if search := strings.TrimSpace(query.Search); search != "" {
		where = append(where, `(instr(lower(a.username), lower(?)) > 0 OR instr(lower(a.display_name), lower(?)) > 0)`)
		arguments = append(arguments, search, search)
	}
	for _, tagID := range query.TagIDs {
		where = append(where, `EXISTS (SELECT 1 FROM account_tags filter_tags WHERE filter_tags.account_id = a.id AND filter_tags.tag_id = ?)`)
		arguments = append(arguments, tagID)
	}
	if query.Cursor != nil {
		where = append(where, `(a.display_order > ? OR (a.display_order = ? AND a.id > ?))`)
		arguments = append(arguments, query.Cursor.DisplayOrder, query.Cursor.DisplayOrder, query.Cursor.AccountID)
	}
	arguments = append(arguments, accountPageSize+1)
	rows, err := db.QueryContext(ctx, pageViewQuery+` WHERE `+strings.Join(where, " AND ")+` ORDER BY a.display_order, a.id LIMIT ?`, arguments...)
	if err != nil {
		return AccountPage{}, fmt.Errorf("list account page: %w", err)
	}
	views := make([]AccountView, 0, accountPageSize+1)
	orders := make([]int64, 0, accountPageSize+1)
	for rows.Next() {
		view, order, err := scanPageView(rows)
		if err != nil {
			rows.Close()
			return AccountPage{}, err
		}
		views = append(views, view)
		orders = append(orders, order)
	}
	if err := rows.Close(); err != nil {
		return AccountPage{}, err
	}
	page := AccountPage{Accounts: views}
	if len(views) > accountPageSize {
		page.Accounts = views[:accountPageSize]
		page.NextCursor = &accountdomain.AccountCursor{DisplayOrder: orders[accountPageSize-1], AccountID: views[accountPageSize-1].ID}
	}
	if err := loadAccountTags(ctx, db, page.Accounts); err != nil {
		return AccountPage{}, err
	}
	return page, nil
}

func (repository *Repository) Move(ctx context.Context, accountID int64, beforeID, afterID *int64) error {
	if accountID <= 0 || beforeID != nil && *beforeID <= 0 || afterID != nil && *afterID <= 0 || beforeID != nil && *beforeID == accountID || afterID != nil && *afterID == accountID {
		return ErrInvalidOrder
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
	var oldOrder, count int64
	if err := tx.QueryRowContext(ctx, `SELECT display_order FROM accounts WHERE id = ?`, accountID).Scan(&oldOrder); err != nil {
		return ErrInvalidOrder
	}
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM accounts`).Scan(&count); err != nil {
		return err
	}
	newOrder := count - 1
	if afterID != nil {
		if err := tx.QueryRowContext(ctx, `SELECT display_order FROM accounts WHERE id = ?`, *afterID).Scan(&newOrder); err != nil {
			return ErrInvalidOrder
		}
		if oldOrder < newOrder {
			newOrder--
		}
	}
	if beforeID != nil {
		var beforeOrder int64
		if err := tx.QueryRowContext(ctx, `SELECT display_order FROM accounts WHERE id = ?`, *beforeID).Scan(&beforeOrder); err != nil {
			return ErrInvalidOrder
		}
		expected := newOrder - 1
		if oldOrder < beforeOrder {
			beforeOrder--
		}
		if beforeOrder != expected {
			return ErrInvalidOrder
		}
	} else if newOrder != 0 && count > 1 {
		return ErrInvalidOrder
	}
	if newOrder == oldOrder {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `UPDATE accounts SET display_order = ? WHERE id = ?`, count, accountID); err != nil {
		return err
	}
	if newOrder < oldOrder {
		if _, err := tx.ExecContext(ctx, `UPDATE accounts SET display_order = display_order + ? WHERE display_order >= ? AND display_order < ?`, count+1, newOrder, oldOrder); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE accounts SET display_order = display_order - ? WHERE display_order > ?`, count, count); err != nil {
			return err
		}
	} else {
		if _, err := tx.ExecContext(ctx, `UPDATE accounts SET display_order = display_order + ? WHERE display_order > ? AND display_order <= ?`, count+1, oldOrder, newOrder); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE accounts SET display_order = display_order - ? WHERE display_order > ?`, count+2, count); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE accounts SET display_order = ?, updated_at_ms = ? WHERE id = ?`, newOrder, repository.now().UTC().UnixMilli(), accountID); err != nil {
		return err
	}
	if err := bumpRevision(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (repository *Repository) GetView(ctx context.Context, accountID int64) (AccountView, error) {
	db, release, err := repository.database(ctx)
	if err != nil {
		return AccountView{}, err
	}
	defer release()
	view, err := scanView(db.QueryRowContext(ctx, viewQuery+` WHERE a.id = ?`, accountID))
	if err != nil {
		return AccountView{}, err
	}
	view.Tags, err = listAccountTags(ctx, db, accountID)
	return view, err
}

func (repository *Repository) GetSession(ctx context.Context, accountID int64) (SessionRecord, error) {
	db, release, err := repository.database(ctx)
	if err != nil {
		return SessionRecord{}, err
	}
	defer release()
	return scanSession(db.QueryRowContext(ctx, sessionQuery+` WHERE account_id = ?`, accountID))
}

func (repository *Repository) MarkState(ctx context.Context, accountID, expectedVersion int64, state SessionState, reason string) error {
	if !state.Valid() {
		return fmt.Errorf("invalid session state")
	}
	db, release, err := repository.database(ctx)
	if err != nil {
		return err
	}
	defer release()
	now := repository.now().UTC().UnixMilli()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var failure any
	if state == StateReauthRequired {
		failure = now
	}
	result, err := tx.ExecContext(ctx, `UPDATE account_sessions SET state = ?, state_reason = ?, last_auth_failure_at_ms = COALESCE(?, last_auth_failure_at_ms), updated_at_ms = ? WHERE account_id = ? AND secret_version = ?`, state, safeReason(reason), failure, now, accountID, expectedVersion)
	if err != nil {
		return err
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		return ErrStaleSecret
	}
	if err := bumpRevision(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (repository *Repository) RecordValidation(ctx context.Context, accountID, expectedVersion int64, identity Identity) error {
	db, release, err := repository.database(ctx)
	if err != nil {
		return err
	}
	defer release()
	now := repository.now().UTC().UnixMilli()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var userID int64
	if err := tx.QueryRowContext(ctx, `SELECT roblox_user_id FROM accounts WHERE id = ?`, accountID).Scan(&userID); err != nil {
		return err
	}
	if userID != identity.RobloxUserID {
		return fmt.Errorf("validated identity does not match the stored account")
	}
	result, err := tx.ExecContext(ctx, `UPDATE account_sessions SET state = CASE WHEN state = ? THEN state ELSE ? END, state_reason = CASE WHEN state = ? THEN state_reason ELSE '' END, last_validated_at_ms = ?, last_success_at_ms = ?, updated_at_ms = ? WHERE account_id = ? AND secret_version = ?`, StateDisabled, StateActive, StateDisabled, now, now, now, accountID, expectedVersion)
	if err != nil {
		return err
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		return ErrStaleSecret
	}
	if _, err := tx.ExecContext(ctx, `UPDATE accounts SET username = ?, display_name = ?, created_at_ms = ?, updated_at_ms = ? WHERE id = ?`, identity.Username, identity.DisplayName, identity.CreatedAt.UTC().UnixMilli(), now, accountID); err != nil {
		return err
	}
	if err := bumpRevision(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (repository *Repository) RecordSuccess(ctx context.Context, accountID, expectedVersion int64) error {
	db, release, err := repository.database(ctx)
	if err != nil {
		return err
	}
	defer release()
	now := repository.now().UTC().UnixMilli()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE account_sessions SET state = CASE WHEN state = ? THEN state ELSE ? END, state_reason = CASE WHEN state = ? THEN state_reason ELSE '' END, last_success_at_ms = ?, updated_at_ms = ? WHERE account_id = ? AND secret_version = ? AND (state != ? OR last_success_at_ms IS NULL OR last_success_at_ms < ?)`, StateDisabled, StateActive, StateDisabled, now, now, accountID, expectedVersion, StateActive, now-int64(time.Minute/time.Millisecond))
	if err != nil {
		return err
	}
	changed, _ := result.RowsAffected()
	if changed > 0 {
		if err := bumpRevision(ctx, tx); err != nil {
			return err
		}
	}
	return tx.Commit()
}

const viewQuery = `
SELECT a.id, a.roblox_user_id, a.username, a.display_name,
       s.state, s.state_reason, s.secret_version, s.cookie_expires_at_ms,
	   a.created_at_ms, s.imported_at_ms, s.rotated_at_ms, s.last_validated_at_ms,
	   s.updated_at_ms
FROM accounts a JOIN account_sessions s ON s.account_id = a.id`

const pageViewQuery = `
SELECT a.id, a.roblox_user_id, a.username, a.display_name,
       s.state, s.state_reason, s.secret_version, s.cookie_expires_at_ms,
	   a.created_at_ms, s.imported_at_ms, s.rotated_at_ms, s.last_validated_at_ms,
	   s.updated_at_ms, a.display_order
FROM accounts a JOIN account_sessions s ON s.account_id = a.id`

const sessionQuery = `
SELECT account_id, roblosecurity, cookie_expires_at_ms, state, state_reason, secret_version,
       imported_at_ms, replaced_at_ms, rotated_at_ms, last_validated_at_ms,
       last_success_at_ms, last_auth_failure_at_ms, updated_at_ms
FROM account_sessions`

type scanner interface{ Scan(...any) error }

func scanView(row scanner) (AccountView, error) {
	var view AccountView
	var expires, rotated, validated sql.NullInt64
	err := row.Scan(&view.ID, &view.RobloxUserID, &view.Username, &view.DisplayName, &view.State, &view.StateReason, &view.SecretVersion, &expires, &view.CreatedAtMS, &view.ImportedAtMS, &rotated, &validated, &view.UpdatedAtMS)
	if errors.Is(err, sql.ErrNoRows) {
		return AccountView{}, ErrNotFound
	}
	if err != nil {
		return AccountView{}, fmt.Errorf("scan account: %w", err)
	}
	view.Tags = []TagView{}
	view.CookieExpiresAtMS = nullMillisPtr(expires)
	view.RotatedAtMS = nullMillisPtr(rotated)
	view.LastValidatedAtMS = nullMillisPtr(validated)
	return view, nil
}

func scanPageView(row scanner) (AccountView, int64, error) {
	var view AccountView
	var order int64
	var expires, rotated, validated sql.NullInt64
	err := row.Scan(&view.ID, &view.RobloxUserID, &view.Username, &view.DisplayName, &view.State, &view.StateReason, &view.SecretVersion, &expires, &view.CreatedAtMS, &view.ImportedAtMS, &rotated, &validated, &view.UpdatedAtMS, &order)
	if err != nil {
		return AccountView{}, 0, fmt.Errorf("scan account page: %w", err)
	}
	view.Tags = []TagView{}
	view.CookieExpiresAtMS = nullMillisPtr(expires)
	view.RotatedAtMS = nullMillisPtr(rotated)
	view.LastValidatedAtMS = nullMillisPtr(validated)
	return view, order, nil
}

func scanSession(row scanner) (SessionRecord, error) {
	var result SessionRecord
	var expires, replaced, rotated, validated, success, failure sql.NullInt64
	var imported, updated int64
	err := row.Scan(&result.AccountID, &result.Roblosecurity, &expires, &result.State, &result.StateReason, &result.SecretVersion, &imported, &replaced, &rotated, &validated, &success, &failure, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return SessionRecord{}, ErrNotFound
	}
	if err != nil {
		return SessionRecord{}, fmt.Errorf("scan account session: %w", err)
	}
	result.ImportedAt = timeFromMillis(imported)
	result.UpdatedAt = timeFromMillis(updated)
	result.CookieExpiresAt = timePtrFromNull(expires)
	result.ReplacedAt = timePtrFromNull(replaced)
	result.RotatedAt = timePtrFromNull(rotated)
	result.LastValidatedAt = timePtrFromNull(validated)
	result.LastSuccessAt = timePtrFromNull(success)
	result.LastAuthFailureAt = timePtrFromNull(failure)
	return result, nil
}

func viewFrom(identity Identity, accountID int64, session SessionRecord, updated time.Time) AccountView {
	return AccountView{ID: accountID, RobloxUserID: identity.RobloxUserID, Username: identity.Username, DisplayName: identity.DisplayName, Tags: []TagView{}, State: session.State, SecretVersion: session.SecretVersion, CookieExpiresAtMS: millisPtr(session.CookieExpiresAt), CreatedAtMS: identity.CreatedAt.UTC().UnixMilli(), ImportedAtMS: session.ImportedAt.UnixMilli(), LastValidatedAtMS: millisPtr(session.LastValidatedAt), UpdatedAtMS: updated.UnixMilli()}
}

func bumpRevision(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `UPDATE vault_state SET storage_revision = storage_revision + 1 WHERE singleton = 1`)
	return err
}

func nullableMillis(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().UnixMilli()
}

func nullMillisPtr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	result := value.Int64
	return &result
}

func copyTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	result := value.UTC().Truncate(time.Millisecond)
	return &result
}

func timeFromMillis(value int64) time.Time { return time.UnixMilli(value).UTC() }

func timePtrFromNull(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}
	result := timeFromMillis(value.Int64)
	return &result
}

func millisPtr(value *time.Time) *int64 {
	if value == nil {
		return nil
	}
	result := value.UTC().UnixMilli()
	return &result
}

func safeReason(reason string) string {
	if len(reason) > 256 {
		return reason[:256]
	}
	return reason
}

func isConstraint(err error) bool {
	var sqliteErr sqlite3.Error
	return errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint
}
