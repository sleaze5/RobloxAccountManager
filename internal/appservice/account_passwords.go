package appservice

import (
	"context"
	"errors"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

func (service *Service) SaveAccountPassword(ctx context.Context, accountID int64, password string) error {
	if err := validateSavedPassword(password); err != nil {
		return err
	}
	ctx, _, done, err := service.beginAccountSession(ctx, accountID, "password-save")
	if err != nil {
		return err
	}
	defer done()
	unlock, err := service.lockAccountSettings(ctx, accountID)
	if err != nil {
		return err
	}
	defer unlock()
	return service.storeAccountPassword(ctx, accountID, password)
}

func (service *Service) storeAccountPassword(ctx context.Context, accountID int64, password string) error {
	if err := service.repo.SetPassword(ctx, accountID, password); err != nil {
		service.logger.Error("account password could not be saved", "operation", "password-save", "account_id", accountID, "error", err)
		return mapRepositoryError(err)
	}
	service.logger.Info("account password saved", "operation", "password-save", "account_id", accountID)
	view, err := service.repo.GetView(ctx, accountID)
	if err != nil {
		return mapRepositoryError(err)
	}
	service.events.send("account:updated", view)
	return nil
}

func (service *Service) GetAccountPassword(ctx context.Context, accountID int64) (string, error) {
	password, err := service.repo.GetPassword(ctx, accountID)
	if err != nil {
		service.logger.Error("saved account password could not be read", "operation", "password-read", "account_id", accountID, "error", err)
		return "", mapRepositoryError(err)
	}
	return password, nil
}

func (service *Service) ValidateAccountPassword(ctx context.Context, accountID int64, password string) (bool, error) {
	if err := validateSavedPassword(password); err != nil {
		return false, err
	}
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "password-validate")
	if err != nil {
		return false, err
	}
	defer done()
	unlock, err := service.lockAccountSettings(ctx, accountID)
	if err != nil {
		return false, err
	}
	defer unlock()
	valid, err := service.users.ValidatePassword(ctx, accountID, version, password)
	if err == nil {
		return valid, nil
	}
	message := "Could not validate the password. Check the account on the Roblox website."
	kind := roblox.KindProtocol
	var remote *roblox.Error
	if errors.As(err, &remote) {
		kind = remote.Kind
		service.logger.Warn("account password validation unavailable", "operation", "password-validate", "account_id", accountID, "error_kind", remote.Kind, "status", remote.Status, "roblox_code", remote.RobloxCode)
		switch remote.Kind {
		case roblox.KindChallengeRequired:
			message = "Could not validate: Roblox requires extra verification. Check the password on the Roblox website."
		case roblox.KindRateLimited:
			message = "Could not validate: Roblox is rate limiting requests. Wait before trying again."
		case roblox.KindReauthRequired, roblox.KindInvalidSession:
			message = "Could not validate: the Roblox session has expired. Sign in again first."
		case roblox.KindVaultLocked:
			message = "Could not validate: unlock the vault first."
		case roblox.KindCancelled:
			message = "Password validation was cancelled because the account session changed."
		case roblox.KindTransport, roblox.KindTimeout, roblox.KindServer:
			message = "Could not validate: Roblox could not be reached. Try again later."
		}
	}
	return false, &roblox.Error{Kind: kind, Endpoint: "password-validate", Message: message, Cause: err}
}

func validateSavedPassword(password string) error {
	if password == "" || len(password) > maxPasswordLength || !utf8.ValidString(password) {
		return accountEditError("account-password", "The password is empty, invalid, or too long.")
	}
	return nil
}
