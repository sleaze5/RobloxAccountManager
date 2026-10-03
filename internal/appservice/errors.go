package appservice

import (
	"errors"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/vault"
)

func mapRepositoryError(err error) error {
	switch {
	case errors.Is(err, vault.ErrLocked):
		return &roblox.Error{Kind: roblox.KindVaultLocked, Message: "The account vault is locked."}
	case errors.Is(err, accounts.ErrNotFound):
		return &roblox.Error{Kind: roblox.KindAccountNotFound, Message: "The account was not found."}
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
			Message:  "The account tag was not found.",
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

func safeVaultError(err error) error {
	if errors.Is(err, vault.ErrInvalidPassword) {
		return &roblox.Error{
			Kind:     roblox.KindForbidden,
			Endpoint: "vault-unlock",
			Message:  "The master password is incorrect.",
		}
	}
	message := "The vault could not be unlocked."
	switch {
	case errors.Is(err, vault.ErrIncomplete):
		message = "The vault files are incomplete. Restore a backup or reset the vault."
	case errors.Is(err, vault.ErrUnsupportedVersion):
		message = "The vault key-file version is unsupported. Restore a compatible backup."
	case errors.Is(err, vault.ErrUnsupportedSchema):
		message = "The vault database schema is unsupported. Restore a compatible backup."
	case errors.Is(err, vault.ErrVaultMismatch):
		message = "The database and security files belong to different vaults."
	case errors.Is(err, vault.ErrDPAPIUnavailable):
		message = "Automatic unlock is unavailable for this user. Enter the master password."
	}
	return &roblox.Error{
		Kind:     roblox.KindProtocol,
		Endpoint: "vault-unlock",
		Message:  message,
		Cause:    err,
	}
}
