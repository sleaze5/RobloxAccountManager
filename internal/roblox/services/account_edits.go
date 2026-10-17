package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	blockedUsersPageSize = 50
	maxBlockedUsersPages = 20
	userNamesBatchSize   = 100
)

type AccountPin struct {
	Enabled  bool
	Unlocked bool
}

type TextModeration string

const (
	TextModerationNone    TextModeration = "none"
	TextModerationPartial TextModeration = "partial"
	TextModerationFull    TextModeration = "full"
)

type TextFilter struct {
	Moderation   TextModeration `json:"moderation"`
	FilteredText string         `json:"filteredText"`
}

func (service *Users) SetDisplayName(ctx context.Context, accountID, version, userID int64, displayName string) error {
	address := "https://users.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/display-names"
	_, err := service.writeAccount(ctx, accountID, version, "display-name-update", http.MethodPatch, address, map[string]string{"newDisplayName": displayName})
	return err
}

func (service *Users) SetDescription(ctx context.Context, accountID, version int64, description string) (string, error) {
	response, err := service.writeAccount(ctx, accountID, version, "description-update", http.MethodPost, "https://users.roblox.com/v1/description", map[string]string{"description": description})
	if err != nil {
		return "", err
	}
	var result struct {
		Description *string `json:"description"`
	}
	if err := json.Unmarshal(response.Body, &result); err != nil {
		return "", &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "description-update", Status: response.Status, Message: "Roblox did not confirm the description. Refresh the profile to check.", Cause: err}
	}
	if result.Description == nil || len(*result.Description) > 4096 {
		return "", &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "description-update", Status: response.Status, Message: "Roblox did not confirm the description. Refresh the profile to check."}
	}
	return strings.TrimSpace(*result.Description), nil
}

// Roblox has no profile text preview. RoSeal and RoValra use the game update filter.
// Shape verified against signed-in responses on 2026-10-17.
func (service *Users) FilterText(ctx context.Context, accountID, version int64, text string) (TextFilter, error) {
	response, err := service.writeAccount(ctx, accountID, version, "description-validate", http.MethodPost, "https://apis.roblox.com/game-update-notifications/v1/filter", text)
	if err != nil {
		return TextFilter{}, err
	}
	var result struct {
		FilteredText    *string `json:"filteredGameUpdateText"`
		ModerationLevel int     `json:"moderationLevel"`
	}
	levels := map[int]TextModeration{1: TextModerationNone, 2: TextModerationPartial, 3: TextModerationFull}
	err = json.Unmarshal(response.Body, &result)
	moderation, known := levels[result.ModerationLevel]
	if err != nil || !known || result.FilteredText == nil {
		return TextFilter{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "description-validate", Status: response.Status, Message: "Roblox returned an invalid filter result. Validate the description again.", Cause: err}
	}
	return TextFilter{Moderation: moderation, FilteredText: *result.FilteredText}, nil
}

func (service *Users) SetEmail(ctx context.Context, accountID, version int64, password, email string) error {
	_, err := service.writeAccount(ctx, accountID, version, "email-update", http.MethodPost, "https://accountsettings.roblox.com/v1/email", url.Values{"password": {password}, "emailAddress": {email}})
	return err
}

// Roblox replaces the session cookie in this response. The client saves it.
func (service *Users) ChangePassword(ctx context.Context, accountID, version int64, currentPassword, newPassword string) error {
	_, err := service.writeAccount(ctx, accountID, version, "password-change", http.MethodPost, "https://auth.roblox.com/v2/user/passwords/change", url.Values{"currentPassword": {currentPassword}, "newPassword": {newPassword}})
	return err
}

func (service *Users) ValidatePassword(ctx context.Context, accountID, version int64, password string) (bool, error) {
	// With identical passwords, Roblox rejects reuse (7) after checking the
	// current password (8 on mismatch). Never retry or submit a different new password.
	_, err := service.writeAccount(ctx, accountID, version, "password-validate", http.MethodPost, "https://auth.roblox.com/v2/user/passwords/change", url.Values{"currentPassword": {password}, "newPassword": {password}})
	var remote *roblox.Error
	if errors.As(err, &remote) &&
		(remote.Kind == roblox.KindForbidden || remote.Kind == roblox.KindProtocol) &&
		(remote.Status == http.StatusBadRequest || remote.Status == http.StatusForbidden) {
		switch remote.RobloxCode {
		case 7:
			return true, nil
		case 8:
			return false, nil
		}
	}
	if err != nil {
		return false, err
	}
	return false, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "password-validate", Message: "Roblox accepted the password request unexpectedly. Check the account on the Roblox website before trying again."}
}

