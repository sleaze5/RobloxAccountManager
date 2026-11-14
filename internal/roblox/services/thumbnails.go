package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
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
	headshots := make([]AvatarHeadshotView, 0, len(unique))
	for start := 0; start < len(unique); start += headshotBatchSize {
		end := min(start+headshotBatchSize, len(unique))
		requestURL := *service.endpoint
		query := requestURL.Query()
		batchUserIDs := make([]string, 0, end-start)
		for _, userID := range unique[start:end] {
			batchUserIDs = append(batchUserIDs, strconv.FormatInt(userID, 10))
		}
		query.Set("userIds", strings.Join(batchUserIDs, ","))
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
		for _, thumbnail := range payload.Data {
			if _, requested := seen[thumbnail.TargetID]; !requested || !strings.EqualFold(thumbnail.State, "Completed") || thumbnail.ImageURL == "" {
				continue
			}
			if !safeThumbnailURL(thumbnail.ImageURL) {
				return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "avatar-headshots", Status: response.Status, Message: "Roblox returned an unsafe avatar headshot URL."}
			}
			headshots = append(headshots, AvatarHeadshotView{RobloxUserID: thumbnail.TargetID, ImageURL: thumbnail.ImageURL})
		}
	}
	return headshots, nil
}

// ThumbnailTarget names one image for the thumbnails batch endpoint. Type is a
// batch type such as AvatarHeadShot, GroupIcon, or GameIcon.
type ThumbnailTarget struct {
	Type     string
	TargetID int64
}

// The image sizes that the thumbnails batch endpoint renders. Roblox renders
// LargeThumbnailSize only for some types, such as Asset and BundleThumbnail.
const (
	SmallThumbnailSize = "150x150"
	LargeThumbnailSize = "420x420"
)

// Images reads one image of the size per target. Targets that Roblox cannot
// render are left out of the map.
func (service *Thumbnails) Images(ctx context.Context, targets []ThumbnailTarget, size string) (map[ThumbnailTarget]string, error) {
	if len(targets) > maxHeadshotIDs {
		return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "thumbnails-batch", Message: "Too many images were requested."}
	}
	type batchRequest struct {
		RequestID  string `json:"requestId"`
		TargetID   int64  `json:"targetId"`
		Type       string `json:"type"`
		Size       string `json:"size"`
		Format     string `json:"format"`
		IsCircular bool   `json:"isCircular"`
	}
	targetByRequestID := make(map[string]ThumbnailTarget, len(targets))
	requests := make([]batchRequest, 0, len(targets))
	for _, target := range targets {
		requestID := target.Type + ":" + strconv.FormatInt(target.TargetID, 10)
		if _, exists := targetByRequestID[requestID]; exists || target.TargetID <= 0 {
			continue
		}
		targetByRequestID[requestID] = target
		requests = append(requests, batchRequest{RequestID: requestID, TargetID: target.TargetID, Type: target.Type, Size: size, Format: "Png"})
	}
	endpoint := &url.URL{Scheme: "https", Host: "thumbnails.roblox.com", Path: "/v1/batch"}
	images := make(map[ThumbnailTarget]string, len(requests))
	for batch := range slices.Chunk(requests, headshotBatchSize) {
		body, err := json.Marshal(batch)
		if err != nil {
			return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "thumbnails-batch", Message: "The image request could not be encoded.", Cause: err}
		}
		response, err := service.client.Do(ctx, roblox.Request{
			Endpoint: "thumbnails-batch", Method: http.MethodPost, URL: endpoint, Body: body,
			ContentType: "application/json", MaxResponseSize: 2 << 20,
			Retry: roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
		})
		if err != nil {
			return nil, err
		}
		var payload struct {
			Data []struct {
				RequestID string `json:"requestId"`
				State     string `json:"state"`
				ImageURL  string `json:"imageUrl"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Body, &payload); err != nil {
			return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "thumbnails-batch", Status: response.Status, Message: "Roblox returned invalid image data.", Cause: err}
		}
		for _, image := range payload.Data {
			target, requested := targetByRequestID[image.RequestID]
			if requested && strings.EqualFold(image.State, "Completed") && safeThumbnailURL(image.ImageURL) {
				images[target] = image.ImageURL
			}
		}
	}
	return images, nil
}

func safeThumbnailURL(imageURL string) bool {
	parsed, err := url.Parse(imageURL)
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
