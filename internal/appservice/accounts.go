package appservice

import (
	"context"
	"errors"
	"fmt"

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

func (service *Service) ReplaceCookie(ctx context.Context, accountID int64, input string) (accounts.AccountView, error) {
	cookie, err := accounts.NormalizeCookie(input)
	if err != nil {
		return accounts.AccountView{}, &roblox.Error{Kind: roblox.KindInvalidCookieInput, Endpoint: "cookie-replacement", Message: err.Error()}
	}
	validated, err := service.users.ValidateCookie(ctx, cookie, service.sessions.BrowserID(accountID))
	if err != nil {
		return accounts.AccountView{}, err
	}
	return service.replaceValidatedCookie(ctx, accountID, validated)
}

func (service *Service) BrowserAccount(ctx context.Context, accountID int64) (accounts.AccountView, roblox.SessionSnapshot, error) {
	view, err := service.repo.GetView(ctx, accountID)
	if err != nil {
		return accounts.AccountView{}, roblox.SessionSnapshot{}, mapRepositoryError(err)
	}
	record, err := service.sessions.Snapshot(accountID)
	return view, record, mapRepositoryError(err)
}

func (service *Service) BrowserCandidateAccount(ctx context.Context, robloxUserID int64) (accounts.AccountView, bool) {
	view, err := service.repo.FindByRobloxUserID(ctx, robloxUserID)
	return view, err == nil
}

func (service *Service) FindAccountByRobloxUserID(ctx context.Context, robloxUserID int64) (*accounts.AccountView, error) {
	view, err := service.repo.FindByRobloxUserID(ctx, robloxUserID)
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
	existing, err := service.repo.FindByRobloxUserID(ctx, validation.Identity.RobloxUserID)
	if err == nil {
		validation.BrowserID = service.sessions.BrowserID(existing.ID)
	} else if !errors.Is(err, accounts.ErrNotFound) {
		return roblox.CookieValidation{}, mapRepositoryError(err)
	}
	return validation, nil
}

func (service *Service) BrowserAvatar(ctx context.Context, robloxUserID int64) string {
	if service.avatars == nil {
		return ""
	}
	views, err := service.avatars.AvatarHeadshots(ctx, []int64{robloxUserID})
	if err != nil || len(views) == 0 {
		return ""
	}
	return views[0].ImageURL
}

func (service *Service) CommitBrowserCandidate(ctx context.Context, validation roblox.CookieValidation) (accounts.AccountView, browser.SaveStatus, error) {
	existing, err := service.repo.FindByRobloxUserID(ctx, validation.Identity.RobloxUserID)
	if err == nil {
		view, replaceErr := service.replaceValidatedCookie(ctx, existing.ID, validation)
		return view, browser.SaveUpdated, replaceErr
	}
	if !errors.Is(err, accounts.ErrNotFound) {
		return accounts.AccountView{}, browser.SaveFailed, mapRepositoryError(err)
	}
	view, record, err := service.repo.Create(ctx, validation.Identity, validation.Cookie, validation.ExpiresAt)
	if errors.Is(err, accounts.ErrDuplicateAccount) {
		duplicate, lookupErr := service.repo.FindByRobloxUserID(ctx, validation.Identity.RobloxUserID)
		return duplicate, browser.SaveDuplicate, lookupErr
	}
	if err != nil {
		return accounts.AccountView{}, browser.SaveFailed, mapRepositoryError(err)
	}
	if err := service.sessions.Put(record, validation.BrowserID); err != nil {
		_ = service.repo.Remove(context.WithoutCancel(ctx), view.ID)
		return accounts.AccountView{}, browser.SaveFailed, err
	}
	service.events.send("account:added", view)
	return view, browser.SaveCreated, nil
}

func (service *Service) SynchronizeBrowserCookie(ctx context.Context, accountID, expectedVersion int64, validation roblox.CookieValidation) (int64, error) {
	view, err := service.repo.GetView(ctx, accountID)
	if err != nil || view.RobloxUserID != validation.Identity.RobloxUserID {
		return 0, errors.New("saved browser account identity changed")
	}
	record, _, err := service.repo.RotateCookie(ctx, accountID, expectedVersion, validation.Cookie, validation.ExpiresAt)
	if err != nil {
		return 0, err
	}
	if err := service.sessions.ApplyRotation(record); err != nil {
		return 0, err
	}
	if err := service.vault.PurgeBackups(); err != nil {
		return 0, err
	}
	if err := service.vault.CreateBackup(ctx); err != nil {
		return 0, err
	}
	service.events.send("account:updated", view)
	return record.SecretVersion, nil
}

func (service *Service) replaceValidatedCookie(ctx context.Context, accountID int64, validated roblox.CookieValidation) (accounts.AccountView, error) {
	current, err := service.repo.GetView(ctx, accountID)
	if err != nil {
		return accounts.AccountView{}, mapRepositoryError(err)
	}
	if current.RobloxUserID != validated.Identity.RobloxUserID {
		return accounts.AccountView{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "cookie-replacement", Message: "The replacement cookie belongs to a different Roblox account."}
	}
	if err := service.vault.PurgeBackups(); err != nil {
		return accounts.AccountView{}, fmt.Errorf("remove backups containing the previous cookie: %w", err)
	}
	view, session, err := service.repo.ReplaceCookie(ctx, accountID, validated.Identity, validated.Cookie, validated.ExpiresAt)
	if err != nil {
		return accounts.AccountView{}, mapRepositoryError(err)
	}
	if err := service.sessions.ApplyRotation(session); err != nil {
		return accounts.AccountView{}, err
	}
	if err := service.vault.CreateBackup(ctx); err != nil {
		return view, fmt.Errorf("the cookie was replaced, but the fresh backup failed: %w", err)
	}
	service.events.send("account:updated", view)
	if service.browser != nil {
		service.browser.DetachAccount(accountID, "The saved cookie changed elsewhere. This browser is no longer synchronized.")
	}
	return view, nil
}

func (service *Service) RemoveAccount(ctx context.Context, accountID int64) error {
	if err := service.vault.PurgeBackups(); err != nil {
		return fmt.Errorf("remove backups containing the account: %w", err)
	}
	if err := service.repo.Remove(ctx, accountID); err != nil {
		return mapRepositoryError(err)
	}
	service.sessions.Remove(accountID)
	service.events.send("account:removed", struct {
		AccountID int64 `json:"accountId"`
	}{AccountID: accountID})
	if service.browser != nil {
		service.browser.DetachAccount(accountID, "The saved account was removed. This browser is no longer synchronized.")
	}
	if err := service.vault.CreateBackup(ctx); err != nil {
		return fmt.Errorf("the account was removed, but the fresh backup failed: %w", err)
	}
	return nil
}

func (service *Service) ListAccounts(ctx context.Context, query accounts.AccountQuery) (accounts.AccountPage, error) {
	page, err := service.repo.Page(ctx, query)
	if err != nil {
		return accounts.AccountPage{}, mapRepositoryError(err)
	}
	for _, account := range page.Accounts {
		service.sessions.BrowserID(account.ID)
	}
	return page, nil
}

func (service *Service) MoveAccount(ctx context.Context, accountID int64, beforeID, afterID *int64) error {
	return mapRepositoryError(service.repo.Move(ctx, accountID, beforeID, afterID))
}

func (service *Service) ListTags(ctx context.Context) ([]accounts.TagView, error) {
	tags, err := service.repo.ListTags(ctx)
	if err != nil {
		return nil, mapRepositoryError(err)
	}
	return tags, nil
}

func (service *Service) CreateTag(ctx context.Context, name string) (accounts.TagView, error) {
	tag, err := service.repo.CreateTag(ctx, name)
	if err != nil {
		return accounts.TagView{}, mapRepositoryError(err)
	}
	service.events.send("tag:changed", tag)
	return tag, nil
}

func (service *Service) RemoveTag(ctx context.Context, tagID int64) error {
	if err := service.repo.RemoveTag(ctx, tagID); err != nil {
		return mapRepositoryError(err)
	}
	service.events.send("tag:changed", struct {
		TagID int64 `json:"tagId"`
	}{TagID: tagID})
	return nil
}

func (service *Service) SetAccountsTag(ctx context.Context, accountIDs []int64, tagID int64, selected bool) ([]accounts.AccountView, error) {
	views, err := service.repo.SetAccountsTag(ctx, accountIDs, tagID, selected)
	if err != nil {
		return nil, mapRepositoryError(err)
	}
	for _, view := range views {
		service.events.send("account:updated", view)
	}
	return views, nil
}

func (service *Service) GetAvatarHeadshots(ctx context.Context, userIDs []int64) ([]robloxservices.AvatarHeadshotView, error) {
	if service.avatars == nil {
		return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "avatar-headshots", Message: "Avatar headshots are unavailable."}
	}
	return service.avatars.AvatarHeadshots(ctx, userIDs)
}

