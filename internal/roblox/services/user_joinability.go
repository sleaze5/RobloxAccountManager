package services

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	userJoinabilityEndpoint = "user-joinability"
	// RoValra keeps a joinability answer for 30 seconds, because a user can
	// change servers at any time.
	userJoinabilityKeepDuration = 30 * time.Second
)

type accountUser struct {
	accountID int64
	userID    int64
}

type keptUserJoinability struct {
	joinability games.UserJoinability
	expiresAt   time.Time
}

// UserJoinability reads whether the user's profile offers the account to join
// them, from the JoinUserActions component that RoValra reads. It is lighter
// than a join check, and it gives no reason.
func (service *Users) UserJoinability(ctx context.Context, accountID, version, userID int64) (games.UserJoinability, error) {
	key := accountUser{accountID: accountID, userID: userID}
	now := time.Now()
	service.joinabilityMutex.Lock()
	kept, isKept := service.keptJoinabilityByAccountUser[key]
	service.joinabilityMutex.Unlock()
	if isKept && now.Before(kept.expiresAt) {
		return kept.joinability, nil
	}
	userIDText := strconv.FormatInt(userID, 10)
	response, err := service.writeAccount(ctx, accountID, version, userJoinabilityEndpoint, http.MethodPost, "https://apis.roblox.com/profile-platform-api/v1/profiles/batch/get", map[string]any{
		"profileType": "User",
		"profileIds":  []string{userIDText},
		"components":  []map[string]string{{"component": "JoinUserActions"}},
	})
	if err != nil {
		return games.UserJoinabilityUnknown, err
	}
	var payload struct {
		Profiles map[string]struct {
			Components struct {
				JoinUserActions *struct {
					Actions []struct {
						Joinability string `json:"joinability"`
					} `json:"actions"`
				} `json:"JoinUserActions"`
			} `json:"components"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(response.Body, &payload); err != nil {
		return games.UserJoinabilityUnknown, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: userJoinabilityEndpoint, Status: response.Status, Message: "Roblox returned invalid profile details.", Cause: err}
	}
	joinability := games.UserJoinabilityUnknown
	if components := payload.Profiles[userIDText].Components.JoinUserActions; components != nil {
		for _, action := range components.Actions {
			if action.Joinability != "" {
				joinability = userJoinabilityOf(action.Joinability)
				break
			}
		}
	}
	service.keepUserJoinability(key, joinability, now)
	return joinability, nil
}

func userJoinabilityOf(joinability string) games.UserJoinability {
	switch joinability {
	case "Joinable":
		return games.UserJoinable
	case "NotJoinable":
		return games.UserNotJoinable
	default:
		return games.UserJoinabilityUnknown
	}
}

// keepUserJoinability keeps an answer and drops the expired ones, so that the
// kept answers stay as few as the users viewed in the last 30 seconds.
func (service *Users) keepUserJoinability(key accountUser, joinability games.UserJoinability, now time.Time) {
	service.joinabilityMutex.Lock()
	defer service.joinabilityMutex.Unlock()
	for keptKey, kept := range service.keptJoinabilityByAccountUser {
		if !now.Before(kept.expiresAt) {
			delete(service.keptJoinabilityByAccountUser, keptKey)
		}
	}
	service.keptJoinabilityByAccountUser[key] = keptUserJoinability{joinability: joinability, expiresAt: now.Add(userJoinabilityKeepDuration)}
}
