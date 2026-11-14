package appservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/browser"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

type ImportStatus string

const (
	ImportCreated   ImportStatus = "created"
	ImportUpdated   ImportStatus = "updated"
	ImportDuplicate ImportStatus = "duplicate"
	ImportFailed    ImportStatus = "failed"
)

func (service *Service) BrowserAccount(ctx context.Context, accountID int64) (accounts.AccountView, roblox.SessionSnapshot, error) {
	view, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return accounts.AccountView{}, roblox.SessionSnapshot{}, mapRepositoryError(err)
	}
	record, err := service.sessions.Snapshot(accountID)
	return view, record, mapRepositoryError(err)
}

func (service *Service) BrowserCandidateAccount(ctx context.Context, robloxUserID int64) (accounts.AccountView, bool) {
	view, err := service.accountRepository.FindByRobloxUserID(ctx, robloxUserID)
	return view, err == nil
}

func (service *Service) FindAccountByRobloxUserID(ctx context.Context, robloxUserID int64) (*accounts.AccountView, error) {
	view, err := service.accountRepository.FindByRobloxUserID(ctx, robloxUserID)
	if errors.Is(err, accounts.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, mapRepositoryError(err)
	}
	return &view, nil
}

func (service *Service) GetRobloxUser(ctx context.Context, robloxUserID int64) (robloxservices.UserView, error) {
	identity, err := service.users.Profile(ctx, robloxUserID)
	if err != nil {
		service.logger.Warn("roblox user lookup failed", "operation", "user-profile", "roblox_user_id", robloxUserID, "error", err)
		return robloxservices.UserView{}, err
	}
	return robloxservices.UserView{UserID: identity.RobloxUserID, Username: identity.Username, DisplayName: identity.DisplayName}, nil
}

func (service *Service) ValidateBrowserCookie(ctx context.Context, cookie, browserID string) (roblox.CookieValidation, error) {
	validation, err := service.users.ValidateCookie(ctx, cookie, browserID)
	if err != nil {
		return roblox.CookieValidation{}, err
	}
	existing, err := service.accountRepository.FindByRobloxUserID(ctx, validation.Identity.RobloxUserID)
	if err == nil {
		validation.BrowserID = service.sessions.BrowserID(existing.ID)
	} else if !errors.Is(err, accounts.ErrNotFound) {
		return roblox.CookieValidation{}, mapRepositoryError(err)
	}
	return validation, nil
}

func (service *Service) BrowserAvatar(ctx context.Context, robloxUserID int64) string {
	views, err := service.thumbnails.AvatarHeadshots(ctx, []int64{robloxUserID})
	if err != nil || len(views) == 0 {
		return ""
	}
	return views[0].ImageURL
}

func (service *Service) CommitBrowserCandidate(ctx context.Context, validation roblox.CookieValidation) (accounts.AccountView, browser.SaveStatus, error) {
	existing, err := service.accountRepository.FindByRobloxUserID(ctx, validation.Identity.RobloxUserID)
	if err == nil {
		view, replaceError := service.replaceValidatedCookie(ctx, existing.ID, validation)
		return view, browser.SaveUpdated, replaceError
	}
	if !errors.Is(err, accounts.ErrNotFound) {
		return accounts.AccountView{}, browser.SaveFailed, mapRepositoryError(err)
	}
	view, record, err := service.accountRepository.Create(ctx, validation.Identity, validation.Cookie, validation.ExpiresAt, validation.Moderation != nil)
	if errors.Is(err, accounts.ErrDuplicateAccount) {
		duplicate, lookupError := service.accountRepository.FindByRobloxUserID(ctx, validation.Identity.RobloxUserID)
		return duplicate, browser.SaveDuplicate, lookupError
	}
	if err != nil {
		return accounts.AccountView{}, browser.SaveFailed, mapRepositoryError(err)
	}
	if err := service.sessions.Put(record, validation.BrowserID); err != nil {
		if removeError := service.accountRepository.Remove(context.WithoutCancel(ctx), view.ID); removeError != nil {
			service.logger.Error("browser account rollback failed", "operation", "browser-account-save", "account_id", view.ID, "error", removeError)
		}
		return accounts.AccountView{}, browser.SaveFailed, err
	}
	service.events.send("account:added", view)
	return view, browser.SaveCreated, nil
}

