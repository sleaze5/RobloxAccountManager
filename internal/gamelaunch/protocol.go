package gamelaunch

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func BuildProtocolURL(ticket, browserTrackerID string, request Request, now time.Time) (string, error) {
	if err := validateTicket(ticket); err != nil {
		return "", err
	}
	if browserTrackerID == "" || strings.IndexFunc(browserTrackerID, func(value rune) bool { return value < '0' || value > '9' }) >= 0 {
		return "", fmt.Errorf("browser tracker ID is invalid")
	}
	launcher, err := buildPlaceLauncherURL(browserTrackerID, request)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"roblox-player:1+launchmode:play+gameinfo:%s+launchtime:%d+placelauncherurl:%s+browsertrackerid:%s+robloxLocale:en_us+gameLocale:en_us+channel:+LaunchExp:InApp",
		ticket, now.UnixMilli(), url.QueryEscape(launcher), browserTrackerID,
	), nil
}

func validateTicket(ticket string) error {
	if ticket == "" || strings.ContainsAny(ticket, "\r\n\x00+") {
		return fmt.Errorf("authentication ticket is invalid")
	}
	return nil
}

func buildPlaceLauncherURL(browserTrackerID string, request Request) (string, error) {
	query, err := launchParameters(request)
	if err != nil {
		return "", err
	}
	query.Set("browserTrackerId", browserTrackerID)
	if request.Teleport && request.supportsTeleport() {
		query.Set("isTeleport", "true")
	}
	if request.LaunchData != "" {
		query.Set("launchData", request.LaunchData)
	}
	launcher := url.URL{Scheme: "https", Host: "assetgame.roblox.com", Path: "/game/PlaceLauncher.ashx", RawQuery: query.Encode()}
	return launcher.String(), nil
}

func (request Request) supportsTeleport() bool {
	return request.PlaceID > 0 && request.UserID == 0 && request.Username == "" && request.ShareCode == "" && request.LinkCode == "" && request.AccessCode == ""
}

func launchParameters(request Request) (url.Values, error) {
	if request.Username != "" || request.ShareCode != "" || request.ShareType != "" || (request.LinkCode != "" && request.AccessCode == "") {
		return nil, fmt.Errorf("launch target has not been resolved")
	}
	if err := validatePrivateCodes(request); err != nil {
		return nil, err
	}
	query := url.Values{}
	if request.UserID != 0 {
		if request.UserID < 0 || request.PlaceID != 0 || request.JobID != "" || request.LinkCode != "" || request.AccessCode != "" {
			return nil, fmt.Errorf("user target is invalid")
		}
		query.Set("request", "RequestFollowUser")
		query.Set("userId", strconv.FormatInt(request.UserID, 10))
		return query, nil
	}
	if request.PlaceID <= 0 {
		return nil, fmt.Errorf("place ID must be positive")
	}
	query.Set("placeId", strconv.FormatInt(request.PlaceID, 10))
	switch {
	case request.LinkCode != "" || request.AccessCode != "":
		query.Set("request", "RequestPrivateGame")
		if request.LinkCode != "" {
			query.Set("linkCode", request.LinkCode)
		}
		if request.AccessCode != "" {
			query.Set("accessCode", request.AccessCode)
		}
	case request.JobID != "":
		if !jobIDPattern.MatchString(request.JobID) {
			return nil, fmt.Errorf("job ID must be a UUID")
		}
		query.Set("request", "RequestGameJob")
		query.Set("gameId", request.JobID)
		query.Set("isPlayTogetherGame", "false")
	default:
		query.Set("request", "RequestGame")
		query.Set("isPlayTogetherGame", "false")
	}
	return query, nil
}
