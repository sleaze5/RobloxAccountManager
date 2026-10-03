package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const userSettingsURL = "https://apis.roblox.com/user-settings-api/v1/user-settings"

type AccountSettings struct {
	client *roblox.Client
}

// SettingsDocument retains presence and null separately. Only allowlisted fields
// are decoded by the application workflow; this document never crosses Wails.
type SettingsDocument map[string]json.RawMessage

type SettingOptions struct {
	CurrentValue json.RawMessage `json:"currentValue"`
	Options      []struct {
		Option struct {
			Value json.RawMessage `json:"optionValue"`
		} `json:"option"`
		Requirement     *string         `json:"requirement"`
		RequiredActions json.RawMessage `json:"requiredActions"`
	} `json:"options"`
}

func NewAccountSettings(client *roblox.Client) *AccountSettings {
	return &AccountSettings{client: client}
}

func (service *AccountSettings) Read(ctx context.Context, accountID, version int64) (SettingsDocument, error) {
	return service.read(ctx, accountID, version, "account-settings", userSettingsURL)
}

func (service *AccountSettings) ReadOptions(ctx context.Context, accountID, version int64) (SettingsDocument, error) {
	return service.read(ctx, accountID, version, "account-settings-options", userSettingsURL+"/settings-and-options")
}

func (service *AccountSettings) ReadVoice(ctx context.Context, accountID, version int64) (SettingsDocument, error) {
	return service.read(ctx, accountID, version, "account-settings-voice", "https://voice.roblox.com/v1/settings")
}

func (service *AccountSettings) ReadOptionsV2(ctx context.Context, accountID, version int64) (SettingsDocument, error) {
	return service.read(ctx, accountID, version, "account-settings-options-v2", "https://apis.roblox.com/user-settings-api/v2/user-settings/settings-and-options-subset?requestedUserSettings=whoCanChatWithMeInExperiences,whoCanWhisperChatWithMeInExperiences,whoCanPartyWithMe,whoCanUsePartyChatWithMe,whoCanUsePartyVoiceWithMe,allowPresetChat")
}

func (service *AccountSettings) read(ctx context.Context, accountID, version int64, endpoint, address string) (SettingsDocument, error) {
	uri, _ := url.Parse(address)
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: endpoint, AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodGet, URL: uri,
		MaxResponseSize: 256 << 10,
		Retry:           roblox.RetryPolicy{MaxAttempts: 3, RetryServerErrors: true, RetryRateLimit: true},
	})
	if err != nil {
		return nil, err
	}
	var document SettingsDocument
	if err := json.Unmarshal(response.Body, &document); err != nil || document == nil {
		return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Status: response.Status, Message: "Roblox returned invalid account settings."}
	}
	return document, nil
}

// Verified against Roblox's signed-in UserSettings editor on 2026-09-09:
// https://js.rbxcdn.com/3ac8cea8a8e2d8962708ae4664947dbaa7350737d3b6bab03f88c294e057a9df-UserSettings.js
// updateUserSettingValue posts one string field to v1, including maturity.
// Experience chat and quick words require v2 option requiredActions checks,
// but still use this v1 POST. No response fields are used; require GET read-back.
func (service *AccountSettings) Update(ctx context.Context, accountID, version int64, key, value string) error {
	switch key {
	case "contentAgeRestriction", "allowSensitiveIssues", "whoCanSeeMyInventory",
		"whoCanChatWithMeInExperiences",
		"whoCanWhisperChatWithMeInExperiences",
		"whoCanSeeMyOnlineStatus", "whoCanJoinMeInExperiences", "privateServerPrivacy",
		"updateFriendsAboutMyActivity", "whoCanTradeWithMe", "tradeQualityFilter", "allowPresetChat",
		"whoCanPartyWithMe", "whoCanUsePartyChatWithMe", "whoCanUsePartyVoiceWithMe",
		"allowVoiceDataUsage", "allowPersonalizedAdvertising", "allowSellShareData",
		"phoneNumberDiscoverability", "friendSuggestions", "whoCanSeeMySocialNetworks":
	default:
		return &roblox.Error{Kind: roblox.KindForbidden, Endpoint: "account-settings-update", Message: "This setting cannot be changed in this app yet."}
	}
	body, err := json.Marshal(map[string]string{key: value})
	if err != nil {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "account-settings-update", Message: "The setting could not be encoded."}
	}
	return service.update(ctx, accountID, version, userSettingsURL, body)
}

func (service *AccountSettings) UpdateBool(ctx context.Context, accountID, version int64, key string, value bool) error {
	address, payloadKey := userSettingsURL, key
	switch key {
	case "isUserOptIn":
		address = "https://voice.roblox.com/v1/settings/user-opt-in"
	case "isAvatarVideoOptIn":
		address, payloadKey = "https://voice.roblox.com/v1/settings/user-opt-in/avatarvideo", "isUserOptIn"
	case "canUploadContacts":
		address, payloadKey = userSettingsURL, "canUploadContacts"
	default:
		return &roblox.Error{Kind: roblox.KindForbidden, Endpoint: "account-settings-update", Message: "This boolean setting is not supported."}
	}
	body, err := json.Marshal(map[string]bool{payloadKey: value})
	if err != nil {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "account-settings-update", Message: "The setting could not be encoded."}
	}
	return service.update(ctx, accountID, version, address, body)
}

func (service *AccountSettings) update(ctx context.Context, accountID, version int64, address string, body []byte) error {
	uri, _ := url.Parse(address)
	_, err := service.client.Do(ctx, roblox.Request{
		Endpoint: "account-settings-update", AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodPost, URL: uri, Body: body,
		ContentType: "application/json", RequiresCSRF: true, MaxResponseSize: 64 << 10,
	})
	return err
}
