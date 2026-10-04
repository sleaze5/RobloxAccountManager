package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	presenceWorkerLimit        = 4
	maxPresenceUsersPerRequest = 50
)

type PresenceType int

const (
	PresenceTypeUnknown   PresenceType = -1
	PresenceTypeOffline   PresenceType = 0
	PresenceTypeOnline    PresenceType = 1
	PresenceTypeInGame    PresenceType = 2
	PresenceTypeInStudio  PresenceType = 3
	PresenceTypeInvisible PresenceType = 4
)

type UserPresence struct {
	UserID           int64        `json:"userId"`
	UserPresenceType PresenceType `json:"userPresenceType"`
	LastLocation     string       `json:"lastLocation,omitempty"`
	PlaceID          *int64       `json:"placeId,omitempty"`
	RootPlaceID      *int64       `json:"rootPlaceId,omitempty"`
	GameID           string       `json:"gameId,omitempty"`
	UniverseID       *int64       `json:"universeId,omitempty"`
}

type PresenceTarget struct {
	AccountID    int64
	RobloxUserID int64
}

type Presence struct {
	client   *roblox.Client
	endpoint *url.URL
}

func NewPresence(client *roblox.Client) *Presence {
	endpoint, _ := url.Parse("https://presence.roblox.com/v1/presence/users")
	return &Presence{client: client, endpoint: endpoint}
}

func (service *Presence) AccountPresences(ctx context.Context, targets []PresenceTarget) ([]UserPresence, error) {
	targets, err := normalizePresenceTargets(targets)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return []UserPresence{}, nil
	}

	type job struct {
		start   int
		targets []PresenceTarget
	}
	type result struct {
		start int
		views []UserPresence
		err   error
	}

	batchCount := (len(targets) + maxPresenceUsersPerRequest - 1) / maxPresenceUsersPerRequest
	jobs := make(chan job, batchCount)
	results := make(chan result, batchCount)
	for start := 0; start < len(targets); start += maxPresenceUsersPerRequest {
		end := min(start+maxPresenceUsersPerRequest, len(targets))
		jobs <- job{start: start, targets: targets[start:end]}
	}
	close(jobs)

	var workers sync.WaitGroup
	for range min(presenceWorkerLimit, batchCount) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for job := range jobs {
				views, err := service.requestPresences(ctx, job.targets)
				results <- result{start: job.start, views: views, err: err}
			}
		}()
	}
	workers.Wait()
	close(results)

	ordered := make([]UserPresence, len(targets))
	available := make([]bool, len(targets))
	availableCount := 0
	var firstError error
	for result := range results {
		if result.err != nil {
			if firstError == nil {
				firstError = result.err
			}
			continue
		}
		for index, view := range result.views {
			position := result.start + index
			ordered[position] = view
			available[position] = true
			availableCount++
		}
	}
	if firstError != nil && (availableCount == 0 || ctx.Err() != nil) {
		return nil, firstError
	}

	views := make([]UserPresence, 0, availableCount)
	for index, view := range ordered {
		if available[index] {
			views = append(views, view)
		}
	}
	return views, nil
}

func normalizePresenceTargets(targets []PresenceTarget) ([]PresenceTarget, error) {
	unique := make([]PresenceTarget, 0, len(targets))
	seen := make(map[int64]struct{}, len(targets))
	for _, target := range targets {
		if target.AccountID <= 0 || target.RobloxUserID <= 0 {
			return nil, &roblox.Error{
				Kind:     roblox.KindProtocol,
				Endpoint: "user-presences",
				Message:  "An invalid account presence was requested.",
			}
		}
		if _, exists := seen[target.RobloxUserID]; exists {
			continue
		}
		seen[target.RobloxUserID] = struct{}{}
		unique = append(unique, target)
	}
	return unique, nil
}

func (service *Presence) requestPresences(ctx context.Context, targets []PresenceTarget) ([]UserPresence, error) {
	userIDs := make([]int64, len(targets))
	for index, target := range targets {
		userIDs[index] = target.RobloxUserID
	}
	body, err := json.Marshal(struct {
		UserIDs []int64 `json:"userIds"`
	}{UserIDs: userIDs})
	if err != nil {
		return nil, &roblox.Error{
			Kind:     roblox.KindProtocol,
			Endpoint: "user-presences",
			Message:  "The presence request could not be created.",
			Cause:    err,
		}
	}
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint:        "user-presences",
		AccountID:       targets[0].AccountID,
		Authenticated:   true,
		Method:          http.MethodPost,
		URL:             service.endpoint,
		Body:            body,
		ContentType:     "application/json",
		RequiresCSRF:    true,
		MaxResponseSize: 512 << 10,
		Retry: roblox.RetryPolicy{
			MaxAttempts:       3,
			Idempotent:        true,
			RetryServerErrors: true,
			RetryRateLimit:    true,
		},
	})
	if err != nil {
		return nil, err
	}
	var payload struct {
		UserPresences []struct {
			UserID       int64  `json:"userId"`
			Type         int    `json:"userPresenceType"`
			LastLocation string `json:"lastLocation"`
			PlaceID      *int64 `json:"placeId"`
			RootPlaceID  *int64 `json:"rootPlaceId"`
			GameID       string `json:"gameId"`
			UniverseID   *int64 `json:"universeId"`
		} `json:"userPresences"`
	}
	if err := json.Unmarshal(response.Body, &payload); err != nil {
		return nil, &roblox.Error{
			Kind:     roblox.KindProtocol,
			Endpoint: "user-presences",
			Status:   response.Status,
			Message:  "Roblox returned invalid presence data.",
			Cause:    err,
		}
	}
	requested := make(map[int64]struct{}, len(targets))
	for _, target := range targets {
		requested[target.RobloxUserID] = struct{}{}
	}
	byUserID := make(map[int64]UserPresence, len(payload.UserPresences))
	for _, item := range payload.UserPresences {
		if _, expected := requested[item.UserID]; !expected {
			continue
		}
		byUserID[item.UserID] = UserPresence{
			UserID:           item.UserID,
			UserPresenceType: presenceType(item.Type),
			LastLocation:     strings.TrimSpace(item.LastLocation),
			PlaceID:          item.PlaceID,
			RootPlaceID:      item.RootPlaceID,
			GameID:           strings.TrimSpace(item.GameID),
			UniverseID:       item.UniverseID,
		}
	}
	views := make([]UserPresence, len(targets))
	for index, target := range targets {
		view, exists := byUserID[target.RobloxUserID]
		if !exists {
			view = UserPresence{UserID: target.RobloxUserID, UserPresenceType: PresenceTypeUnknown}
		}
		views[index] = view
	}
	return views, nil
}

func presenceType(value int) PresenceType {
	switch value {
	case 0:
		return PresenceTypeOffline
	case 1:
		return PresenceTypeOnline
	case 2:
		return PresenceTypeInGame
	case 3:
		return PresenceTypeInStudio
	case 4:
		return PresenceTypeInvisible
	default:
		return PresenceTypeUnknown
	}
}
