package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	presenceWorkerLimit = 4
	presenceBatchSize   = 100
)

type PresenceType int

const (
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

// AccountPresences reads each account's presence with that account's own session.
// Roblox reports a user as offline to viewers excluded by the user's online-status
// privacy, so a batch read through one account can hide the others' real presence.
// Accounts whose read fails are omitted so callers keep their last known presence.
func (service *Presence) AccountPresences(ctx context.Context, targets []PresenceTarget) ([]UserPresence, error) {
	targets, err := normalizePresenceTargets(targets)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return []UserPresence{}, nil
	}

	views := make([]UserPresence, len(targets))
	failures := make([]error, len(targets))
	jobs := make(chan int, len(targets))
	for index := range targets {
		jobs <- index
	}
	close(jobs)
	var workers sync.WaitGroup
	for range min(presenceWorkerLimit, len(targets)) {
		workers.Go(func() {
			for index := range jobs {
				views[index], failures[index] = service.requestPresence(ctx, targets[index])
			}
		})
	}
	workers.Wait()

	available := make([]UserPresence, 0, len(targets))
	var firstError error
	for index, failure := range failures {
		if failure == nil {
			available = append(available, views[index])
		} else if firstError == nil {
			firstError = failure
		}
	}
	if firstError != nil && (len(available) == 0 || ctx.Err() != nil) {
		return nil, firstError
	}
	return available, nil
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

// UserPresences reads other users' presence as the account sees it.
func (service *Presence) UserPresences(ctx context.Context, accountID, version int64, userIDs []int64) ([]UserPresence, error) {
	presences := make([]UserPresence, 0, len(userIDs))
	for batch := range slices.Chunk(userIDs, presenceBatchSize) {
		page, err := service.request(ctx, accountID, version, batch)
		if err != nil {
			return nil, err
		}
		presences = append(presences, page...)
	}
	return presences, nil
}

func (service *Presence) requestPresence(ctx context.Context, target PresenceTarget) (UserPresence, error) {
	presences, err := service.request(ctx, target.AccountID, 0, []int64{target.RobloxUserID})
	if err != nil {
		return UserPresence{}, err
	}
	for _, presence := range presences {
		if presence.UserID == target.RobloxUserID {
			return presence, nil
		}
	}
	return UserPresence{}, invalidPresence(http.StatusOK, nil)
}

func (service *Presence) request(ctx context.Context, accountID, version int64, userIDs []int64) ([]UserPresence, error) {
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
		Endpoint:              "user-presences",
		AccountID:             accountID,
		ExpectedSecretVersion: version,
		Authenticated:         true,
		Method:                http.MethodPost,
		URL:                   service.endpoint,
		Body:                  body,
		ContentType:           "application/json",
		RequiresCSRF:          true,
		MaxResponseSize:       256 << 10,
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
		return nil, invalidPresence(response.Status, err)
	}
	presences := make([]UserPresence, 0, len(payload.UserPresences))
	for _, userPresence := range payload.UserPresences {
		presences = append(presences, UserPresence{
			UserID:           userPresence.UserID,
			UserPresenceType: presenceType(userPresence.Type),
			LastLocation:     strings.TrimSpace(userPresence.LastLocation),
			PlaceID:          userPresence.PlaceID,
			RootPlaceID:      userPresence.RootPlaceID,
			GameID:           strings.TrimSpace(userPresence.GameID),
			UniverseID:       userPresence.UniverseID,
		})
	}
	return presences, nil
}

func invalidPresence(status int, cause error) error {
	return &roblox.Error{
		Kind:     roblox.KindProtocol,
		Endpoint: "user-presences",
		Status:   status,
		Message:  "Roblox returned invalid presence data.",
		Cause:    cause,
	}
}

func presenceType(code int) PresenceType {
	switch code {
	case 1:
		return PresenceTypeOnline
	case 2:
		return PresenceTypeInGame
	case 3:
		return PresenceTypeInStudio
	case 4:
		return PresenceTypeInvisible
	default:
		return PresenceTypeOffline
	}
}
