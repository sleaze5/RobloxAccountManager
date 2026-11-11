package services

import (
	"context"
	"encoding/json"
	"errors"
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
	collectiblePageSize     = 100
	transactionPageSize     = 25
	maxCollectiblePages     = 50
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

type UserTransactionTotals struct {
	PendingRobux  int64
	IncomingRobux int64
	OutgoingRobux int64
}

// TransactionTotals reads the Robux totals of the past month. Roblox reports
// outgoing Robux as a negative number. Shape verified against a signed-in
// summary response captured on 2026-10-15.
func (service *Users) TransactionTotals(ctx context.Context, accountID, version, userID int64) (UserTransactionTotals, error) {
	var payload struct {
		PendingRobux  *int64 `json:"pendingRobuxTotal"`
		IncomingRobux *int64 `json:"incomingRobuxTotal"`
		OutgoingRobux *int64 `json:"outgoingRobuxTotal"`
	}
	address := "https://apis.roblox.com/transaction-records/v1/users/" + strconv.FormatInt(userID, 10) + "/transaction-totals?timeFrame=Month&transactionType=summary"
	if err := service.readAccountDetails(ctx, accountID, version, "profile-transaction-totals", address, &payload); err != nil {
		return UserTransactionTotals{}, err
	}
	if payload.PendingRobux == nil || *payload.PendingRobux < 0 || payload.IncomingRobux == nil || *payload.IncomingRobux < 0 || payload.OutgoingRobux == nil || *payload.OutgoingRobux > 0 {
		return UserTransactionTotals{}, invalidProfileDetails("profile-transaction-totals")
	}
	return UserTransactionTotals{PendingRobux: *payload.PendingRobux, IncomingRobux: *payload.IncomingRobux, OutgoingRobux: -*payload.OutgoingRobux}, nil
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

// CommunityCount counts the communities a user belongs to. The v2 list leaves
// out descriptions and shouts, so it stays small for users in many communities.
func (service *Users) CommunityCount(ctx context.Context, accountID, version, userID int64) (int64, error) {
	var payload struct {
		Data *[]json.RawMessage `json:"data"`
	}
	address := "https://groups.roblox.com/v2/users/" + strconv.FormatInt(userID, 10) + "/groups/roles"
	if err := service.readAccountDetails(ctx, accountID, version, "profile-communities", address, &payload); err != nil {
		return 0, err
	}
	if payload.Data == nil {
		return 0, invalidProfileDetails("profile-communities")
	}
	return int64(len(*payload.Data)), nil
}

// Shapes of the unread and inbound counts verified against signed-in
// responses captured on 2026-10-15.

func (service *Users) UnreadMessageCount(ctx context.Context, accountID, version int64) (int64, error) {
	return service.readCount(ctx, accountID, version, "profile-unread-messages", "https://privatemessages.roblox.com/v1/messages/unread/count")
}

// InboundTradeCount names the trade status, because Roblox reads a number in
// its place as a trade ID.
func (service *Users) InboundTradeCount(ctx context.Context, accountID, version int64) (int64, error) {
	return service.readCount(ctx, accountID, version, "profile-inbound-trades", "https://trades.roblox.com/v1/trades/Inbound/count")
}

func (service *Users) UnreadNotificationCount(ctx context.Context, accountID, version int64) (int64, error) {
	var payload struct {
		UnreadNotifications *int64 `json:"unreadNotifications"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "profile-unread-notifications", "https://notifications.roblox.com/v2/stream-notifications/unread-count", &payload); err != nil {
		return 0, err
	}
	if payload.UnreadNotifications == nil || *payload.UnreadNotifications < 0 {
		return 0, invalidProfileDetails("profile-unread-notifications")
	}
	return *payload.UnreadNotifications, nil
}

func (service *Users) readCount(ctx context.Context, accountID, version int64, endpoint, address string) (int64, error) {
	var payload struct {
		Count *int64 `json:"count"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, endpoint, address, &payload); err != nil {
		return 0, err
	}
	if payload.Count == nil || *payload.Count < 0 {
		return 0, invalidProfileDetails(endpoint)
	}
	return *payload.Count, nil
}

type UserPremiumSubscription struct {
	Name         string
	RobuxStipend int64
	RenewsAt     time.Time
	ExpiresAt    time.Time
}

// PremiumSubscription returns nil for an account without Premium, because
// Roblox answers 404 then. The shape follows the Roblox billing page on
// 2026-10-16. A response without a renewal or expiration date is still valid.
func (service *Users) PremiumSubscription(ctx context.Context, accountID, version, userID int64) (*UserPremiumSubscription, error) {
	var payload struct {
		Product *struct {
			Name         string `json:"subscriptionName"`
			RobuxStipend int64  `json:"robuxStipendAmount"`
			Renewal      string `json:"renewal"`
			Expiration   string `json:"expiration"`
		} `json:"subscriptionProductModel"`
	}
	address := "https://premiumfeatures.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/subscriptions/details"
	err := service.readAccountDetails(ctx, accountID, version, "profile-premium-subscription", address, &payload)
	if remote, ok := errors.AsType[*roblox.Error](err); ok && remote.Status == http.StatusNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if payload.Product == nil || len(payload.Product.Name) > 128 || payload.Product.RobuxStipend < 0 {
		return nil, invalidProfileDetails("profile-premium-subscription")
	}
	subscription := &UserPremiumSubscription{Name: strings.TrimSpace(payload.Product.Name), RobuxStipend: payload.Product.RobuxStipend}
	if renewsAt, err := time.Parse(time.RFC3339, strings.TrimSpace(payload.Product.Renewal)); err == nil {
		subscription.RenewsAt = renewsAt.UTC()
	}
	if expiresAt, err := time.Parse(time.RFC3339, strings.TrimSpace(payload.Product.Expiration)); err == nil {
		subscription.ExpiresAt = expiresAt.UTC()
	}
	return subscription, nil
}

type UserTradeEligibility struct {
	CanTrade bool
	// Eligibility is the Roblox status name, such as Eligible or
	// IneligibleMissingPremiumMembership.
	Eligibility string
	// FreeTrades is nil when Roblox sends no free trade allowance.
	FreeTrades *UserFreeTrades
}

type UserFreeTrades struct {
	Remaining int64  `json:"remaining"`
	Limit     int64  `json:"limit"`
	Window    string `json:"window"`
}

// TradeEligibility follows the Roblox trades v2 API schema.
func (service *Users) TradeEligibility(ctx context.Context, accountID, version int64) (UserTradeEligibility, error) {
	var payload struct {
		CanTrade    *bool  `json:"canTrade"`
		Eligibility string `json:"tradeEligibility"`
		Allowance   *struct {
			Limit     *int64 `json:"limit"`
			Remaining *int64 `json:"remaining"`
			Window    string `json:"window"`
		} `json:"freeTradesAllowance"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "profile-trade-eligibility", "https://trades.roblox.com/v2/users/me/can-trade", &payload); err != nil {
		return UserTradeEligibility{}, err
	}
	if payload.CanTrade == nil || len(payload.Eligibility) > 64 {
		return UserTradeEligibility{}, invalidProfileDetails("profile-trade-eligibility")
	}
	eligibility := UserTradeEligibility{CanTrade: *payload.CanTrade, Eligibility: payload.Eligibility}
	if allowance := payload.Allowance; allowance != nil && allowance.Limit != nil && allowance.Remaining != nil && *allowance.Limit >= 0 && *allowance.Remaining >= 0 && len(allowance.Window) <= 16 {
		eligibility.FreeTrades = &UserFreeTrades{Remaining: *allowance.Remaining, Limit: *allowance.Limit, Window: strings.ToLower(allowance.Window)}
	}
	return eligibility, nil
}

// CanTradeWith follows the Roblox trades v2 API schema. The eligibility is the
// Roblox status name, such as Eligible or TargetUserIneligible.
func (service *Users) CanTradeWith(ctx context.Context, accountID, version, userID int64) (UserTradeEligibility, error) {
	var payload struct {
		CanTrade    *bool  `json:"canTrade"`
		Eligibility string `json:"mutualTradeEligibility"`
	}
	address := "https://trades.roblox.com/v2/users/" + strconv.FormatInt(userID, 10) + "/can-trade-with"
	if err := service.readAccountDetails(ctx, accountID, version, "friend-trade-eligibility", address, &payload); err != nil {
		return UserTradeEligibility{}, err
	}
	if payload.CanTrade == nil || len(payload.Eligibility) > 64 {
		return UserTradeEligibility{}, invalidProfileDetails("friend-trade-eligibility")
	}
	return UserTradeEligibility{CanTrade: *payload.CanTrade, Eligibility: payload.Eligibility}, nil
}

type UserTransaction struct {
	IDHash     string
	CreatedAt  time.Time
	Amount     int64
	IsPending  bool
	ItemName   string
	ItemType   string
	AgentName  string
	AgentType  string
	PlaceName  string
	PlaceID    int64
	UniverseID int64
}

// Transactions reads one page of Robux transactions of one Roblox type, such
// as Purchase or Sale, newest first. Shape verified against signed-in
// responses captured on 2026-10-15.
func (service *Users) Transactions(ctx context.Context, accountID, version, userID int64, transactionType, cursor string) ([]UserTransaction, string, error) {
	var payload struct {
		Data *[]struct {
			IDHash    string `json:"idHash"`
			Created   string `json:"created"`
			IsPending bool   `json:"isPending"`
			Agent     *struct {
				Name string `json:"name"`
			} `json:"agent"`
			Currency *struct {
				Amount *int64 `json:"amount"`
			} `json:"currency"`
			Details *struct {
				Name  string `json:"name"`
				Type  string `json:"type"`
				Place *struct {
					Name string `json:"name"`
				} `json:"place"`
			} `json:"details"`
		} `json:"data"`
		NextPageCursor *string `json:"nextPageCursor"`
	}
	query := url.Values{"limit": {strconv.Itoa(transactionPageSize)}, "transactionType": {transactionType}}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	address := "https://apis.roblox.com/transaction-records/v1/users/" + strconv.FormatInt(userID, 10) + "/transactions?" + query.Encode()
	if err := service.readAccountDetails(ctx, accountID, version, "robux-transactions", address, &payload); err != nil {
		return nil, "", err
	}
	if payload.Data == nil || len(*payload.Data) > transactionPageSize {
		return nil, "", invalidProfileDetails("robux-transactions")
	}
	transactions := make([]UserTransaction, 0, len(*payload.Data))
	for _, listedTransaction := range *payload.Data {
		if listedTransaction.Currency == nil || listedTransaction.Currency.Amount == nil {
			return nil, "", invalidProfileDetails("robux-transactions")
		}
		transaction := UserTransaction{IDHash: listedTransaction.IDHash, Amount: *listedTransaction.Currency.Amount, IsPending: listedTransaction.IsPending}
		if createdAt, err := time.Parse(time.RFC3339, strings.TrimSpace(listedTransaction.Created)); err == nil {
			transaction.CreatedAt = createdAt.UTC()
		}
		if agent := listedTransaction.Agent; agent != nil {
			transaction.AgentName = strings.TrimSpace(agent.Name)
		}
		if details := listedTransaction.Details; details != nil {
			transaction.ItemName, transaction.ItemType = strings.TrimSpace(details.Name), details.Type
			if place := details.Place; place != nil {
				transaction.PlaceName = strings.TrimSpace(place.Name)
			}
		}
		transactions = append(transactions, transaction)
	}
	nextCursor := ""
	if payload.NextPageCursor != nil {
		nextCursor = *payload.NextPageCursor
	}
	if len(nextCursor) > 4096 {
		return nil, "", invalidProfileDetails("robux-transactions")
	}
	return transactions, nextCursor, nil
}

type UserCollectibles struct {
	Count                   int64
	RecentAveragePriceTotal int64
}

// Collectibles counts a user's limited items and adds up their recent average
// prices. Shape verified against a collectibles response captured on 2026-10-15.
func (service *Users) Collectibles(ctx context.Context, accountID, version, userID int64) (UserCollectibles, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	collectibles := UserCollectibles{}
	seenCursors := map[string]bool{}
	cursor := ""
	for range maxCollectiblePages {
		var payload struct {
			Data *[]struct {
				RecentAveragePrice *int64 `json:"recentAveragePrice"`
			} `json:"data"`
			NextPageCursor *string `json:"nextPageCursor"`
		}
		query := url.Values{"limit": {strconv.Itoa(collectiblePageSize)}, "sortOrder": {"Desc"}}
		if cursor != "" {
			query.Set("cursor", cursor)
		}
		address := "https://inventory.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/assets/collectibles?" + query.Encode()
		if err := service.readAccountDetails(ctx, accountID, version, "profile-collectibles", address, &payload); err != nil {
			return UserCollectibles{}, err
		}
		if payload.Data == nil || len(*payload.Data) > collectiblePageSize {
			return UserCollectibles{}, invalidProfileDetails("profile-collectibles")
		}
		for _, listedCollectible := range *payload.Data {
			collectibles.Count++
			if listedCollectible.RecentAveragePrice != nil && *listedCollectible.RecentAveragePrice > 0 {
				collectibles.RecentAveragePriceTotal += *listedCollectible.RecentAveragePrice
			}
		}
		if payload.NextPageCursor == nil || *payload.NextPageCursor == "" {
			return collectibles, nil
		}
		cursor = *payload.NextPageCursor
		if len(cursor) > 4096 || seenCursors[cursor] {
			return UserCollectibles{}, invalidProfileDetails("profile-collectibles")
		}
		seenCursors[cursor] = true
	}
	return UserCollectibles{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "profile-collectibles", Message: "This account has too many collectibles to count."}
}

func (service *Users) readAccountDetails(ctx context.Context, accountID, version int64, endpoint, address string, target any) error {
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
	if err := json.Unmarshal(response.Body, target); err != nil {
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
