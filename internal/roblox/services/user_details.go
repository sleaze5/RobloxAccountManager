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

type UserAccountDetails struct {
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

func (service *Users) AccountInfo(ctx context.Context, accountID, version, userID int64) (UserAccountDetails, error) {
	var accountDetails UserAccountDetails
	err := service.readAccountDetails(ctx, accountID, version, "profile-account", "https://users.roblox.com/v1/users/authenticated/app-launch-info", &accountDetails)
	if err == nil && accountDetails.ID <= 0 {
		err = invalidProfileDetails("profile-account")
	}
	if err == nil && accountDetails.ID != userID {
		err = profileIdentityError()
	}
	return accountDetails, err
}

func (service *Users) Robux(ctx context.Context, accountID, version int64) (*int64, error) {
	var payload struct {
		Robux *int64 `json:"robux"`
	}
	err := service.readAccountDetails(ctx, accountID, version, "profile-robux", "https://economy.roblox.com/v1/user/currency", &payload)
	if err == nil && (payload.Robux == nil || *payload.Robux < 0) {
		err = invalidProfileDetails("profile-robux")
	}
	return payload.Robux, err
}

func (service *Users) PendingRobux(ctx context.Context, accountID, version, userID int64) (*int64, error) {
	var payload struct {
		PendingRobux *int64 `json:"pendingRobuxTotal"`
	}
	address := "https://apis.roblox.com/transaction-records/v1/users/" + strconv.FormatInt(userID, 10) + "/transaction-totals?timeFrame=Month&transactionType=pendingRobux"
	err := service.readAccountDetails(ctx, accountID, version, "profile-pending-robux", address, &payload)
	if err == nil && (payload.PendingRobux == nil || *payload.PendingRobux < 0) {
		err = invalidProfileDetails("profile-pending-robux")
	}
	return payload.PendingRobux, err
}

func (service *Users) AgeVerified(ctx context.Context, accountID, version int64) (*bool, error) {
	var payload struct {
		IsVerified *bool `json:"isVerified"`
	}
	err := service.readAccountDetails(ctx, accountID, version, "profile-age-verification", "https://apis.roblox.com/age-verification-service/v1/age-verification/verified-age", &payload)
	if err == nil && payload.IsVerified == nil {
		err = invalidProfileDetails("profile-age-verification")
	}
	return payload.IsVerified, err
}

func (service *Users) TwoStepVerification(ctx context.Context, accountID, version, userID int64) ([]UserVerificationMethod, error) {
	var payload struct {
		Methods *[]UserVerificationMethod `json:"methods"`
	}
	address := "https://twostepverification.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/configuration"
	if err := service.readAccountDetails(ctx, accountID, version, "profile-two-step", address, &payload); err != nil {
		return nil, err
	}
	if payload.Methods == nil {
		return nil, invalidProfileDetails("profile-two-step")
	}
	for _, method := range *payload.Methods {
		if method.Enabled == nil || method.MediaType == "" || len(method.MediaType) > 64 {
			return nil, invalidProfileDetails("profile-two-step")
		}
	}
	return *payload.Methods, nil
}

type UserVoiceChat struct {
	OptedIn, Verified, CanVerifyAge, OptInDisabled, Suspended bool
	SuspendedUntil                                            time.Time
}

// Shape verified against signed-in responses on 2026-10-14. bannedUntil matches RoSeal and Altman.
func (service *Users) VoiceChat(ctx context.Context, accountID, version int64) (UserVoiceChat, error) {
	var payload struct {
		OptedIn       *bool `json:"isUserOptIn"`
		Verified      bool  `json:"isVerifiedForVoice"`
		CanVerifyAge  bool  `json:"canVerifyAgeForVoice"`
		OptInDisabled bool  `json:"isOptInDisabled"`
		Banned        *bool `json:"isBanned"`
		BannedUntil   *struct {
			Seconds int64 `json:"Seconds"`
		} `json:"bannedUntil"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "profile-voice-chat", "https://voice.roblox.com/v1/settings", &payload); err != nil {
		return UserVoiceChat{}, err
	}
	if payload.OptedIn == nil || payload.Banned == nil {
		return UserVoiceChat{}, invalidProfileDetails("profile-voice-chat")
	}
	voice := UserVoiceChat{
		OptedIn: *payload.OptedIn, Verified: payload.Verified, CanVerifyAge: payload.CanVerifyAge,
		OptInDisabled: payload.OptInDisabled, Suspended: *payload.Banned,
	}
	if voice.Suspended && payload.BannedUntil != nil && payload.BannedUntil.Seconds > 0 {
		voice.SuspendedUntil = time.Unix(payload.BannedUntil.Seconds, 0)
	}
	return voice, nil
}

type UserAbout struct {
	Description   string
	VerifiedBadge bool
	Banned        bool
	CreatedAt     time.Time
}

type UserSocialLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (service *Users) SocialLinks(ctx context.Context, accountID, version int64) ([]UserSocialLink, error) {
	return service.socialLinks(ctx, accountID, version, "profile-social-links", "https://accountinformation.roblox.com/v1/promotion-channels?alwaysReturnUrls=true")
}

func (service *Users) UserSocialLinks(ctx context.Context, accountID, version, userID int64) ([]UserSocialLink, error) {
	address := "https://accountinformation.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/promotion-channels?alwaysReturnUrls=true"
	return service.socialLinks(ctx, accountID, version, "friend-social-links", address)
}

func (service *Users) socialLinks(ctx context.Context, accountID, version int64, endpoint, address string) ([]UserSocialLink, error) {
	var payload *struct {
		Facebook string `json:"facebook"`
		Twitter  string `json:"twitter"`
		YouTube  string `json:"youtube"`
		Twitch   string `json:"twitch"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, endpoint, address, &payload); err != nil {
		return nil, err
	}
	if payload == nil {
		return nil, invalidProfileDetails(endpoint)
	}
	channels := []struct {
		name, address string
		hosts         []string
	}{
		{"Facebook", payload.Facebook, []string{"facebook.com"}},
		{"X", payload.Twitter, []string{"twitter.com", "x.com"}},
		{"YouTube", payload.YouTube, []string{"youtube.com", "youtu.be"}},
		{"Twitch", payload.Twitch, []string{"twitch.tv"}},
	}
	links := make([]UserSocialLink, 0, len(channels))
	for _, channel := range channels {
		address := strings.TrimSpace(channel.address)
		if address == "" {
			continue
		}
		endpointURL, err := url.Parse(address)
		if err != nil || len(address) > 2048 || endpointURL.Scheme != "https" || endpointURL.User != nil || (endpointURL.Port() != "" && endpointURL.Port() != "443") {
			return nil, invalidProfileDetails(endpoint)
		}
		host := strings.ToLower(endpointURL.Hostname())
		allowed := false
		for _, domain := range channel.hosts {
			if host == domain || strings.HasSuffix(host, "."+domain) {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, invalidProfileDetails(endpoint)
		}
		links = append(links, UserSocialLink{Name: channel.name, URL: endpointURL.String()})
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
		var payload struct {
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
		if err := service.readAccountDetails(ctx, accountID, version, "profile-previous-names", address, &payload); err != nil {
			return nil, err
		}
		if payload.Data == nil || len(*payload.Data) > usernameHistoryPageSize {
			return nil, invalidProfileDetails("profile-previous-names")
		}
		for _, entry := range *payload.Data {
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
		if payload.NextPageCursor == nil || *payload.NextPageCursor == "" {
			return names, nil
		}
		cursor = *payload.NextPageCursor
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
	var payload struct {
		ID               int64   `json:"id"`
		Description      *string `json:"description"`
		HasVerifiedBadge bool    `json:"hasVerifiedBadge"`
		IsBanned         bool    `json:"isBanned"`
		Created          string  `json:"created"`
	}
	address := "https://users.roblox.com/v1/users/" + strconv.FormatInt(userID, 10)
	if err := service.readAccountDetails(ctx, accountID, version, "profile-about", address, &payload); err != nil {
		return UserAbout{}, err
	}
	if payload.ID != userID || payload.Description == nil || len(*payload.Description) > 4096 {
		return UserAbout{}, invalidProfileDetails("profile-about")
	}
	about := UserAbout{Description: strings.TrimSpace(*payload.Description), VerifiedBadge: payload.HasVerifiedBadge, Banned: payload.IsBanned}
	if createdAt, err := time.Parse(time.RFC3339, strings.TrimSpace(payload.Created)); err == nil {
		about.CreatedAt = createdAt.UTC()
	}
	return about, nil
}

func (service *Users) SocialCount(ctx context.Context, accountID, version, userID int64, counter string) (*int64, error) {
	switch counter {
	case "friends", "followers", "followings":
	default:
		return nil, invalidProfileDetails("profile-social")
	}
	var payload struct {
		Count *int64 `json:"count"`
	}
	endpoint := "profile-" + counter
	address := "https://friends.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/" + counter + "/count"
	err := service.readAccountDetails(ctx, accountID, version, endpoint, address, &payload)
	if err == nil && (payload.Count == nil || *payload.Count < 0) {
		err = invalidProfileDetails(endpoint)
	}
	return payload.Count, err
}

func (service *Users) PrimaryGroup(ctx context.Context, accountID, version, userID int64) (*UserPrimaryGroup, error) {
	var payload *struct {
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
	if err := service.readAccountDetails(ctx, accountID, version, "profile-primary-group", address, &payload); err != nil {
		return nil, err
	}
	if payload == nil {
		return nil, nil
	}
	if payload.Group == nil || payload.Group.ID <= 0 || strings.TrimSpace(payload.Group.Name) == "" || len(payload.Group.Name) > 256 || payload.Role == nil || len(payload.Role.Name) > 256 {
		return nil, invalidProfileDetails("profile-primary-group")
	}
	return &UserPrimaryGroup{
		ID: payload.Group.ID, Name: strings.TrimSpace(payload.Group.Name), Role: strings.TrimSpace(payload.Role.Name),
		Members: max(payload.Group.MemberCount, 0), Verified: payload.Group.HasVerifiedBadge,
	}, nil
}

func (service *Users) readAccountDetails(ctx context.Context, accountID, version int64, endpoint, address string, result any) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	endpointURL, _ := url.Parse(address)
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: endpoint, AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodGet, URL: endpointURL,
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