func (service *Service) GetAccountPresences(ctx context.Context, accountID *int64) ([]robloxservices.UserPresence, error) {
	if service.presence == nil {
		return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "user-presences", Message: "Account presence is unavailable."}
	}
	targets, err := service.presenceTargets(ctx, accountID)
	if err != nil {
		return nil, err
	}
	presences, err := service.presence.AccountPresences(ctx, targets)
	if err != nil {
		service.logger.Warn("account presence update failed", "operation", "presence-read", "accounts", len(targets), "error", err)
	} else if len(presences) != len(targets) {
		service.logger.Warn("account presence update incomplete", "operation", "presence-read", "accounts", len(targets), "updated", len(presences))
	}
	return presences, err
}

func (service *Service) presenceTargets(ctx context.Context, accountID *int64) ([]robloxservices.PresenceTarget, error) {
	if accountID != nil {
		view, err := service.repo.GetView(ctx, *accountID)
		if err != nil {
			return nil, mapRepositoryError(err)
		}
		return []robloxservices.PresenceTarget{{AccountID: view.ID, RobloxUserID: view.RobloxUserID}}, nil
	}
	targets := []robloxservices.PresenceTarget{}
	var cursor *accounts.AccountCursor
	for {
		page, err := service.repo.Page(ctx, accounts.AccountQuery{Cursor: cursor})
		if err != nil {
			return nil, mapRepositoryError(err)
		}
		for _, view := range page.Accounts {
			targets = append(targets, robloxservices.PresenceTarget{AccountID: view.ID, RobloxUserID: view.RobloxUserID})
		}
		cursor = page.NextCursor
		if cursor == nil {
			break
		}
	}
	return targets, nil
}

