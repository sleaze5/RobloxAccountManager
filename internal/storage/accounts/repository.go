package accountstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ncruces/go-sqlite3"
	accountdomain "github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/vault"
)

const accountPageSize = 10

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

func (repository *Repository) leaseDatabase(ctx context.Context) (*sql.DB, func(), error) {
	database, release, err := repository.provider.Lease(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", accountdomain.ErrUnavailable, err)
	}
	return database, release, nil
}

func (repository *Repository) Create(ctx context.Context, identity accountdomain.Identity, cookie string, expiresAt *time.Time) (accountdomain.AccountView, accountdomain.SessionRecord, error) {
	if identity.RobloxUserID <= 0 || identity.Username == "" || cookie == "" {
		return accountdomain.AccountView{}, accountdomain.SessionRecord{}, errors.New("invalid account record")
	}
	database, release, err := repository.leaseDatabase(ctx)
	if err != nil {
		return accountdomain.AccountView{}, accountdomain.SessionRecord{}, err
	}
	defer release()
	now := time.Now().UTC().Truncate(time.Millisecond)
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return accountdomain.AccountView{}, accountdomain.SessionRecord{}, fmt.Errorf("begin account insert: %w", err)
	}
	defer transaction.Rollback()
	insertion, err := transaction.ExecContext(ctx, `
INSERT INTO accounts(roblox_user_id, username, display_name, display_order, created_at_ms, updated_at_ms)
SELECT ?, ?, ?, COALESCE(MAX(display_order) + 1, 0), ?, ? FROM accounts`, identity.RobloxUserID, identity.Username, identity.DisplayName, identity.CreatedAt.UTC().UnixMilli(), now.UnixMilli())
	if err != nil {
		if isConstraint(err) {
			return accountdomain.AccountView{}, accountdomain.SessionRecord{}, accountdomain.ErrDuplicateAccount
		}
		return accountdomain.AccountView{}, accountdomain.SessionRecord{}, fmt.Errorf("insert account: %w", err)
	}
	accountID, err := insertion.LastInsertId()
	if err != nil {
		return accountdomain.AccountView{}, accountdomain.SessionRecord{}, fmt.Errorf("read inserted account ID: %w", err)
	}
	if _, err := transaction.ExecContext(ctx, `
INSERT INTO account_sessions(account_id, roblosecurity, cookie_expires_at_ms, state, secret_version, imported_at_ms, last_validated_at_ms, last_success_at_ms, updated_at_ms)
VALUES(?, ?, ?, ?, 1, ?, ?, ?, ?)`, accountID, cookie, nullableUnixMilliseconds(expiresAt), accountdomain.StateActive, now.UnixMilli(), now.UnixMilli(), now.UnixMilli(), now.UnixMilli()); err != nil {
		return accountdomain.AccountView{}, accountdomain.SessionRecord{}, fmt.Errorf("insert account session: %w", err)
	}
	if err := vault.BumpRevision(ctx, transaction); err != nil {
		return accountdomain.AccountView{}, accountdomain.SessionRecord{}, err
	}
	if err := transaction.Commit(); err != nil {
		return accountdomain.AccountView{}, accountdomain.SessionRecord{}, fmt.Errorf("commit account insert: %w", err)
	}
	session := accountdomain.SessionRecord{AccountID: accountID, Roblosecurity: cookie, CookieExpiresAt: copyTime(expiresAt), State: accountdomain.StateActive, SecretVersion: 1, ImportedAt: now, LastValidatedAt: &now}
	repository.logger.InfoContext(ctx, "account stored", "operation", "account-store", "account_id", accountID, "roblox_user_id", identity.RobloxUserID)
	return viewFrom(identity, accountID, session, now), session, nil
}

func (repository *Repository) FindByRobloxUserID(ctx context.Context, userID int64) (accountdomain.AccountView, error) {
	database, release, err := repository.leaseDatabase(ctx)
	if err != nil {
		return accountdomain.AccountView{}, err
	}
	defer release()
	view, err := scanView(database.QueryRowContext(ctx, viewQuery+` WHERE a.roblox_user_id = ?`, userID))
	if err != nil {
		return accountdomain.AccountView{}, err
	}
	view.Tags, err = listAccountTags(ctx, database, view.ID)
	return view, err
}

