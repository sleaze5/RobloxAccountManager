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
	onlineStatusWorkerLimit = 4
	onlineStatusBatchSize   = 100
)

type OnlineStatusType int

const (
	OnlineStatusTypeOffline   OnlineStatusType = 0
	OnlineStatusTypeOnline    OnlineStatusType = 1
	OnlineStatusTypeInGame    OnlineStatusType = 2
	OnlineStatusTypeInStudio  OnlineStatusType = 3
	OnlineStatusTypeInvisible OnlineStatusType = 4
)

type OnlineStatus struct {
	UserID       int64            `json:"userId"`
	StatusType   OnlineStatusType `json:"statusType"`
	LastLocation string           `json:"lastLocation,omitempty"`
	PlaceID      *int64           `json:"placeId,omitempty"`
	RootPlaceID  *int64           `json:"rootPlaceId,omitempty"`
	ServerID     string           `json:"serverId,omitempty"`
	UniverseID   *int64           `json:"universeId,omitempty"`
}

type OnlineStatusTarget struct {
	AccountID    int64
	RobloxUserID int64
}

type OnlineStatuses struct {
	client   *roblox.Client
	endpoint *url.URL
}

func NewOnlineStatuses(client *roblox.Client) *OnlineStatuses {
	endpoint, _ := url.Parse("https://presence.roblox.com/v1/presence/users")
	return &OnlineStatuses{client: client, endpoint: endpoint}
}

// AccountOnlineStatuses reads each account's online status with that account's own cookie.
// Roblox reports a user as offline to viewers excluded by the user's online-status
// privacy, so a batch read through one account can hide the others' real online status.
// Accounts whose read fails are omitted so callers keep their last known online status.
func (service *OnlineStatuses) AccountOnlineStatuses(ctx context.Context, targets []OnlineStatusTarget) ([]OnlineStatus, error) {
	targets, err := normalizeOnlineStatusTargets(targets)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return []OnlineStatus{}, nil
	}

	views := make([]OnlineStatus, len(targets))
	failures := make([]error, len(targets))
	jobs := make(chan int, len(targets))
	for index := range targets {
		jobs <- index
	}
	close(jobs)
	var workers sync.WaitGroup
	for range min(onlineStatusWorkerLimit, len(targets)) {
		workers.Go(func() {
			for index := range jobs {
				views[index], failures[index] = service.requestOnlineStatus(ctx, targets[index])
			}
		})
	}
	workers.Wait()

	available := make([]OnlineStatus, 0, len(targets))
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

func normalizeOnlineStatusTargets(targets []OnlineStatusTarget) ([]OnlineStatusTarget, error) {
	unique := make([]OnlineStatusTarget, 0, len(targets))
	seen := make(map[int64]struct{}, len(targets))
	for _, target := range targets {
		if target.AccountID <= 0 || target.RobloxUserID <= 0 {
			return nil, &roblox.Error{
				Kind:     roblox.KindProtocol,
				Endpoint: "user-online-statuses",
				Message:  "The online status request is invalid.",
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

// UserOnlineStatuses reads other users' online status as the account sees it.
func (service *OnlineStatuses) UserOnlineStatuses(ctx context.Context, accountID, version int64, userIDs []int64) ([]OnlineStatus, error) {
	onlineStatuses := make([]OnlineStatus, 0, len(userIDs))
	for batch := range slices.Chunk(userIDs, onlineStatusBatchSize) {
		page, err := service.request(ctx, accountID, version, batch)
		if err != nil {
			return nil, err
		}
		onlineStatuses = append(onlineStatuses, page...)
	}
	return onlineStatuses, nil
}

func (service *OnlineStatuses) requestOnlineStatus(ctx context.Context, target OnlineStatusTarget) (OnlineStatus, error) {
	onlineStatuses, err := service.request(ctx, target.AccountID, 0, []int64{target.RobloxUserID})
	if err != nil {
		return OnlineStatus{}, err
	}
	for _, onlineStatus := range onlineStatuses {
		if onlineStatus.UserID == target.RobloxUserID {
			return onlineStatus, nil
		}
	}
	return OnlineStatus{}, invalidOnlineStatus(http.StatusOK, nil)
}

func (service *OnlineStatuses) request(ctx context.Context, accountID, version int64, userIDs []int64) ([]OnlineStatus, error) {
	body, err := json.Marshal(struct {
		UserIDs []int64 `json:"userIds"`
	}{UserIDs: userIDs})
	if err != nil {
		return nil, &roblox.Error{
			Kind:     roblox.KindProtocol,
			Endpoint: "user-online-statuses",
			Message:  "Could not create the online status request.",
			Cause:    err,
		}
	}
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint:              "user-online-statuses",
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
	// Roblox calls online statuses presences.
	var payload struct {
		UserOnlineStatuses []struct {
			UserID       int64  `json:"userId"`
			Type         int    `json:"userPresenceType"`
			LastLocation string `json:"lastLocation"`
			PlaceID      *int64 `json:"placeId"`
			RootPlaceID  *int64 `json:"rootPlaceId"`
			// Roblox calls the server ID gameId.
			ServerID   string `json:"gameId"`
			UniverseID *int64 `json:"universeId"`
		} `json:"userPresences"`
	}
	if err := json.Unmarshal(response.Body, &payload); err != nil {
		return nil, invalidOnlineStatus(response.Status, err)
	}
	onlineStatuses := make([]OnlineStatus, 0, len(payload.UserOnlineStatuses))
	for _, onlineStatus := range payload.UserOnlineStatuses {
		onlineStatuses = append(onlineStatuses, OnlineStatus{
			UserID:       onlineStatus.UserID,
			StatusType:   onlineStatusType(onlineStatus.Type),
			LastLocation: strings.TrimSpace(onlineStatus.LastLocation),
			PlaceID:      onlineStatus.PlaceID,
			RootPlaceID:  onlineStatus.RootPlaceID,
			ServerID:     strings.TrimSpace(onlineStatus.ServerID),
			UniverseID:   onlineStatus.UniverseID,
		})
	}
	return onlineStatuses, nil
}

func invalidOnlineStatus(status int, cause error) error {
	return &roblox.Error{
		Kind:     roblox.KindProtocol,
		Endpoint: "user-online-statuses",
		Status:   status,
		Message:  "Roblox returned an invalid online status.",
		Cause:    cause,
	}
}

func onlineStatusType(code int) OnlineStatusType {
	switch code {
	case 1:
		return OnlineStatusTypeOnline
	case 2:
		return OnlineStatusTypeInGame
	case 3:
		return OnlineStatusTypeInStudio
	case 4:
		return OnlineStatusTypeInvisible
	default:
		return OnlineStatusTypeOffline
	}
}