func (service *Users) Pin(ctx context.Context, accountID, version int64) (AccountPin, error) {
	var result struct {
		Enabled       *bool    `json:"isEnabled"`
		UnlockedUntil *float64 `json:"unlockedUntil"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "account-pin", "https://auth.roblox.com/v1/account/pin", &result); err != nil {
		return AccountPin{}, err
	}
	if result.Enabled == nil {
		return AccountPin{}, invalidProfileDetails("account-pin")
	}
	return AccountPin{Enabled: *result.Enabled, Unlocked: *result.Enabled && result.UnlockedUntil != nil && *result.UnlockedUntil > 0}, nil
}

func (service *Users) UnlockPin(ctx context.Context, accountID, version int64, pin string) error {
	_, err := service.writeAccount(ctx, accountID, version, "account-pin-unlock", http.MethodPost, "https://auth.roblox.com/v1/account/pin/unlock", url.Values{"pin": {pin}})
	return err
}

// Shape verified against signed-in responses on 2026-10-12. Item fields were
// not observed, so names come from the users API.
func (service *Users) BlockedUsers(ctx context.Context, accountID, version int64) ([]UserView, error) {
	ids := []int64{}
	seen := map[int64]struct{}{}
	cursor := ""
	for range maxBlockedUsersPages {
		var result struct {
			Data *struct {
				BlockedUserIDs []int64 `json:"blockedUserIds"`
				BlockedUsers   []struct {
					UserID int64 `json:"userId"`
				} `json:"blockedUsers"`
				Cursor *string `json:"cursor"`
			} `json:"data"`
		}
		query := url.Values{"count": {strconv.Itoa(blockedUsersPageSize)}}
		if cursor != "" {
			query.Set("cursor", cursor)
		}
		if err := service.readAccountDetails(ctx, accountID, version, "blocked-users", "https://apis.roblox.com/user-blocking-api/v1/users/get-blocked-users?"+query.Encode(), &result); err != nil {
			return nil, err
		}
		if result.Data == nil {
			return nil, invalidProfileDetails("blocked-users")
		}
		page := result.Data.BlockedUserIDs
		for _, user := range result.Data.BlockedUsers {
			page = append(page, user.UserID)
		}
		for _, id := range page {
			if _, exists := seen[id]; id > 0 && !exists {
				seen[id] = struct{}{}
				ids = append(ids, id)
			}
		}
		if result.Data.Cursor == nil || *result.Data.Cursor == "" {
			return service.names(ctx, ids)
		}
		cursor = *result.Data.Cursor
	}
	return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "blocked-users", Message: "Roblox returned too many blocked users to list."}
}

func (service *Users) Unblock(ctx context.Context, accountID, version, userID int64) error {
	address := "https://apis.roblox.com/user-blocking-api/v1/users/" + strconv.FormatInt(userID, 10) + "/unblock-user"
	response, err := service.writeAccount(ctx, accountID, version, "user-unblock", http.MethodPost, address, nil)
	if err != nil {
		return err
	}
	var result struct {
		Error *string `json:"error"`
	}
	if json.Unmarshal(response.Body, &result) == nil && result.Error != nil && *result.Error != "" {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "user-unblock", Status: response.Status, Message: *result.Error}
	}
	return nil
}

// Users that Roblox no longer returns keep only their ID.
func (service *Users) names(ctx context.Context, ids []int64) ([]UserView, error) {
	users := make([]UserView, 0, len(ids))
	endpoint, _ := url.Parse("https://users.roblox.com/v1/users")
	for start := 0; start < len(ids); start += userNamesBatchSize {
		batch := ids[start:min(start+userNamesBatchSize, len(ids))]
		body, err := json.Marshal(struct {
			UserIDs            []int64 `json:"userIds"`
			ExcludeBannedUsers bool    `json:"excludeBannedUsers"`
		}{UserIDs: batch})
		if err != nil {
			return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "user-names", Message: "The request could not be encoded.", Cause: err}
		}
		response, err := service.client.Do(ctx, roblox.Request{
			Endpoint: "user-names", Method: http.MethodPost, URL: endpoint,
			Body: body, ContentType: "application/json", MaxResponseSize: 256 << 10,
			Retry: roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
		})
		if err != nil {
			return nil, err
		}
		var result struct {
			Data []struct {
				ID          int64  `json:"id"`
				Name        string `json:"name"`
				DisplayName string `json:"displayName"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Body, &result); err != nil {
			return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "user-names", Status: response.Status, Message: "Roblox returned invalid user details.", Cause: err}
		}
		found := make(map[int64]UserView, len(result.Data))
		for _, user := range result.Data {
			found[user.ID] = UserView{UserID: user.ID, Username: user.Name, DisplayName: user.DisplayName}
		}
		for _, id := range batch {
			user, exists := found[id]
			if !exists {
				user = UserView{UserID: id}
			}
			users = append(users, user)
		}
	}
	return users, nil
}

func (service *Users) writeAccount(ctx context.Context, accountID, version int64, endpoint, method, address string, payload any) (*roblox.Response, error) {
	var body []byte
	contentType := "application/json"
	if values, ok := payload.(url.Values); ok {
		body = []byte(values.Encode())
		contentType = "application/x-www-form-urlencoded"
	} else if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "The request could not be encoded.", Cause: err}
		}
	}
	uri, _ := url.Parse(address)
	return service.client.Do(ctx, roblox.Request{
		Endpoint: endpoint, AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: method, URL: uri, Body: body,
		ContentType: contentType, RequiresCSRF: true, MaxResponseSize: 64 << 10,
	})
}