func (repository *Repository) ReplaceCookie(ctx context.Context, accountID int64, identity accountdomain.Identity, cookie string, expiresAt *time.Time) (accountdomain.AccountView, accountdomain.SessionRecord, error) {
	if accountID <= 0 || identity.RobloxUserID <= 0 || identity.Username == "" || cookie == "" {
		return accountdomain.AccountView{}, accountdomain.SessionRecord{}, errors.New("invalid replacement account record")
	}
	database, release, err := repository.leaseDatabase(ctx)
	if err != nil {
		return accountdomain.AccountView{}, accountdomain.SessionRecord{}, err
	}
	defer release()
	now := time.Now().UTC().Truncate(time.Millisecond)
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return accountdomain.AccountView{}, accountdomain.SessionRecord{}, fmt.Errorf("begin cookie replacement: %w", err)
	}
	defer transaction.Rollback()
	if err := validateReplacementIdentity(ctx, transaction, accountID, identity); err != nil {
		return accountdomain.AccountView{}, accountdomain.SessionRecord{}, err
	}
	if _, err := transaction.ExecContext(ctx, `UPDATE accounts SET username = ?, display_name = ?, created_at_ms = ?, updated_at_ms = ? WHERE id = ?`, identity.Username, identity.DisplayName, identity.CreatedAt.UTC().UnixMilli(), now.UnixMilli(), accountID); err != nil {
		return accountdomain.AccountView{}, accountdomain.SessionRecord{}, fmt.Errorf("update replaced account: %w", err)
	}
	if _, err := transaction.ExecContext(ctx, `
UPDATE account_sessions SET roblosecurity = ?, cookie_expires_at_ms = ?, state = ?, state_reason = '', secret_version = secret_version + 1,
replaced_at_ms = ?, last_validated_at_ms = ?, last_success_at_ms = ?, last_auth_failure_at_ms = NULL, updated_at_ms = ? WHERE account_id = ?`, cookie, nullableUnixMilliseconds(expiresAt), accountdomain.StateActive, now.UnixMilli(), now.UnixMilli(), now.UnixMilli(), now.UnixMilli(), accountID); err != nil {
		return accountdomain.AccountView{}, accountdomain.SessionRecord{}, fmt.Errorf("update replaced session: %w", err)
	}
	if err := vault.BumpRevision(ctx, transaction); err != nil {
		return accountdomain.AccountView{}, accountdomain.SessionRecord{}, err
	}
	view, err := scanView(transaction.QueryRowContext(ctx, viewQuery+` WHERE a.id = ?`, accountID))
	if err != nil {
		return accountdomain.AccountView{}, accountdomain.SessionRecord{}, fmt.Errorf("read replacement account: %w", err)
	}
	view.Tags, err = listAccountTags(ctx, transaction, accountID)
	if err != nil {
		return accountdomain.AccountView{}, accountdomain.SessionRecord{}, fmt.Errorf("read replacement account tags: %w", err)
	}
	session, err := scanSession(transaction.QueryRowContext(ctx, sessionQuery+` WHERE account_id = ?`, accountID))
	if err != nil {
		return accountdomain.AccountView{}, accountdomain.SessionRecord{}, fmt.Errorf("read replacement session: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return accountdomain.AccountView{}, accountdomain.SessionRecord{}, fmt.Errorf("commit cookie replacement: %w", err)
	}
	return view, session, nil
}

func (repository *Repository) RotateCookie(ctx context.Context, accountID, expectedVersion int64, cookie string, expiresAt *time.Time) (accountdomain.SessionRecord, error) {
	if cookie == "" {
		return accountdomain.SessionRecord{}, errors.New("rotated cookie is empty")
	}
	database, release, err := repository.leaseDatabase(ctx)
	if err != nil {
		return accountdomain.SessionRecord{}, err
	}
	defer release()
	now := time.Now().UTC().Truncate(time.Millisecond)
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return accountdomain.SessionRecord{}, fmt.Errorf("begin cookie rotation: %w", err)
	}
	defer transaction.Rollback()
	update, err := transaction.ExecContext(ctx, `UPDATE account_sessions SET roblosecurity = ?, cookie_expires_at_ms = ?, secret_version = secret_version + 1, rotated_at_ms = ?, updated_at_ms = ? WHERE account_id = ? AND secret_version = ?`, cookie, nullableUnixMilliseconds(expiresAt), now.UnixMilli(), now.UnixMilli(), accountID, expectedVersion)
	if err != nil {
		return accountdomain.SessionRecord{}, fmt.Errorf("rotate account cookie: %w", err)
	}
	changed, err := update.RowsAffected()
	if err != nil {
		return accountdomain.SessionRecord{}, fmt.Errorf("count rotated sessions: %w", err)
	}
	if changed == 0 {
		return accountdomain.SessionRecord{}, accountdomain.ErrStaleSecret
	}
	current, err := scanSession(transaction.QueryRowContext(ctx, sessionQuery+` WHERE account_id = ?`, accountID))
	if err != nil {
		return accountdomain.SessionRecord{}, err
	}
	if err := vault.BumpRevision(ctx, transaction); err != nil {
		return accountdomain.SessionRecord{}, err
	}
	if err := transaction.Commit(); err != nil {
		return accountdomain.SessionRecord{}, fmt.Errorf("commit cookie rotation: %w", err)
	}
	return current, nil
}

