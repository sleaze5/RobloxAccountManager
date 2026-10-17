package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	usernameHistoryPageSize = 100
	maxUsernameHistoryPages = 20
)

type UserAccountInfo struct {
	ID                    int64  `json:"id"`
	AgeBracket            *int   `json:"ageBracket"`
	CountryCode           string `json:"countryCode"`
	IsPremium             *bool  `json:"isPremium"`
	HasRobloxSubscription *bool  `json:"hasRobloxSubscription"`
}

type UserVerificationMethod struct {
	MediaType string `json:"mediaType"`
	Enabled   *bool  `json:"enabled"`
}

func (service *Users) AccountInfo(ctx context.Context, accountID, version, userID int64) (UserAccountInfo, error) {
	var result UserAccountInfo
	err := service.readAccountDetails(ctx, accountID, version, "profile-account", "https://users.roblox.com/v1/users/authenticated/app-launch-info", &result)
	if err == nil && result.ID <= 0 {
		err = invalidProfileDetails("profile-account")
	}
	if err == nil && result.ID != userID {
		err = profileIdentityError()
	}
	return result, err
}

func (service *Users) Robux(ctx context.Context, accountID, version int64) (*int64, error) {
	var result struct {
		Robux *int64 `json:"robux"`
	}
	err := service.readAccountDetails(ctx, accountID, version, "profile-robux", "https://economy.roblox.com/v1/user/currency", &result)
	if err == nil && (result.Robux == nil || *result.Robux < 0) {
		err = invalidProfileDetails("profile-robux")
	}
	return result.Robux, err
}

func (service *Users) PendingRobux(ctx context.Context, accountID, version, userID int64) (*int64, error) {
	var result struct {
		PendingRobux *int64 `json:"pendingRobuxTotal"`
	}
	address := "https://apis.roblox.com/transaction-records/v1/users/" + strconv.FormatInt(userID, 10) + "/transaction-totals?timeFrame=Month&transactionType=pendingRobux"
	err := service.readAccountDetails(ctx, accountID, version, "profile-pending-robux", address, &result)
	if err == nil && (result.PendingRobux == nil || *result.PendingRobux < 0) {
		err = invalidProfileDetails("profile-pending-robux")
	}
	return result.PendingRobux, err
}

func (service *Users) AgeVerified(ctx context.Context, accountID, version int64) (*bool, error) {
	var result struct {
		IsVerified *bool `json:"isVerified"`
	}
	err := service.readAccountDetails(ctx, accountID, version, "profile-age-verification", "https://apis.roblox.com/age-verification-service/v1/age-verification/verified-age", &result)
	if err == nil && result.IsVerified == nil {
		err = invalidProfileDetails("profile-age-verification")
	}
	return result.IsVerified, err
}

func (service *Users) TwoStepVerification(ctx context.Context, accountID, version, userID int64) ([]UserVerificationMethod, error) {
	var result struct {
		Methods *[]UserVerificationMethod `json:"methods"`
	}
	address := "https://twostepverification.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/configuration"
	if err := service.readAccountDetails(ctx, accountID, version, "profile-two-step", address, &result); err != nil {
		return nil, err
	}
	if result.Methods == nil {
		return nil, invalidProfileDetails("profile-two-step")
	}
	for _, method := range *result.Methods {
		if method.Enabled == nil || method.MediaType == "" || len(method.MediaType) > 64 {
			return nil, invalidProfileDetails("profile-two-step")
		}
	}
	return *result.Methods, nil
}

type UserVoiceChat struct {
	OptedIn, Verified, CanVerifyAge, OptInDisabled, Suspended bool
	SuspendedUntil                                            time.Time
}

