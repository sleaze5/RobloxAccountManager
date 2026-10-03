package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	headshotBatchSize = 100
	maxHeadshotIDs    = 1000
)

type AvatarHeadshotView struct {
	RobloxUserID int64  `json:"robloxUserId"`
	ImageURL     string `json:"imageUrl"`
}

type Thumbnails struct {
	client   *roblox.Client
	endpoint *url.URL
}

func NewThumbnails(client *roblox.Client) *Thumbnails {
	endpoint, _ := url.Parse("https://thumbnails.roblox.com/v1/users/avatar-headshot")
	return &Thumbnails{client: client, endpoint: endpoint}
}

func (service *Thumbnails) AvatarHeadshots(ctx context.Context, userIDs []int64) ([]AvatarHeadshotView, error) {
	if len(userIDs) > maxHeadshotIDs {
		return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "avatar-headshots", Message: "Too many avatar headshots were requested."}
	}
	unique := make([]int64, 0, len(userIDs))
	seen := make(map[int64]struct{}, len(userIDs))
	for _, userID := range userIDs {
		if userID <= 0 {
			return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "avatar-headshots", Message: "An invalid Roblox user ID was requested."}
		}
		if _, exists := seen[userID]; exists {
			continue
		}
		seen[userID] = struct{}{}
		unique = append(unique, userID)
	}
	result := make([]AvatarHeadshotView, 0, len(unique))
	for start := 0; start < len(unique); start += headshotBatchSize {
		end := min(start+headshotBatchSize, len(unique))
		requestURL := *service.endpoint
		query := requestURL.Query()
		values := make([]string, 0, end-start)
		for _, userID := range unique[start:end] {
			values = append(values, strconv.FormatInt(userID, 10))
		}
		query.Set("userIds", strings.Join(values, ","))
		query.Set("size", "720x720")
		query.Set("format", "Png")
		query.Set("isCircular", "false")
		requestURL.RawQuery = query.Encode()
		response, err := service.client.Do(ctx, roblox.Request{
			Endpoint:        "avatar-headshots",
			Method:          http.MethodGet,
			URL:             &requestURL,
			MaxResponseSize: 2 << 20,
			Retry: roblox.RetryPolicy{
				MaxAttempts:       3,
				RetryServerErrors: true,
				RetryRateLimit:    true,
			},
		})
		if err != nil {
			return nil, err
		}
		var payload struct {
			Data []struct {
				TargetID int64  `json:"targetId"`
				State    string `json:"state"`
				ImageURL string `json:"imageUrl"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Body, &payload); err != nil {
			return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "avatar-headshots", Status: response.Status, Message: "Roblox returned invalid avatar headshot data.", Cause: err}
		}
		for _, item := range payload.Data {
			if _, requested := seen[item.TargetID]; !requested || !strings.EqualFold(item.State, "Completed") || item.ImageURL == "" {
				continue
			}
			if !safeThumbnailURL(item.ImageURL) {
				return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "avatar-headshots", Status: response.Status, Message: "Roblox returned an unsafe avatar headshot URL."}
			}
			result = append(result, AvatarHeadshotView{RobloxUserID: item.TargetID, ImageURL: item.ImageURL})
		}
	}
	return result, nil
}

func safeThumbnailURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") || parsed.User != nil || parsed.Fragment != "" {
		return false
	}
	if parsed.Port() != "" && parsed.Port() != "443" {
		return false
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host != "rbxcdn.com" && !strings.HasSuffix(host, ".rbxcdn.com") {
		return false
	}
	return parsed.Path != "" && parsed.Path != "/"
}