func (service *Service) SynchronizeBrowserCookie(ctx context.Context, accountID, expectedVersion int64, validation roblox.CookieValidation) (int64, error) {
	view, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return 0, mapRepositoryError(err)
	}
	if view.RobloxUserID != validation.Identity.RobloxUserID {
		return 0, errors.New("saved browser account identity changed")
	}
	record, err := service.accountRepository.RotateCookie(ctx, accountID, expectedVersion, validation.Cookie, validation.ExpiresAt)
	if err != nil {
		return 0, err
	}
	applyError := service.sessions.ApplyRotation(record)
	service.events.SessionChanged(accountID)
	if applyError != nil {
		return 0, applyError
	}
	logger := service.logger.With("operation", "browser-cookie-synchronization", "account_id", accountID)
	service.events.CredentialBackupChanged(service.refreshCredentialBackup(ctx, logger, true))
	return record.SecretVersion, nil
}

func (service *Service) replaceValidatedCookie(ctx context.Context, accountID int64, validated roblox.CookieValidation) (accounts.AccountView, error) {
	current, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return accounts.AccountView{}, mapRepositoryError(err)
	}
	if current.RobloxUserID != validated.Identity.RobloxUserID {
		return accounts.AccountView{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "cookie-replacement", Message: "The replacement cookie belongs to a different Roblox account."}
	}
	if err := service.vault.PurgeBackups(); err != nil {
		return accounts.AccountView{}, fmt.Errorf("remove backups containing the previous cookie: %w", err)
	}
	view, session, err := service.accountRepository.ReplaceCookie(ctx, accountID, validated.Identity, validated.Cookie, validated.ExpiresAt, validated.Moderation != nil)
	if err != nil {
		return accounts.AccountView{}, mapRepositoryError(err)
	}
	logger := service.logger.With("operation", "cookie-replacement", "account_id", accountID)
	if err := service.sessions.ApplyRotation(session); err != nil {
		logger.Warn("cookie saved but runtime session changed before reconciliation", "error", err)
	}
	service.events.send("account:updated", view)
	service.browser.DetachAccount(accountID, "The saved cookie changed elsewhere. This browser is no longer synchronized.")
	service.events.CredentialBackupChanged(service.refreshCredentialBackup(ctx, logger, false))
	return view, nil
}

func (service *Service) RemoveAccount(ctx context.Context, accountID int64) error {
	if err := service.vault.PurgeBackups(); err != nil {
		return fmt.Errorf("remove backups containing the account: %w", err)
	}
	if err := service.accountRepository.Remove(ctx, accountID); err != nil {
		return mapRepositoryError(err)
	}
	service.sessions.Remove(accountID)
	service.events.send("account:removed", struct {
		AccountID int64 `json:"accountId"`
	}{AccountID: accountID})
	service.browser.DetachAccount(accountID, "The saved account was removed. This browser is no longer synchronized.")
	if err := service.vault.CreateBackup(ctx); err != nil {
		return fmt.Errorf("the account was removed, but the fresh backup failed: %w", err)
	}
	return nil
}

func (service *Service) ListAccounts(ctx context.Context, query accounts.AccountQuery) (accounts.AccountPage, error) {
	page, err := service.accountRepository.Page(ctx, query)
	if err != nil {
		return accounts.AccountPage{}, mapRepositoryError(err)
	}
	for index := range page.Accounts {
		account := &page.Accounts[index]
		service.sessions.BrowserID(account.ID)
		if strings.HasPrefix(account.Username, "User ") {
			service.refreshAccountMetadata(account.ID, account.RobloxUserID)
		}
		if service.attachModeration(ctx, account) {
			service.refreshAccountModeration(account.ID, account.RobloxUserID)
		}
		service.attachVoiceSuspension(ctx, account)
	}
	return page, nil
}

// attachModeration adds the cached punishment of a moderated account to its
// view, and reports whether the cache had none.
func (service *Service) attachModeration(ctx context.Context, view *accounts.AccountView) bool {
	if view.State != accounts.StateModerated {
		return false
	}
	view.Moderation = service.users.CachedModeration(ctx, view.RobloxUserID)
	return view.Moderation == nil
}