func (repository *Repository) Remove(ctx context.Context, accountID int64) error {
	database, release, err := repository.leaseDatabase(ctx)
	if err != nil {
		return err
	}
	defer release()
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin account removal: %w", err)
	}
	defer transaction.Rollback()
	var order int64
	if err := transaction.QueryRowContext(ctx, `SELECT display_order FROM accounts WHERE id = ?`, accountID).Scan(&order); errors.Is(err, sql.ErrNoRows) {
		return accountdomain.ErrNotFound
	} else if err != nil {
		return fmt.Errorf("read removed account: %w", err)
	}
	if _, err := transaction.ExecContext(ctx, `DELETE FROM accounts WHERE id = ?`, accountID); err != nil {
		return fmt.Errorf("delete account: %w", err)
	}
	var remaining int64
	if err := transaction.QueryRowContext(ctx, `SELECT count(*) FROM accounts`).Scan(&remaining); err != nil {
		return fmt.Errorf("count remaining accounts: %w", err)
	}
	// Two shifts keep display_order unique: move later rows above every existing value, then back down to one below their old value.
	if _, err := transaction.ExecContext(ctx, `UPDATE accounts SET display_order = display_order + ? WHERE display_order > ?`, remaining+2, order); err != nil {
		return fmt.Errorf("shift account order: %w", err)
	}
	if _, err := transaction.ExecContext(ctx, `UPDATE accounts SET display_order = display_order - ? WHERE display_order > ?`, remaining+3, remaining); err != nil {
		return fmt.Errorf("compact account order: %w", err)
	}
	if err := vault.BumpRevision(ctx, transaction); err != nil {
		return err
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit account removal: %w", err)
	}
	repository.logger.InfoContext(ctx, "account removed", "operation", "account-remove", "account_id", accountID)
	return nil
}

func (repository *Repository) Page(ctx context.Context, query accountdomain.AccountQuery) (accountdomain.AccountPage, error) {
	if err := validatePageQuery(query); err != nil {
		return accountdomain.AccountPage{}, err
	}
	database, release, err := repository.leaseDatabase(ctx)
	if err != nil {
		return accountdomain.AccountPage{}, err
	}
	defer release()
	where := []string{"1 = 1"}
	arguments := make([]any, 0, 6)
	if search := strings.TrimSpace(query.Search); search != "" {
		where = append(where, `(instr(lower(a.username), lower(?)) > 0 OR instr(lower(a.display_name), lower(?)) > 0 OR instr(CAST(a.roblox_user_id AS TEXT), ?) > 0)`)
		arguments = append(arguments, search, search, search)
	}
	for _, tagID := range query.TagIDs {
		where = append(where, `EXISTS (SELECT 1 FROM account_tags filter_tags WHERE filter_tags.account_id = a.id AND filter_tags.tag_id = ?)`)
		arguments = append(arguments, tagID)
	}
	var total int64
	if err := database.QueryRowContext(ctx, `SELECT count(*) FROM accounts a JOIN account_sessions s ON s.account_id = a.id WHERE `+strings.Join(where, " AND "), arguments...).Scan(&total); err != nil {
		return accountdomain.AccountPage{}, fmt.Errorf("count account page: %w", err)
	}
	if query.Cursor != nil {
		where = append(where, `(a.display_order > ? OR (a.display_order = ? AND a.id > ?))`)
		arguments = append(arguments, query.Cursor.DisplayOrder, query.Cursor.DisplayOrder, query.Cursor.AccountID)
	}
	arguments = append(arguments, accountPageSize+1)
	rows, err := database.QueryContext(ctx, viewColumns+`, a.display_order`+viewTables+` WHERE `+strings.Join(where, " AND ")+` ORDER BY a.display_order, a.id LIMIT ?`, arguments...)
	if err != nil {
		return accountdomain.AccountPage{}, fmt.Errorf("list account page: %w", err)
	}
	defer rows.Close()
	views := make([]accountdomain.AccountView, 0, accountPageSize+1)
	orders := make([]int64, 0, accountPageSize+1)
	for rows.Next() {
		var order int64
		view, err := scanView(rows, &order)
		if err != nil {
			return accountdomain.AccountPage{}, err
		}
		views = append(views, view)
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return accountdomain.AccountPage{}, fmt.Errorf("read account page: %w", err)
	}
	if err := rows.Close(); err != nil {
		return accountdomain.AccountPage{}, fmt.Errorf("close account page: %w", err)
	}
	page := accountdomain.AccountPage{Accounts: views, Total: total}
	if len(views) > accountPageSize {
		page.Accounts = views[:accountPageSize]
		page.NextCursor = &accountdomain.AccountCursor{DisplayOrder: orders[accountPageSize-1], AccountID: views[accountPageSize-1].ID}
	}
	if err := loadAccountTags(ctx, database, page.Accounts); err != nil {
		return accountdomain.AccountPage{}, err
	}
	return page, nil
}

