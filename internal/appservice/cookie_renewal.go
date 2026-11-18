package appservice

import (
	"context"
	"errors"

	"github.com/sleaze5/RobloxAccountManager/internal/logging"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

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
	ctx = logging.StartOperation(ctx, "renewal")
	logger := service.logger.With("operation", "cookie-renewal", "account_id", accountID)
	logger.DebugContext(ctx, "cookie renewal started")
	if err := service.authentication.RefreshCookie(ctx, accountID, snapshot.SecretVersion); err != nil {
		kind, status := roblox.KindProtocol, 0
		if remote, ok := errors.AsType[*roblox.Error](err); ok {
			kind, status = remote.Kind, remote.Status
		}
		logger.WarnContext(ctx, "could not renew the cookie", "error_kind", kind, "status", status)
		if kind == roblox.KindTransport || kind == roblox.KindTimeout || kind == roblox.KindServer {
			return CookieRenewalResult{}, &roblox.Error{Kind: kind, Endpoint: "cookie-renewal", Status: status, Message: "Could not confirm the renewal. If the saved session stops working, sign in again.", Cause: err}
		}
		return CookieRenewalResult{}, err
	}
	service.browser.DetachAccount(accountID, "The saved cookie was renewed. Reopen this account's browser to use the new session.")
	service.events.SessionChanged(accountID)
	renewalResult := CookieRenewalResult{Warning: service.refreshCredentialBackup(ctx, logger, true)}
	logger.InfoContext(ctx, "cookie renewal completed", "backup_warning", renewalResult.Warning != "")
	return renewalResult, nil
}
