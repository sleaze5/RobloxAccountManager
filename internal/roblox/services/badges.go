package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	universeBadgesPageSize = 100
	// MaxUniverseBadges bounds the badges read for one universe, so that a
	// universe with thousands of badges cannot make one request run for minutes.
	MaxUniverseBadges = 1000
	// Roblox accepts at most 100 badge IDs in one awarded dates request.
	badgeAwardBatchSize       = 100
	maxBadgeNameLength        = 100
	maxBadgeDescriptionLength = 1000
)

type Badges struct {
	client *roblox.Client
}

func NewBadges(client *roblox.Client) *Badges {
	return &Badges{client: client}
}

// ForUniverse lists the badges of the universe in the order the creator ranks
// them, without icons. It reads at most MaxUniverseBadges badges.
func (service *Badges) ForUniverse(ctx context.Context, universeID int64) ([]games.Badge, error) {
	badges := []games.Badge{}
	cursor := ""
	for range MaxUniverseBadges / universeBadgesPageSize {
		parameters := url.Values{"limit": {strconv.Itoa(universeBadgesPageSize)}, "sortBy": {"Rank"}, "sortOrder": {"Asc"}}
		if cursor != "" {
			parameters.Set("cursor", cursor)
		}
		var payload struct {
			NextPageCursor string `json:"nextPageCursor"`
			Data           []struct {
				ID                 int64  `json:"id"`
				DisplayName        string `json:"displayName"`
				DisplayDescription string `json:"displayDescription"`
				Enabled            bool   `json:"enabled"`
				Created            string `json:"created"`
				Statistics         struct {
					PastDayAwardedCount int64   `json:"pastDayAwardedCount"`
					AwardedCount        int64   `json:"awardedCount"`
					WinRatePercentage   float64 `json:"winRatePercentage"`
				} `json:"statistics"`
			} `json:"data"`
		}
		address := &url.URL{Scheme: "https", Host: "badges.roblox.com", Path: "/v1/universes/" + strconv.FormatInt(universeID, 10) + "/badges", RawQuery: parameters.Encode()}
		response, err := service.client.Do(ctx, roblox.Request{
			Endpoint: "game-badges", Method: http.MethodGet, URL: address, MaxResponseSize: 2 << 20,
			Retry: roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
		})
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(response.Body, &payload); err != nil {
			return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "game-badges", Status: response.Status, Message: "Roblox returned invalid badge data.", Cause: err}
		}
		if len(payload.Data) > universeBadgesPageSize || len(payload.NextPageCursor) > maxServerCursorLength {
			return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "game-badges", Status: response.Status, Message: "Roblox returned invalid badge data."}
		}
		for _, listedBadge := range payload.Data {
			if !games.ValidID(listedBadge.ID) {
				continue
			}
			badges = append(badges, games.Badge{
				BadgeID: listedBadge.ID, Name: boundedText(listedBadge.DisplayName, maxBadgeNameLength),
				Description: boundedText(listedBadge.DisplayDescription, maxBadgeDescriptionLength), Enabled: listedBadge.Enabled,
				AwardedCount: max(listedBadge.Statistics.AwardedCount, 0), AwardedPastDay: max(listedBadge.Statistics.PastDayAwardedCount, 0),
				PastDayWinFraction: min(max(listedBadge.Statistics.WinRatePercentage, 0), 1), CreatedAtMs: robloxTimeMs(listedBadge.Created),
			})
		}
		if payload.NextPageCursor == "" {
			break
		}
		cursor = payload.NextPageCursor
	}
	return badges, nil
}

// Awards reads when the Roblox user earned each of the badges. Badges the user
// has not earned are left out.
func (service *Badges) Awards(ctx context.Context, accountID, version, robloxUserID int64, badgeIDs []int64) ([]games.BadgeAward, error) {
	awards := []games.BadgeAward{}
	requested := make(map[int64]bool, len(badgeIDs))
	for _, badgeID := range badgeIDs {
		requested[badgeID] = true
	}
	path := "/v1/users/" + strconv.FormatInt(robloxUserID, 10) + "/badges/awarded-dates"
	for batch := range slices.Chunk(badgeIDs, badgeAwardBatchSize) {
		formattedIDs := make([]string, len(batch))
		for index, badgeID := range batch {
			formattedIDs[index] = strconv.FormatInt(badgeID, 10)
		}
		address := &url.URL{Scheme: "https", Host: "badges.roblox.com", Path: path, RawQuery: url.Values{"badgeIds": {strings.Join(formattedIDs, ",")}}.Encode()}
		response, err := service.client.Do(ctx, roblox.Request{
			Endpoint: "badge-awards", AccountID: accountID, ExpectedSecretVersion: version,
			Authenticated: true, Method: http.MethodGet, URL: address, MaxResponseSize: 256 << 10,
			Retry: roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
		})
		if err != nil {
			return nil, err
		}
		var payload struct {
			Data []struct {
				BadgeID     int64  `json:"badgeId"`
				AwardedDate string `json:"awardedDate"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Body, &payload); err != nil {
			return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "badge-awards", Status: response.Status, Message: "Roblox returned invalid badge data.", Cause: err}
		}
		for _, listedAward := range payload.Data {
			if requested[listedAward.BadgeID] {
				awards = append(awards, games.BadgeAward{BadgeID: listedAward.BadgeID, AwardedAtMs: robloxTimeMs(listedAward.AwardedDate)})
			}
		}
	}
	return awards, nil
}

func boundedText(rawText string, maxLength int) string {
	rawText = strings.TrimSpace(rawText)
	if !utf8.ValidString(rawText) || utf8.RuneCountInString(rawText) > maxLength {
		return ""
	}
	return rawText
}