func (service *Service) MoveAccount(ctx context.Context, accountID int64, beforeID, afterID *int64) error {
	return mapRepositoryError(service.accountRepository.Move(ctx, accountID, beforeID, afterID))
}

func (service *Service) ListTags(ctx context.Context) ([]accounts.TagView, error) {
	tags, err := service.accountRepository.ListTags(ctx)
	if err != nil {
		return nil, mapRepositoryError(err)
	}
	return tags, nil
}

func (service *Service) CreateTag(ctx context.Context, name string) (accounts.TagView, error) {
	tag, err := service.accountRepository.CreateTag(ctx, name)
	if err != nil {
		return accounts.TagView{}, mapRepositoryError(err)
	}
	service.events.send("tag:changed", tag)
	return tag, nil
}

func (service *Service) RemoveTag(ctx context.Context, tagID int64) error {
	if err := service.accountRepository.RemoveTag(ctx, tagID); err != nil {
		return mapRepositoryError(err)
	}
	service.events.send("tag:changed", struct {
		TagID int64 `json:"tagId"`
	}{TagID: tagID})
	return nil
}

func (service *Service) SetAccountsTag(ctx context.Context, accountIDs []int64, tagID int64, selected bool) ([]accounts.AccountView, error) {
	views, err := service.accountRepository.SetAccountsTag(ctx, accountIDs, tagID, selected)
	if err != nil {
		return nil, mapRepositoryError(err)
	}
	for _, view := range views {
		service.events.send("account:updated", view)
	}
	return views, nil
}

func (service *Service) GetAvatarHeadshots(ctx context.Context, userIDs []int64) ([]robloxservices.AvatarHeadshotView, error) {
	return service.thumbnails.AvatarHeadshots(ctx, userIDs)
}

func (service *Service) GetAccountPresences(ctx context.Context, accountID *int64) ([]robloxservices.UserPresence, error) {
	targets, err := service.presenceTargets(ctx, accountID)
	if err != nil {
		return nil, err
	}
	presences, err := service.presence.AccountPresences(ctx, targets)
	service.refreshOnlineVoiceSuspensions(ctx, targets, presences)
	if err != nil {
		service.logger.Warn("account presence update failed", "operation", "presence-read", "accounts", len(targets), "error", err)
	} else if len(presences) != len(targets) {
		service.logger.Warn("account presence update incomplete", "operation", "presence-read", "accounts", len(targets), "updated", len(presences))
	}
	return presences, err
}

// presenceTargets leaves out moderated accounts, because Roblox rejects their
// presence requests and shows them offline.
func (service *Service) presenceTargets(ctx context.Context, accountID *int64) ([]robloxservices.PresenceTarget, error) {
	var views []accounts.AccountView
	if accountID != nil {
		view, err := service.accountRepository.GetView(ctx, *accountID)
		if err != nil {
			return nil, mapRepositoryError(err)
		}
		views = []accounts.AccountView{view}
	} else {
		var err error
		if views, err = service.savedAccounts(ctx); err != nil {
			return nil, err
		}
	}
	targets := make([]robloxservices.PresenceTarget, 0, len(views))
	for _, view := range views {
		if view.State != accounts.StateModerated {
			targets = append(targets, robloxservices.PresenceTarget{AccountID: view.ID, RobloxUserID: view.RobloxUserID})
		}
	}
	return targets, nil
}

func (service *Service) savedAccounts(ctx context.Context) ([]accounts.AccountView, error) {
	views := []accounts.AccountView{}
	var cursor *accounts.AccountCursor
	for {
		page, err := service.accountRepository.Page(ctx, accounts.AccountQuery{Cursor: cursor})
		if err != nil {
			return nil, mapRepositoryError(err)
		}
		views = append(views, page.Accounts...)
		cursor = page.NextCursor
		if cursor == nil {
			return views, nil
		}
	}
}

