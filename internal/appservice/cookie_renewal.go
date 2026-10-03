package appservice

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

var cookieRenewalOperationID atomic.Uint64

type CookieRenewalResult struct {
	Warning string `json:"warning,omitempty"`
}

func (service *Service) RenewCookie(ctx context.Context, accountID int64) (CookieRenewalResult, error) {
	if !service.vault.Unlocked() {
		return CookieRenewalResult{}, &roblox.Error{Kind: roblox.KindVaultLocked, Endpoint: "cookie-renewal", Message: "Unlock the vault before renewing a cookie."}
	}
	if _, running := service.cookieRenewals.LoadOrStore(accountID, struct{}{}); running {
		return CookieRenewalResult{}, &roblox.Error{Kind: roblox.KindForbidden, Endpoint: "cookie-renewal", Message: "This account's cookie is already being renewed."}
	}
	defer service.cookieRenewals.Delete(accountID)
	ctx, done := service.beginValidation(ctx)
	defer done()
	snapshot, err := service.sessions.Snapshot(accountID)
	if err != nil {
		return CookieRenewalResult{}, err
	}
	logger := service.logger.With("operation_id", fmt.Sprintf("renewal-%d", cookieRenewalOperationID.Add(1)), "operation", "cookie-renewal", "account_id", accountID)
	logger.Debug("cookie renewal started")
	if err := service.auth.RefreshCookie(ctx, accountID, snapshot.SecretVersion); err != nil {
		kind, status := roblox.KindProtocol, 0
		var remote *roblox.Error
		if errors.As(err, &remote) {
			kind, status = remote.Kind, remote.Status
		}
		logger.Warn("cookie renewal failed", "error_kind", kind, "status", status)
		if kind == roblox.KindTransport || kind == roblox.KindTimeout || kind == roblox.KindServer {
			return CookieRenewalResult{}, &roblox.Error{Kind: kind, Endpoint: "cookie-renewal", Status: status, Message: "The renewal could not be confirmed. If the saved session stops working, sign in again.", Cause: err}
		}
		return CookieRenewalResult{}, err
	}
	if service.browser != nil {
		service.browser.DetachAccount(accountID, "The saved cookie was renewed. Reopen this account's browser to use the new session.")
	}
	service.events.SessionChanged(accountID)
	result := CookieRenewalResult{}
	backupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if err := service.vault.PurgeBackups(); err != nil {
		logger.Warn("renewed cookie saved but old backups could not be removed", "error", err)
		result.Warning = "The renewed cookie is saved, but old backups could not be removed."
	} else if err := service.vault.CreateBackup(backupCtx); err != nil {
		logger.Warn("renewed cookie saved but backup failed", "error", err)
		result.Warning = "The renewed cookie is saved, but a fresh backup could not be created."
	}
	logger.Info("cookie renewal completed", "backup_warning", result.Warning != "")
	return result, nil
}
