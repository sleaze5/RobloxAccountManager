package appservice

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

const (
	maxDisplayNameLength = 64
	maxDescriptionLength = 1000
	maxEmailLength       = 320
	maxPasswordLength    = 200
)

var (
	accountEditOperationID atomic.Uint64
	accountPinPattern      = regexp.MustCompile(`^[0-9]{4}$`)
)

type PinState string

const (
	PinOff      PinState = "off"
	PinLocked   PinState = "locked"
	PinUnlocked PinState = "unlocked"
)

type BlockedUser struct {
	UserID      int64  `json:"userId"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	ImageURL    string `json:"imageUrl"`
}

func pinState(pin robloxservices.AccountPin) PinState {
	switch {
	case !pin.Enabled:
		return PinOff
	case pin.Unlocked:
		return PinUnlocked
	default:
		return PinLocked
	}
}

func (service *Service) ChangeDisplayName(ctx context.Context, accountID int64, displayName string) error {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" || utf8.RuneCountInString(displayName) > maxDisplayNameLength || strings.ContainsFunc(displayName, unicode.IsControl) {
		return accountEditError("display-name-update", "Enter a valid display name.")
	}
	account, err := service.repo.GetView(ctx, accountID)
	if err != nil {
		return mapRepositoryError(err)
	}
	err = service.editAccount(ctx, accountID, "display-name-update", "Roblox rejected the display name. It can change once every 7 days.", func(ctx context.Context, version int64) error {
		return service.users.SetDisplayName(ctx, accountID, version, account.RobloxUserID, displayName)
	})
	if err != nil {
		return err
	}
	if _, err := service.ValidateAccount(ctx, accountID); err != nil {
		service.logger.Warn("display name refresh failed after change", "operation", "display-name-update", "account_id", accountID, "error", err)
	}
	return nil
}

func (service *Service) ChangeAccountEmail(ctx context.Context, accountID int64, password, email string) error {
	email = strings.TrimSpace(email)
	if !strings.Contains(email, "@") || len(email) > maxEmailLength || strings.ContainsFunc(email, unicode.IsSpace) {
		return accountEditError("email-update", "Enter a valid email address.")
	}
	if password == "" || len(password) > maxPasswordLength {
		return accountEditError("email-update", "Enter the account password.")
	}
	return service.editAccount(ctx, accountID, "email-update", "Roblox rejected the change. Check the password and try again.", func(ctx context.Context, version int64) error {
		return service.users.SetEmail(ctx, accountID, version, password, email)
	})
}

func (service *Service) ValidateAccountDescription(ctx context.Context, accountID int64, description string) (robloxservices.TextFilter, error) {
	description, err := normalizeDescription("description-validate", description)
	if err != nil {
		return robloxservices.TextFilter{}, err
	}
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "description-validate")
	if err != nil {
		return robloxservices.TextFilter{}, err
	}
	defer done()
	return service.users.FilterText(ctx, accountID, version, description)
}

func (service *Service) ChangeAccountDescription(ctx context.Context, accountID int64, description string) (string, error) {
	description, err := normalizeDescription("description-update", description)
	if err != nil {
		return "", err
	}
	var saved string
	err = service.editAccount(ctx, accountID, "description-update", "The account PIN is locked. Unlock it in account settings and try again.", func(ctx context.Context, version int64) error {
		var err error
		saved, err = service.users.SetDescription(ctx, accountID, version, description)
		return err
	})
	return saved, err
}

func normalizeDescription(operation, description string) (string, error) {
	description = strings.TrimSpace(description)
	if utf8.RuneCountInString(description) > maxDescriptionLength {
		return "", accountEditError(operation, fmt.Sprintf("Shorten the description to %d characters or fewer.", maxDescriptionLength))
	}
	return description, nil
}

func (service *Service) ChangeAccountPassword(ctx context.Context, accountID int64, currentPassword, newPassword string) error {
	if currentPassword == "" || newPassword == "" || len(currentPassword) > maxPasswordLength || len(newPassword) > maxPasswordLength {
		return accountEditError("password-change", "Enter the current and new passwords.")
	}
	unlock, err := service.lockAccountSettings(ctx, accountID)
	if err != nil {
		return err
	}
	defer unlock()
	return service.editAccount(ctx, accountID, "password-change", "Roblox rejected the change. Check the current password and try again.", func(ctx context.Context, version int64) error {
		account, err := service.repo.GetView(ctx, accountID)
		if err != nil {
			return mapRepositoryError(err)
		}
		if err := service.users.ChangePassword(ctx, accountID, version, currentPassword, newPassword); err != nil {
			return err
		}
		if !account.HasSavedPassword {
			return nil
		}
		// Cookie rotation can cancel the request context after Roblox commits the change.
		saveContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := service.storeAccountPassword(saveContext, accountID, newPassword); err != nil {
			return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "password-change", Message: "Roblox changed the password, but its saved copy could not be updated. Save the new password in the vault.", Cause: err}
		}
		return nil
	})
}

func (service *Service) UnlockAccountPin(ctx context.Context, accountID int64, pin string) error {
	if !accountPinPattern.MatchString(pin) {
		return accountEditError("account-pin-unlock", "Enter the 4-digit PIN.")
	}
	return service.editAccount(ctx, accountID, "account-pin-unlock", "Roblox rejected the PIN. Check it and try again.", func(ctx context.Context, version int64) error {
		return service.users.UnlockPin(ctx, accountID, version, pin)
	})
}

func (service *Service) UnblockUser(ctx context.Context, accountID, userID int64) error {
	return service.editAccount(ctx, accountID, "user-unblock", "", func(ctx context.Context, version int64) error {
		return service.users.Unblock(ctx, accountID, version, userID)
	})
}

func (service *Service) GetBlockedUsers(ctx context.Context, accountID int64) ([]BlockedUser, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "blocked-users")
	if err != nil {
		return nil, err
	}
	defer done()
	users, err := service.users.BlockedUsers(ctx, accountID, version)
	if err != nil {
		return nil, err
	}
	blocked := make([]BlockedUser, 0, len(users))
	ids := make([]int64, 0, len(users))
	for _, user := range users {
		blocked = append(blocked, BlockedUser{UserID: user.UserID, Username: user.Username, DisplayName: user.DisplayName})
		ids = append(ids, user.UserID)
	}
	if len(ids) == 0 {
		return blocked, nil
	}
	headshots, err := service.avatars.AvatarHeadshots(ctx, ids)
	if err != nil {
		service.logger.Warn("blocked user avatars unavailable", "operation", "blocked-users", "account_id", accountID, "error", err)
		return blocked, nil
	}
	images := make(map[int64]string, len(headshots))
	for _, headshot := range headshots {
		images[headshot.RobloxUserID] = headshot.ImageURL
	}
	for index := range blocked {
		blocked[index].ImageURL = images[blocked[index].UserID]
	}
	return blocked, nil
}

// rejected replaces Roblox's generic 403 message, which hides the reason.
func (service *Service) editAccount(ctx context.Context, accountID int64, operation, rejected string, edit func(context.Context, int64) error) error {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, operation)
	if err != nil {
		return err
	}
	defer done()
	logger := service.logger.With("operation_id", fmt.Sprintf("%s-%d", operation, accountEditOperationID.Add(1)), "operation", operation, "account_id", accountID)
	err = edit(ctx, version)
	if err == nil {
		logger.Info("account edit finished")
		return nil
	}
	var remote *roblox.Error
	if !errors.As(err, &remote) {
		logger.Warn("account edit failed", "error", err)
		return fmt.Errorf("%s: %w", operation, err)
	}
	logger.Warn("account edit failed", "error_kind", remote.Kind, "status", remote.Status, "roblox_code", remote.RobloxCode)
	switch {
	case remote.Kind == roblox.KindChallengeRequired:
		return &roblox.Error{Kind: remote.Kind, Endpoint: operation, Status: remote.Status, Message: "Roblox requires extra verification for this change. Make it on the Roblox website.", Cause: err}
	case remote.Kind == roblox.KindForbidden && remote.Status == http.StatusForbidden && rejected != "":
		return &roblox.Error{Kind: remote.Kind, Endpoint: operation, Status: remote.Status, Message: rejected, Cause: err}
	}
	return err
}

func accountEditError(operation, message string) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: operation, Message: message}
}