func (repository *Repository) Move(ctx context.Context, accountID int64, beforeID, afterID *int64) error {
	if err := validateMove(accountID, beforeID, afterID); err != nil {
		return err
	}
	database, release, err := repository.leaseDatabase(ctx)
	if err != nil {
		return err
	}
	defer release()
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin account move: %w", err)
	}
	defer transaction.Rollback()
	var oldOrder, count int64
	if err := transaction.QueryRowContext(ctx, `SELECT display_order FROM accounts WHERE id = ?`, accountID).Scan(&oldOrder); err != nil {
		return accountdomain.ErrInvalidOrder
	}
	if err := transaction.QueryRowContext(ctx, `SELECT count(*) FROM accounts`).Scan(&count); err != nil {
		return fmt.Errorf("count accounts: %w", err)
	}
	newOrder, err := moveDestination(ctx, transaction, oldOrder, count, beforeID, afterID)
	if err != nil {
		return err
	}
	if newOrder == oldOrder {
		return nil
	}
	if _, err := transaction.ExecContext(ctx, `UPDATE accounts SET display_order = ? WHERE id = ?`, count, accountID); err != nil {
		return fmt.Errorf("park moved account: %w", err)
	}
	if err := shiftMovedAccounts(ctx, transaction, oldOrder, newOrder, count); err != nil {
		return err
	}
	if _, err := transaction.ExecContext(ctx, `UPDATE accounts SET display_order = ?, updated_at_ms = ? WHERE id = ?`, newOrder, time.Now().UTC().UnixMilli(), accountID); err != nil {
		return fmt.Errorf("place moved account: %w", err)
	}
	if err := vault.BumpRevision(ctx, transaction); err != nil {
		return err
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit account move: %w", err)
	}
	return nil
}

func (repository *Repository) GetView(ctx context.Context, accountID int64) (accountdomain.AccountView, error) {
	database, release, err := repository.leaseDatabase(ctx)
	if err != nil {
		return accountdomain.AccountView{}, err
	}
	defer release()
	view, err := scanView(database.QueryRowContext(ctx, viewQuery+` WHERE a.id = ?`, accountID))
	if err != nil {
		return accountdomain.AccountView{}, err
	}
	view.Tags, err = listAccountTags(ctx, database, accountID)
	return view, err
}

func (repository *Repository) GetSession(ctx context.Context, accountID int64) (accountdomain.SessionRecord, error) {
	database, release, err := repository.leaseDatabase(ctx)
	if err != nil {
		return accountdomain.SessionRecord{}, err
	}
	defer release()
	return scanSession(database.QueryRowContext(ctx, sessionQuery+` WHERE account_id = ?`, accountID))
}

