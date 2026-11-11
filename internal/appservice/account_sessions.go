package appservice

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/sleaze5/RobloxAccountManager/internal/logging"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

// AccountSession is never persisted or logged.
type AccountSession struct {
	Agent        string `json:"agent"`
	Browser      string `json:"browser"`
	OS           string `json:"os"`
	Location     string `json:"location"`
	IP           string `json:"ip"`
	LastActiveMs int64  `json:"lastActiveMs"`
	Current      bool   `json:"current"`
}

func (service *Service) GetAccountSessions(ctx context.Context, accountID int64) ([]AccountSession, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "account-sessions")
	if err != nil {
		return nil, err
	}
	defer done()
	sessions, err := service.users.Sessions(ctx, accountID, version)
	if err != nil {
		return nil, err
	}
	views := make([]AccountSession, 0, len(sessions))
	for _, session := range sessions {
		view := AccountSession{
			Agent:    session.Agent,
			Browser:  session.Browser,
			OS:       session.OS,
			Location: strings.Join(slices.DeleteFunc([]string{session.City, session.Subdivision, session.Country}, func(part string) bool { return part == "" }), ", "),
			IP:       session.IP,
			Current:  session.Current,
		}
		if !session.LastActive.IsZero() {
			view.LastActiveMs = session.LastActive.UnixMilli()
		}
		views = append(views, view)
	}
	slices.SortStableFunc(views, func(left, right AccountSession) int {
		if left.Current != right.Current {
			if left.Current {
				return -1
			}
			return 1
		}
		return cmp.Compare(right.LastActiveMs, left.LastActiveMs)
	})
	return views, nil
}

func (service *Service) SignOutOtherSessions(ctx context.Context, accountID int64) (CookieRenewalResult, error) {
	if _, running := service.cookieRenewals.LoadOrStore(accountID, struct{}{}); running {
		return CookieRenewalResult{}, &roblox.Error{Kind: roblox.KindForbidden, Endpoint: "sessions-sign-out", Message: "This account's cookie is already being replaced. Try again in a moment."}
	}
	defer service.cookieRenewals.Delete(accountID)
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "sessions-sign-out")
	if err != nil {
		return CookieRenewalResult{}, err
	}
	defer done()
	logger := service.logger.With("operation_id", logging.NewOperationID("sessions-sign-out"), "operation", "sessions-sign-out", "account_id", accountID)
	if err := service.authentication.SignOutOtherSessions(ctx, accountID, version); err != nil {
		kind, status := roblox.KindProtocol, 0
		if remote, ok := errors.AsType[*roblox.Error](err); ok {
			kind, status = remote.Kind, remote.Status
		}
		logger.Warn("signing out other sessions failed", "error_kind", kind, "status", status)
		if kind == roblox.KindChallengeRequired {
			return CookieRenewalResult{}, &roblox.Error{Kind: kind, Endpoint: "sessions-sign-out", Status: status, Message: "Roblox requires extra verification to sign out other sessions. Do it on the Roblox website.", Cause: err}
		}
		return CookieRenewalResult{}, err
	}
	service.browser.DetachAccount(accountID, "Other sessions were signed out. Reopen this account's browser to use the new session.")
	service.events.SessionChanged(accountID)
	renewalResult := CookieRenewalResult{Warning: service.refreshCredentialBackup(ctx, logger, false)}
	logger.Info("signed out other sessions", "backup_warning", renewalResult.Warning != "")
	return renewalResult, nil
}