// Shape verified against signed-in responses on 2026-10-14. bannedUntil matches RoSeal and Altman.
func (service *Users) VoiceChat(ctx context.Context, accountID, version int64) (UserVoiceChat, error) {
	var result struct {
		OptedIn       *bool `json:"isUserOptIn"`
		Verified      bool  `json:"isVerifiedForVoice"`
		CanVerifyAge  bool  `json:"canVerifyAgeForVoice"`
		OptInDisabled bool  `json:"isOptInDisabled"`
		Banned        *bool `json:"isBanned"`
		BannedUntil   *struct {
			Seconds int64 `json:"Seconds"`
		} `json:"bannedUntil"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "profile-voice-chat", "https://voice.roblox.com/v1/settings", &result); err != nil {
		return UserVoiceChat{}, err
	}
	if result.OptedIn == nil || result.Banned == nil {
		return UserVoiceChat{}, invalidProfileDetails("profile-voice-chat")
	}
	voice := UserVoiceChat{
		OptedIn: *result.OptedIn, Verified: result.Verified, CanVerifyAge: result.CanVerifyAge,
		OptInDisabled: result.OptInDisabled, Suspended: *result.Banned,
	}
	if voice.Suspended && result.BannedUntil != nil && result.BannedUntil.Seconds > 0 {
		voice.SuspendedUntil = time.Unix(result.BannedUntil.Seconds, 0)
	}
	return voice, nil
}

type UserAbout struct {
	Description   string
	VerifiedBadge bool
}

type UserSocialLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (service *Users) SocialLinks(ctx context.Context, accountID, version int64) ([]UserSocialLink, error) {
	var result *struct {
		Facebook string `json:"facebook"`
		Twitter  string `json:"twitter"`
		YouTube  string `json:"youtube"`
		Twitch   string `json:"twitch"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "profile-social-links", "https://accountinformation.roblox.com/v1/promotion-channels?alwaysReturnUrls=true", &result); err != nil {
		return nil, err
	}
	if result == nil {
		return nil, invalidProfileDetails("profile-social-links")
	}
	channels := []struct {
		name, address string
		hosts         []string
	}{
		{"Facebook", result.Facebook, []string{"facebook.com"}},
		{"X", result.Twitter, []string{"twitter.com", "x.com"}},
		{"YouTube", result.YouTube, []string{"youtube.com", "youtu.be"}},
		{"Twitch", result.Twitch, []string{"twitch.tv"}},
	}
	links := make([]UserSocialLink, 0, len(channels))
	for _, channel := range channels {
		address := strings.TrimSpace(channel.address)
		if address == "" {
			continue
		}
		uri, err := url.Parse(address)
		if err != nil || len(address) > 2048 || uri.Scheme != "https" || uri.User != nil || (uri.Port() != "" && uri.Port() != "443") {
			return nil, invalidProfileDetails("profile-social-links")
		}
		host := strings.ToLower(uri.Hostname())
		allowed := false
		for _, domain := range channel.hosts {
			if host == domain || strings.HasSuffix(host, "."+domain) {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, invalidProfileDetails("profile-social-links")
		}
		links = append(links, UserSocialLink{Name: channel.name, URL: uri.String()})
	}
	return links, nil
}

func (service *Users) PreviousNames(ctx context.Context, accountID, version, userID int64) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	names := []string{}
	seenNames, seenCursors := map[string]bool{}, map[string]bool{}
	cursor := ""
	for range maxUsernameHistoryPages {
		var result struct {
			Data *[]struct {
				Name string `json:"name"`
			} `json:"data"`
			NextPageCursor *string `json:"nextPageCursor"`
		}
		query := url.Values{"limit": {strconv.Itoa(usernameHistoryPageSize)}, "sortOrder": {"Desc"}}
		if cursor != "" {
			query.Set("cursor", cursor)
		}
		address := "https://users.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/username-history?" + query.Encode()
		if err := service.readAccountDetails(ctx, accountID, version, "profile-previous-names", address, &result); err != nil {
			return nil, err
		}
		if result.Data == nil || len(*result.Data) > usernameHistoryPageSize {
			return nil, invalidProfileDetails("profile-previous-names")
		}
		for _, entry := range *result.Data {
			name := strings.TrimSpace(entry.Name)
			if name == "" || len(name) > 64 {
				return nil, invalidProfileDetails("profile-previous-names")
			}
			key := strings.ToLower(name)
			if !seenNames[key] {
				seenNames[key] = true
				names = append(names, name)
			}
		}
		if result.NextPageCursor == nil || *result.NextPageCursor == "" {
			return names, nil
		}
		cursor = *result.NextPageCursor
		if len(cursor) > 4096 || seenCursors[cursor] {
			return nil, invalidProfileDetails("profile-previous-names")
		}
		seenCursors[cursor] = true
	}
	return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "profile-previous-names", Message: "Roblox returned too many previous names to list."}
}