func (repository *Repository) MarkState(ctx context.Context, accountID, expectedVersion int64, state accountdomain.SessionState, reason string) error {
	if !state.Valid() {
		return errors.New("invalid session state")
	}
	database, release, err := repository.leaseDatabase(ctx)
	if err != nil {
		return err
	}
	defer release()
	now := time.Now().UTC().UnixMilli()
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin session state update: %w", err)
	}
	defer transaction.Rollback()
	var failure any
	if state == accountdomain.StateReauthRequired {
		failure = now
	}
	update, err := transaction.ExecContext(ctx, `UPDATE account_sessions SET state = ?, state_reason = ?, last_auth_failure_at_ms = COALESCE(?, last_auth_failure_at_ms), updated_at_ms = ? WHERE account_id = ? AND secret_version = ?`, state, safeReason(reason), failure, now, accountID, expectedVersion)
	if err != nil {
		return fmt.Errorf("update session state: %w", err)
	}
	changed, err := update.RowsAffected()
	if err != nil {
		return fmt.Errorf("count updated session states: %w", err)
	}
	if changed == 0 {
		return accountdomain.ErrStaleSecret
	}
	if err := vault.BumpRevision(ctx, transaction); err != nil {
		return err
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit session state: %w", err)
	}
	return nil
}

func (repository *Repository) RecordValidation(ctx context.Context, accountID, expectedVersion int64, identity accountdomain.Identity) error {
	database, release, err := repository.leaseDatabase(ctx)
	if err != nil {
		return err
	}
	defer release()
	now := time.Now().UTC().UnixMilli()
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin validation record: %w", err)
	}
	defer transaction.Rollback()
	var userID int64
	if err := transaction.QueryRowContext(ctx, `SELECT roblox_user_id FROM accounts WHERE id = ?`, accountID).Scan(&userID); err != nil {
		return fmt.Errorf("read validated account: %w", err)
	}
	if userID != identity.RobloxUserID {
		return errors.New("validated identity does not match the stored account")
	}
	update, err := transaction.ExecContext(ctx, `UPDATE account_sessions SET state = CASE WHEN state = ? THEN state ELSE ? END, state_reason = CASE WHEN state = ? THEN state_reason ELSE '' END, last_validated_at_ms = ?, last_success_at_ms = ?, updated_at_ms = ? WHERE account_id = ? AND secret_version = ?`, accountdomain.StateDisabled, accountdomain.StateActive, accountdomain.StateDisabled, now, now, now, accountID, expectedVersion)
	if err != nil {
		return fmt.Errorf("record validation: %w", err)
	}
	changed, err := update.RowsAffected()
	if err != nil {
		return fmt.Errorf("count recorded validations: %w", err)
	}
	if changed == 0 {
		return accountdomain.ErrStaleSecret
	}
	if _, err := transaction.ExecContext(ctx, `UPDATE accounts SET username = ?, display_name = ?, created_at_ms = ?, updated_at_ms = ? WHERE id = ?`, identity.Username, identity.DisplayName, identity.CreatedAt.UTC().UnixMilli(), now, accountID); err != nil {
		return fmt.Errorf("update validated account: %w", err)
	}
	if err := vault.BumpRevision(ctx, transaction); err != nil {
		return err
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit validation record: %w", err)
	}
	return nil
}

func (repository *Repository) RecordSuccess(ctx context.Context, accountID, expectedVersion int64) error {
	database, release, err := repository.leaseDatabase(ctx)
	if err != nil {
		return err
	}
	defer release()
	now := time.Now().UTC().UnixMilli()
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin success record: %w", err)
	}
	defer transaction.Rollback()
	update, err := transaction.ExecContext(ctx, `UPDATE account_sessions SET state = CASE WHEN state = ? THEN state ELSE ? END, state_reason = CASE WHEN state = ? THEN state_reason ELSE '' END, last_success_at_ms = ?, updated_at_ms = ? WHERE account_id = ? AND secret_version = ? AND (state != ? OR last_success_at_ms IS NULL OR last_success_at_ms < ?)`, accountdomain.StateDisabled, accountdomain.StateActive, accountdomain.StateDisabled, now, now, accountID, expectedVersion, accountdomain.StateActive, now-int64(time.Minute/time.Millisecond))
	if err != nil {
		return fmt.Errorf("record success: %w", err)
	}
	changed, err := update.RowsAffected()
	if err != nil {
		return fmt.Errorf("count recorded successes: %w", err)
	}
	if changed > 0 {
		if err := vault.BumpRevision(ctx, transaction); err != nil {
			return err
		}
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit success record: %w", err)
	}
	return nil
}

