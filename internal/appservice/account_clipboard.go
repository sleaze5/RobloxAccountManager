package appservice

import (
	"context"
	"strconv"
	"strings"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

type AccountCopyField string

const (
	CopyDisplayName AccountCopyField = "display-name"
	CopyUsername    AccountCopyField = "username"
	CopyUserID      AccountCopyField = "user-id"
	CopyCookie      AccountCopyField = "cookie"
)

func (service *Service) CopyAccountFields(ctx context.Context, accountIDs []int64, field AccountCopyField, copyText func(string) bool) error {
	values := make([]string, 0, len(accountIDs))
	sessions := make([]roblox.SessionSnapshot, 0, len(accountIDs))
	for _, accountID := range accountIDs {
		if field == CopyCookie {
			session, err := service.sessions.Snapshot(accountID)
			if err != nil {
				return mapRepositoryError(err)
			}
			sessions = append(sessions, session)
			values = append(values, session.Cookie)
			continue
		}
		value, err := service.accountFieldValue(ctx, accountID, field)
		if err != nil {
			return err
		}
		values = append(values, value)
	}
	for _, accountID := range accountIDs {
		if err := service.checkClipboardAccount(ctx, accountID); err != nil {
			return err
		}
	}
	for _, session := range sessions {
		if session.Context.Err() != nil || !service.sessions.VersionCurrent(session.AccountID, session.SecretVersion) {
			return clipboardError("An account session changed. Try again.")
		}
	}
	if !copyText(strings.Join(values, "\n")) {
		service.logger.Warn("account field clipboard write failed", "operation", "account-copy", "account_count", len(accountIDs), "field", field)
		return clipboardError("The account details could not be copied. Try again.")
	}
	return nil
}

func (service *Service) accountFieldValue(ctx context.Context, accountID int64, field AccountCopyField) (string, error) {
	view, err := service.repo.GetView(ctx, accountID)
	if err != nil {
		return "", mapRepositoryError(err)
	}
	var value string
	switch field {
	case CopyDisplayName:
		value = view.DisplayName
		if value == "" {
			value = view.Username
		}
	case CopyUsername:
		value = view.Username
	case CopyUserID:
		value = strconv.FormatInt(view.RobloxUserID, 10)
	default:
		return "", clipboardError("The requested account field is invalid.")
	}
	return value, nil
}

func (service *Service) checkClipboardAccount(ctx context.Context, accountID int64) error {
	if ctx.Err() != nil || !service.vault.Unlocked() {
		return clipboardError("The copy was cancelled or the vault is locked.")
	}
	_, err := service.repo.GetView(ctx, accountID)
	return mapRepositoryError(err)
}

func clipboardError(message string) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "account-clipboard", Message: message}
}