type UserPrimaryGroup struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Members  int64  `json:"members"`
	Verified bool   `json:"verified"`
}

func (service *Users) About(ctx context.Context, accountID, version, userID int64) (UserAbout, error) {
	var result struct {
		ID               int64   `json:"id"`
		Description      *string `json:"description"`
		HasVerifiedBadge bool    `json:"hasVerifiedBadge"`
	}
	address := "https://users.roblox.com/v1/users/" + strconv.FormatInt(userID, 10)
	if err := service.readAccountDetails(ctx, accountID, version, "profile-about", address, &result); err != nil {
		return UserAbout{}, err
	}
	if result.ID != userID || result.Description == nil || len(*result.Description) > 4096 {
		return UserAbout{}, invalidProfileDetails("profile-about")
	}
	return UserAbout{Description: strings.TrimSpace(*result.Description), VerifiedBadge: result.HasVerifiedBadge}, nil
}

func (service *Users) SocialCount(ctx context.Context, accountID, version, userID int64, counter string) (*int64, error) {
	switch counter {
	case "friends", "followers", "followings":
	default:
		return nil, invalidProfileDetails("profile-social")
	}
	var result struct {
		Count *int64 `json:"count"`
	}
	endpoint := "profile-" + counter
	address := "https://friends.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/" + counter + "/count"
	err := service.readAccountDetails(ctx, accountID, version, endpoint, address, &result)
	if err == nil && (result.Count == nil || *result.Count < 0) {
		err = invalidProfileDetails(endpoint)
	}
	return result.Count, err
}

func (service *Users) PrimaryGroup(ctx context.Context, accountID, version, userID int64) (*UserPrimaryGroup, error) {
	var result *struct {
		Group *struct {
			ID               int64  `json:"id"`
			Name             string `json:"name"`
			MemberCount      int64  `json:"memberCount"`
			HasVerifiedBadge bool   `json:"hasVerifiedBadge"`
		} `json:"group"`
		Role *struct {
			Name string `json:"name"`
		} `json:"role"`
	}
	address := "https://groups.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/groups/primary/role"
	if err := service.readAccountDetails(ctx, accountID, version, "profile-primary-group", address, &result); err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	if result.Group == nil || result.Group.ID <= 0 || strings.TrimSpace(result.Group.Name) == "" || len(result.Group.Name) > 256 || result.Role == nil || len(result.Role.Name) > 256 {
		return nil, invalidProfileDetails("profile-primary-group")
	}
	return &UserPrimaryGroup{
		ID: result.Group.ID, Name: strings.TrimSpace(result.Group.Name), Role: strings.TrimSpace(result.Role.Name),
		Members: max(result.Group.MemberCount, 0), Verified: result.Group.HasVerifiedBadge,
	}, nil
}

func (service *Users) readAccountDetails(ctx context.Context, accountID, version int64, endpoint, address string, result any) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	uri, _ := url.Parse(address)
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: endpoint, AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodGet, URL: uri,
		ContentType: "application/json", MaxResponseSize: 64 << 10,
		Retry: roblox.RetryPolicy{MaxAttempts: 2, RetryServerErrors: true, RetryRateLimit: true},
	})
	if err != nil {
		return err
	}
	if err := json.Unmarshal(response.Body, result); err != nil {
		return invalidProfileDetails(endpoint)
	}
	return nil
}

func invalidProfileDetails(endpoint string) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Roblox returned invalid profile details."}
}

func profileIdentityError() error {
	return &roblox.Error{Kind: roblox.KindReauthRequired, Endpoint: "account-profile", Message: "The session identity changed. Replace the cookie."}
}