const viewColumns = `
SELECT a.id, a.roblox_user_id, COALESCE(NULLIF(a.username, ''), 'User ' || a.roblox_user_id), a.display_name,
       s.state, s.state_reason, s.secret_version, s.cookie_expires_at_ms,
	   a.created_at_ms, s.imported_at_ms, s.rotated_at_ms, s.last_validated_at_ms,
	   s.updated_at_ms, s.password <> ''`

const viewQuery = viewColumns + viewTables

const viewTables = `
FROM accounts a JOIN account_sessions s ON s.account_id = a.id`

const sessionQuery = `
SELECT account_id, roblosecurity, cookie_expires_at_ms, state, secret_version, imported_at_ms, last_validated_at_ms
FROM account_sessions`

type scanner interface{ Scan(...any) error }

func scanView(row scanner, extra ...any) (accountdomain.AccountView, error) {
	var view accountdomain.AccountView
	var expires, rotated, validated sql.NullInt64
	err := row.Scan(append([]any{&view.ID, &view.RobloxUserID, &view.Username, &view.DisplayName, &view.State, &view.StateReason, &view.SecretVersion, &expires, &view.CreatedAtMS, &view.ImportedAtMS, &rotated, &validated, &view.UpdatedAtMS, &view.HasSavedPassword}, extra...)...)
	if errors.Is(err, sql.ErrNoRows) {
		return accountdomain.AccountView{}, accountdomain.ErrNotFound
	}
	if err != nil {
		return accountdomain.AccountView{}, fmt.Errorf("scan account: %w", err)
	}
	view.Tags = []accountdomain.TagView{}
	view.CookieExpiresAtMS = int64PointerFromNull(expires)
	view.RotatedAtMS = int64PointerFromNull(rotated)
	view.LastValidatedAtMS = int64PointerFromNull(validated)
	return view, nil
}

func scanSession(row scanner) (accountdomain.SessionRecord, error) {
	var session accountdomain.SessionRecord
	var expires, validated sql.NullInt64
	var imported int64
	err := row.Scan(&session.AccountID, &session.Roblosecurity, &expires, &session.State, &session.SecretVersion, &imported, &validated)
	if errors.Is(err, sql.ErrNoRows) {
		return accountdomain.SessionRecord{}, accountdomain.ErrNotFound
	}
	if err != nil {
		return accountdomain.SessionRecord{}, fmt.Errorf("scan account session: %w", err)
	}
	session.ImportedAt = timeFromUnixMilliseconds(imported)
	session.CookieExpiresAt = timePointerFromNull(expires)
	session.LastValidatedAt = timePointerFromNull(validated)
	return session, nil
}

func viewFrom(identity accountdomain.Identity, accountID int64, session accountdomain.SessionRecord, updated time.Time) accountdomain.AccountView {
	return accountdomain.AccountView{ID: accountID, RobloxUserID: identity.RobloxUserID, Username: identity.Username, DisplayName: identity.DisplayName, Tags: []accountdomain.TagView{}, State: session.State, SecretVersion: session.SecretVersion, CookieExpiresAtMS: unixMillisecondsPointer(session.CookieExpiresAt), CreatedAtMS: identity.CreatedAt.UTC().UnixMilli(), ImportedAtMS: session.ImportedAt.UnixMilli(), LastValidatedAtMS: unixMillisecondsPointer(session.LastValidatedAt), UpdatedAtMS: updated.UnixMilli()}
}

func nullableUnixMilliseconds(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().UnixMilli()
}

func int64PointerFromNull(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	milliseconds := value.Int64
	return &milliseconds
}

func copyTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	truncated := value.UTC().Truncate(time.Millisecond)
	return &truncated
}

func timeFromUnixMilliseconds(value int64) time.Time { return time.UnixMilli(value).UTC() }

func timePointerFromNull(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}
	timestamp := timeFromUnixMilliseconds(value.Int64)
	return &timestamp
}

func unixMillisecondsPointer(value *time.Time) *int64 {
	if value == nil {
		return nil
	}
	milliseconds := value.UTC().UnixMilli()
	return &milliseconds
}

func safeReason(reason string) string {
	if len(reason) > 256 {
		return reason[:256]
	}
	return reason
}

func isConstraint(err error) bool {
	return errors.Is(err, sqlite3.CONSTRAINT)
}

