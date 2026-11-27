package appservice

import (
	"context"
	"errors"
	"log/slog"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/vault"
)

func mapRepositoryError(err error) error {
	switch {
	case errors.Is(err, vault.ErrLocked):
		return &roblox.Error{Kind: roblox.KindVaultLocked, Message: "The account vault is locked."}
	case errors.Is(err, accounts.ErrNotFound):
		return &roblox.Error{Kind: roblox.KindAccountNotFound, Message: "Could not find the account."}
	case errors.Is(err, accounts.ErrDuplicateAccount):
		return &roblox.Error{
			Kind:     roblox.KindProtocol,
			Endpoint: "cookie-import",
			Message:  "This Roblox account already exists. Remove it before importing it again.",
		}
	case errors.Is(err, accounts.ErrTagNotFound):
		return &roblox.Error{
			Kind:     roblox.KindProtocol,
			Endpoint: "account-tag",
			Message:  "Could not find the tag.",
		}
	case errors.Is(err, accounts.ErrDuplicateTag):
		return &roblox.Error{
			Kind:     roblox.KindProtocol,
			Endpoint: "account-tag",
			Message:  "A tag with this name already exists.",
		}
	case errors.Is(err, accounts.ErrInvalidTagName):
		return &roblox.Error{
			Kind:     roblox.KindProtocol,
			Endpoint: "account-tag",
			Message:  err.Error(),
		}
	case errors.Is(err, accounts.ErrInvalidOrder):
		return &roblox.Error{
			Kind:     roblox.KindProtocol,
			Endpoint: "account-order",
			Message:  "The account order is invalid.",
		}
	default:
		return err
	}
}

// keyringUnavailableMessage explains a failed automatic unlock setup. The
// operating system keyring protects the automatic unlock key.
const keyringUnavailableMessage = "The system keyring is not available, so automatic unlock cannot work."

const legacyVaultMessage = "An older version made this vault, and this version cannot open it. Open it in version 0.9.2, then update to this version."

func safeVaultError(err error) error {
	if errors.Is(err, vault.ErrInvalidPassword) {
		return &roblox.Error{
			Kind:     roblox.KindForbidden,
			Endpoint: "vault-unlock",
			Message:  "The master password is incorrect.",
		}
	}
	return &roblox.Error{
		Kind:     roblox.KindProtocol,
		Endpoint: "vault-unlock",
		Message:  vaultFailureMessage(err, "Could not unlock the vault."),
		Cause:    err,
	}
}

func vaultFailureMessage(err error, fallback string) string {
	switch {
	case errors.Is(err, vault.ErrIncomplete):
		return "The vault files are incomplete. Restore a backup or reset the vault."
	case errors.Is(err, vault.ErrLegacyVault):
		return legacyVaultMessage
	case errors.Is(err, vault.ErrUnsupportedVersion):
		return "The vault key-file version is unsupported. Restore a compatible backup."
	case errors.Is(err, vault.ErrUnsupportedSchema):
		return "The vault database schema is unsupported. Restore a compatible backup."
	case errors.Is(err, vault.ErrVaultMismatch):
		return "The database and security files belong to different vaults. Restore a backup or reset the vault."
	case errors.Is(err, vault.ErrAutomaticUnlockUnavailable):
		return "Automatic unlock is unavailable for this user. Enter the master password."
	default:
		return fallback
	}
}

// operationFailure logs a failed Roblox operation. An error that is not a
// *roblox.Error becomes one with the fallback message, so the UI never shows
// internal details.
func operationFailure(ctx context.Context, logger *slog.Logger, err error, endpoint, fallback string) error {
	logger.WarnContext(ctx, endpoint+" operation failed", "error", err)
	if _, ok := errors.AsType[*roblox.Error](err); ok {
		return err
	}
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: fallback, Cause: err}
}