func (service *Service) ValidateAccount(ctx context.Context, accountID int64) (accounts.AccountView, error) {
	current, err := service.repo.GetView(ctx, accountID)
	if err != nil {
		return accounts.AccountView{}, mapRepositoryError(err)
	}
	identity, err := service.users.Authenticated(ctx, accountID)
	if err != nil {
		return accounts.AccountView{}, err
	}
	snapshot, snapshotErr := service.sessions.Snapshot(accountID)
	if snapshotErr != nil {
		return accounts.AccountView{}, snapshotErr
	}
	if identity.RobloxUserID != current.RobloxUserID {
		_ = service.repo.MarkState(context.WithoutCancel(ctx), accountID, snapshot.SecretVersion, accounts.StateReauthRequired, "Roblox returned a different account identity.")
		service.sessions.UpdateState(accountID, snapshot.SecretVersion, accounts.StateReauthRequired)
		return accounts.AccountView{}, &roblox.Error{Kind: roblox.KindReauthRequired, Endpoint: "authenticated-user", Message: "The session identity changed. Replace the cookie."}
	}
	if err := service.repo.RecordValidation(ctx, accountID, snapshot.SecretVersion, identity); err != nil && !errors.Is(err, accounts.ErrStaleSecret) {
		return accounts.AccountView{}, mapRepositoryError(err)
	}
	service.sessions.UpdateState(accountID, snapshot.SecretVersion, accounts.StateActive)
	view, err := service.repo.GetView(ctx, accountID)
	if err != nil {
		return accounts.AccountView{}, mapRepositoryError(err)
	}
	service.events.send("account:updated", view)
	return view, nil
}