func validateReplacementIdentity(ctx context.Context, transaction *sql.Tx, accountID int64, identity accountdomain.Identity) error {
	var storedUserID int64
	if err := transaction.QueryRowContext(ctx, `SELECT roblox_user_id FROM accounts WHERE id = ?`, accountID).Scan(&storedUserID); errors.Is(err, sql.ErrNoRows) {
		return accountdomain.ErrNotFound
	} else if err != nil {
		return fmt.Errorf("read replaced account: %w", err)
	}
	if storedUserID != identity.RobloxUserID {
		return errors.New("replacement cookie belongs to a different Roblox account")
	}
	return nil
}

func validatePageQuery(query accountdomain.AccountQuery) error {
	if query.Cursor != nil && (query.Cursor.DisplayOrder < 0 || query.Cursor.AccountID <= 0) {
		return accountdomain.ErrInvalidOrder
	}
	for _, tagID := range query.TagIDs {
		if tagID <= 0 {
			return accountdomain.ErrTagNotFound
		}
	}
	return nil
}

func validateMove(accountID int64, beforeID, afterID *int64) error {
	if accountID <= 0 || beforeID != nil && *beforeID <= 0 || afterID != nil && *afterID <= 0 || beforeID != nil && *beforeID == accountID || afterID != nil && *afterID == accountID {
		return accountdomain.ErrInvalidOrder
	}
	return nil
}

func moveDestination(ctx context.Context, transaction *sql.Tx, oldOrder, count int64, beforeID, afterID *int64) (int64, error) {
	newOrder := count - 1
	if afterID != nil {
		if err := transaction.QueryRowContext(ctx, `SELECT display_order FROM accounts WHERE id = ?`, *afterID).Scan(&newOrder); err != nil {
			return 0, accountdomain.ErrInvalidOrder
		}
		if oldOrder < newOrder {
			newOrder--
		}
	}
	if beforeID != nil {
		var beforeOrder int64
		if err := transaction.QueryRowContext(ctx, `SELECT display_order FROM accounts WHERE id = ?`, *beforeID).Scan(&beforeOrder); err != nil {
			return 0, accountdomain.ErrInvalidOrder
		}
		expected := newOrder - 1
		if oldOrder < beforeOrder {
			beforeOrder--
		}
		if beforeOrder != expected {
			return 0, accountdomain.ErrInvalidOrder
		}
	} else if newOrder != 0 && count > 1 {
		return 0, accountdomain.ErrInvalidOrder
	}
	return newOrder, nil
}

func shiftMovedAccounts(ctx context.Context, transaction *sql.Tx, oldOrder, newOrder, count int64) error {
	if newOrder < oldOrder {
		if _, err := transaction.ExecContext(ctx, `UPDATE accounts SET display_order = display_order + ? WHERE display_order >= ? AND display_order < ?`, count+1, newOrder, oldOrder); err != nil {
			return fmt.Errorf("shift accounts down: %w", err)
		}
		if _, err := transaction.ExecContext(ctx, `UPDATE accounts SET display_order = display_order - ? WHERE display_order > ?`, count, count); err != nil {
			return fmt.Errorf("compact accounts down: %w", err)
		}
	} else {
		if _, err := transaction.ExecContext(ctx, `UPDATE accounts SET display_order = display_order + ? WHERE display_order > ? AND display_order <= ?`, count+1, oldOrder, newOrder); err != nil {
			return fmt.Errorf("shift accounts up: %w", err)
		}
		if _, err := transaction.ExecContext(ctx, `UPDATE accounts SET display_order = display_order - ? WHERE display_order > ?`, count+2, count); err != nil {
			return fmt.Errorf("compact accounts up: %w", err)
		}
	}
	return nil
}

func (repository *Repository) RestoreNames(ctx context.Context, accountID int64, identity accountdomain.Identity) error {
	database, release, err := repository.leaseDatabase(ctx)
	if err != nil {
		return err
	}
	defer release()
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin account name recovery: %w", err)
	}
	defer transaction.Rollback()
	update, err := transaction.ExecContext(ctx, `UPDATE accounts SET username = ?, display_name = ? WHERE id = ? AND roblox_user_id = ? AND username = ''`, identity.Username, identity.DisplayName, accountID, identity.RobloxUserID)
	if err != nil {
		return fmt.Errorf("restore saved account names: %w", err)
	}
	changed, err := update.RowsAffected()
	if err != nil {
		return fmt.Errorf("count restored account names: %w", err)
	}
	if changed > 0 {
		if err := vault.BumpRevision(ctx, transaction); err != nil {
			return err
		}
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit account name recovery: %w", err)
	}
	return nil
}