func (service *Service) ValidateAccount(ctx context.Context, accountID int64) (accounts.AccountView, error) {
	current, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return accounts.AccountView{}, mapRepositoryError(err)
	}
	identity, err := service.users.Authenticated(ctx, accountID)
	if remote, ok := errors.AsType[*roblox.Error](err); ok && remote.Kind == roblox.KindModerated {
		return service.recordModeration(ctx, current)
	}
	if err != nil {
		return accounts.AccountView{}, err
	}
	snapshot, snapshotError := service.sessions.Snapshot(accountID)
	if snapshotError != nil {
		return accounts.AccountView{}, snapshotError
	}
	if identity.RobloxUserID != current.RobloxUserID {
		if err := service.accountRepository.MarkState(context.WithoutCancel(ctx), accountID, snapshot.SecretVersion, accounts.StateReauthRequired, "Roblox returned a different account identity."); err != nil && !errors.Is(err, accounts.ErrStaleSecret) {
			service.logger.Warn("account reauthentication state could not be saved", "operation", "account-validation", "account_id", accountID, "error", err)
		}
		service.sessions.UpdateState(accountID, snapshot.SecretVersion, accounts.StateReauthRequired)
		return accounts.AccountView{}, &roblox.Error{Kind: roblox.KindReauthRequired, Endpoint: "authenticated-user", Message: "The session identity changed. Replace the cookie."}
	}
	if err := service.accountRepository.RecordValidation(ctx, accountID, snapshot.SecretVersion, identity); err != nil && !errors.Is(err, accounts.ErrStaleSecret) {
		return accounts.AccountView{}, mapRepositoryError(err)
	}
	service.sessions.UpdateState(accountID, snapshot.SecretVersion, accounts.StateActive)
	view, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return accounts.AccountView{}, mapRepositoryError(err)
	}
	service.events.send("account:updated", view)
	return view, nil
}

// reactivationCacheDelay matches the wait of the Roblox website after a
// reactivation, because Roblox keeps answering with the moderation until its
// cache refreshes.
const reactivationCacheDelay = 2 * time.Second

// ReactivateAccount accepts the moderation of a warned account, or of a banned
// account whose ban ended, and validates the account again.
func (service *Service) ReactivateAccount(ctx context.Context, accountID int64) (accounts.AccountView, error) {
	err := service.editAccount(ctx, accountID, "account-reactivate", "Roblox did not reactivate this account. Check the ban end time and try again.", func(ctx context.Context, version int64) error {
		return service.users.Reactivate(ctx, accountID, version)
	})
	if err != nil {
		return accounts.AccountView{}, err
	}
	timer := time.NewTimer(reactivationCacheDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return accounts.AccountView{}, &roblox.Error{Kind: roblox.KindCancelled, Endpoint: "account-reactivate", Message: "The account was reactivated. Refresh it to see its status.", Cause: ctx.Err()}
	case <-timer.C:
	}
	view, err := service.ValidateAccount(ctx, accountID)
	if err != nil {
		return accounts.AccountView{}, err
	}
	if view.State == accounts.StateModerated {
		return accounts.AccountView{}, &roblox.Error{Kind: roblox.KindModerated, Endpoint: "account-reactivate", Message: "Roblox still reports this account as moderated. Try again in a minute."}
	}
	return view, nil
}

// recordModeration saves that Roblox reported the account as moderated, and
// caches its punishment.
func (service *Service) recordModeration(ctx context.Context, current accounts.AccountView) (accounts.AccountView, error) {
	snapshot, err := service.sessions.Snapshot(current.ID)
	if err != nil {
		return accounts.AccountView{}, err
	}
	if err := service.accountRepository.MarkModerated(ctx, current.ID, snapshot.SecretVersion); err != nil && !errors.Is(err, accounts.ErrStaleSecret) {
		return accounts.AccountView{}, mapRepositoryError(err)
	}
	service.sessions.UpdateState(current.ID, snapshot.SecretVersion, accounts.StateModerated)
	if _, err := service.users.Moderation(ctx, current.ID, snapshot.SecretVersion, current.RobloxUserID); err != nil && ctx.Err() == nil {
		service.logger.Warn("account moderation details unavailable; keeping the cached details", "operation", "account-validation", "account_id", current.ID, "error", err)
	}
	view, err := service.accountRepository.GetView(ctx, current.ID)
	if err != nil {
		return accounts.AccountView{}, mapRepositoryError(err)
	}
	service.attachModeration(ctx, &view)
	service.events.send("account:updated", view)
	return view, nil
}
