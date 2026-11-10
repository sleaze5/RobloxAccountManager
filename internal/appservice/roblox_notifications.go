package appservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/sleaze5/RobloxAccountManager/internal/logging"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

const (
	maxRobloxNotificationStartIndex = 1000
	maxRobloxNotificationIDLength   = 64
	maxRobloxNotificationActionSize = 128
	// Dismiss all reads at most this many pages of 20 before it dismisses.
	maxRobloxNotificationDismissPages = 25
)

var robloxNotificationOperationID atomic.Uint64

type RobloxNotificationImageKind string

const (
	RobloxNotificationImageUser  RobloxNotificationImageKind = "user"
	RobloxNotificationImageGroup RobloxNotificationImageKind = "group"
	RobloxNotificationImageGame  RobloxNotificationImageKind = "game"
	RobloxNotificationImageOther RobloxNotificationImageKind = "other"
)

type RobloxNotificationCategory string

const (
	RobloxNotificationInvites     RobloxNotificationCategory = "invites"
	RobloxNotificationFriends     RobloxNotificationCategory = "friends"
	RobloxNotificationCommunities RobloxNotificationCategory = "communities"
	RobloxNotificationExperiences RobloxNotificationCategory = "experiences"
	RobloxNotificationOther       RobloxNotificationCategory = "other"
)

// RobloxNotificationUser is the person a notification is about, such as the
// sender of an invite. Nickname is the account's private name for a friend.
type RobloxNotificationUser struct {
	UserID      int64  `json:"userId"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Nickname    string `json:"nickname"`
}

// RobloxNotificationJoin is the experience that an invite points to.
// UniverseID is 0 when Roblox does not report it.
type RobloxNotificationJoin struct {
	PlaceID    int64 `json:"placeId"`
	UniverseID int64 `json:"universeId"`
}

// RobloxNotificationAction is the main button of a notification. Exactly one of
// Join, ChatUserID, and LinkURL is set.
type RobloxNotificationAction struct {
	Label      string                  `json:"label"`
	Join       *RobloxNotificationJoin `json:"join"`
	ChatUserID int64                   `json:"chatUserId"`
	LinkURL    string                  `json:"linkUrl"`
}

type RobloxNotificationView struct {
	ID          string                      `json:"id"`
	Title       string                      `json:"title"`
	Body        string                      `json:"body"`
	CreatedAtMs int64                       `json:"createdAtMs"`
	ImageURL    string                      `json:"imageUrl"`
	ImageKind   RobloxNotificationImageKind `json:"imageKind"`
	Category    RobloxNotificationCategory  `json:"category"`
	User        *RobloxNotificationUser     `json:"user"`
	// DismissAction is the server action of the Dismiss button. It is empty
	// when Roblox offers no dismissal that persists.
	DismissAction string                    `json:"dismissAction"`
	Action        *RobloxNotificationAction `json:"action"`
}

type RobloxNotificationPage struct {
	Notifications []RobloxNotificationView `json:"notifications"`
	// NextStartIndex is 0 when no more notifications exist.
	NextStartIndex int `json:"nextStartIndex"`
}

func (service *Service) GetRobloxNotifications(ctx context.Context, accountID int64, startIndex int) (RobloxNotificationPage, error) {
	if startIndex < 0 || startIndex > maxRobloxNotificationStartIndex {
		return RobloxNotificationPage{}, robloxNotificationInputError("These notifications have expired. Refresh notifications.")
	}
	ctx, version, logger, done, err := service.beginRobloxNotifications(ctx, accountID, "list")
	if err != nil {
		return RobloxNotificationPage{}, err
	}
	defer done()
	entries, err := service.notificationStream.Recent(ctx, accountID, version, startIndex)
	if err != nil {
		return RobloxNotificationPage{}, robloxNotificationFailure(ctx, logger, err)
	}
	views, err := service.robloxNotificationViews(ctx, logger, accountID, version, entries)
	if err != nil {
		return RobloxNotificationPage{}, err
	}
	page := RobloxNotificationPage{Notifications: views}
	if len(entries) >= robloxservices.NotificationStreamPageSize {
		page.NextStartIndex = startIndex + len(entries)
	}
	return page, nil
}

func (service *Service) DismissRobloxNotification(ctx context.Context, accountID int64, notificationID, dismissAction string) error {
	if !isRobloxNotificationToken(notificationID, maxRobloxNotificationIDLength) || !isRobloxNotificationToken(dismissAction, maxRobloxNotificationActionSize) {
		return robloxNotificationInputError("This notification is unavailable. Refresh notifications.")
	}
	ctx, version, logger, done, err := service.beginRobloxNotifications(ctx, accountID, "dismiss")
	if err != nil {
		return err
	}
	defer done()
	if err := service.notificationStream.RunAction(ctx, accountID, version, notificationID, dismissAction); err != nil {
		return robloxNotificationFailure(ctx, logger, err)
	}
	logger.DebugContext(ctx, "roblox notification dismissed")
	return nil
}

// DismissAllRobloxNotifications dismisses every notification of the category,
// or of every category when it is empty, that Roblox lets the account dismiss.
// It returns how many it dismissed.
func (service *Service) DismissAllRobloxNotifications(ctx context.Context, accountID int64, category RobloxNotificationCategory) (int, error) {
	switch category {
	case "", RobloxNotificationInvites, RobloxNotificationFriends, RobloxNotificationCommunities, RobloxNotificationExperiences, RobloxNotificationOther:
	default:
		return 0, robloxNotificationInputError("Choose a notification type to dismiss.")
	}
	ctx, version, logger, done, err := service.beginRobloxNotifications(ctx, accountID, "dismiss-all")
	if err != nil {
		return 0, err
	}
	defer done()
	type dismissal struct{ notificationID, action string }
	dismissals, startIndex := []dismissal{}, 0
	// Read every page before dismissing, because each dismissal shifts the
	// start index of the entries after it.
	for range maxRobloxNotificationDismissPages {
		entries, err := service.notificationStream.Recent(ctx, accountID, version, startIndex)
		if err != nil {
			return 0, robloxNotificationFailure(ctx, logger, err)
		}
		for _, entry := range entries {
			if category != "" && (entry.Content == nil || robloxNotificationCategory(entry.Content.NotificationType) != category) {
				continue
			}
			if action := robloxNotificationDismissAction(entry); action != "" && isRobloxNotificationToken(entry.ID, maxRobloxNotificationIDLength) {
				dismissals = append(dismissals, dismissal{notificationID: entry.ID, action: action})
			}
		}
		if len(entries) < robloxservices.NotificationStreamPageSize {
			break
		}
		startIndex += len(entries)
	}
	for index, pending := range dismissals {
		if err := service.notificationStream.RunAction(ctx, accountID, version, pending.notificationID, pending.action); err != nil {
			logger.WarnContext(ctx, "roblox notification dismissal stopped", "dismissed", index, "total", len(dismissals), "error", err)
			return index, robloxNotificationInputError(fmt.Sprintf("Dismissed %d of %d notifications. Try again to dismiss the rest.", index, len(dismissals)))
		}
	}
	logger.InfoContext(ctx, "roblox notifications dismissed", "count", len(dismissals))
	return len(dismissals), nil
}

func (service *Service) beginRobloxNotifications(parent context.Context, accountID int64, operation string) (context.Context, int64, *slog.Logger, func(), error) {
	ctx, version, done, err := service.beginAccountSession(parent, accountID, "roblox-notifications")
	if err != nil {
		return nil, 0, nil, nil, err
	}
	ctx = logging.WithOperation(ctx, fmt.Sprintf("roblox-notifications-%d", robloxNotificationOperationID.Add(1)))
	logger := service.logs.Module("application.roblox-notifications").With("operation", operation)
	return ctx, version, logger, done, nil
}

func (service *Service) robloxNotificationViews(ctx context.Context, logger *slog.Logger, accountID, version int64, entries []robloxservices.NotificationStreamEntry) ([]RobloxNotificationView, error) {
	views := make([]RobloxNotificationView, 0, len(entries))
	targets := make([]robloxservices.ThumbnailTarget, 0, len(entries))
	userIDs := []int64{}
	for _, entry := range entries {
		if view, target, ok := robloxNotificationView(entry); ok {
			views = append(views, view)
			targets = append(targets, target)
			if view.User != nil {
				userIDs = append(userIDs, view.User.UserID)
			}
		}
	}
	var images map[robloxservices.ThumbnailTarget]string
	var nicknames map[int64]string
	var users []robloxservices.UserView
	reads := []accountRead{{"images", func() error {
		var readError error
		images, readError = service.thumbnails.Images(ctx, targets)
		return readError
	}}}
	if len(userIDs) > 0 {
		reads = append(reads, accountRead{"names", func() error {
			var readError error
			users, readError = service.users.Names(ctx, userIDs)
			return readError
		}}, accountRead{"nicknames", func() error {
			var readError error
			nicknames, readError = service.users.Nicknames(ctx, accountID, version, userIDs)
			return readError
		}})
	}
	if _, err := service.runAccountReads(ctx, accountID, version, "roblox-notifications", logger, reads); err != nil {
		return nil, err
	}
	userByID := make(map[int64]robloxservices.UserView, len(users))
	for _, user := range users {
		userByID[user.UserID] = user
	}
	for index := range views {
		view := &views[index]
		view.ImageURL = images[targets[index]]
		if view.User != nil {
			user := userByID[view.User.UserID]
			view.User.Username, view.User.DisplayName, view.User.Nickname = user.Username, user.DisplayName, nicknames[view.User.UserID]
		}
	}
	return views, nil
}

// robloxNotificationView reads the current state of an entry. The target is the
// image to show, or the zero target when the entry has none.
func robloxNotificationView(entry robloxservices.NotificationStreamEntry) (RobloxNotificationView, robloxservices.ThumbnailTarget, bool) {
	state, ok := robloxNotificationState(entry)
	if !ok || entry.ID == "" {
		return RobloxNotificationView{}, robloxservices.ThumbnailTarget{}, false
	}
	view := RobloxNotificationView{
		ID:            entry.ID,
		CreatedAtMs:   chatTime(entry.EventDate),
		ImageKind:     RobloxNotificationImageOther,
		Category:      robloxNotificationCategory(entry.Content.NotificationType),
		DismissAction: robloxNotificationDismissAction(entry),
	}
	if len(state.VisualItems.TextBody) > 0 {
		text := state.VisualItems.TextBody[0]
		if text.Title != nil {
			view.Title = strings.TrimSpace(text.Title.Text)
		}
		if text.Label != nil {
			view.Body = strings.TrimSpace(text.Label.Text)
		}
	}
	if view.Title == "" && view.Body == "" {
		return RobloxNotificationView{}, robloxservices.ThumbnailTarget{}, false
	}
	var target robloxservices.ThumbnailTarget
	if len(state.VisualItems.Thumbnail) > 0 {
		thumbnail := state.VisualItems.Thumbnail[0]
		targetID, _ := strconv.ParseInt(thumbnail.ID, 10, 64)
		target, view.ImageKind = robloxNotificationThumbnailTarget(thumbnail.IDType, targetID)
		if view.ImageKind == RobloxNotificationImageUser {
			view.User = &RobloxNotificationUser{UserID: targetID}
		}
	}
	view.Action = robloxNotificationMainAction(state.VisualItems.Button, robloxNotificationUniverseID(entry.Content.ClientEventsPayload))
	return view, target, true
}

// robloxNotificationCategory groups notification types for the type filter.
// Roblox does not document its notification types, so types are matched by
// keyword. Observed types on 2026-11-09: ExperienceInvitation, GroupShout, and
// FriendRequestAccepted.
func robloxNotificationCategory(notificationType string) RobloxNotificationCategory {
	name := strings.ToLower(notificationType)
	contains := func(keywords ...string) bool {
		return slices.ContainsFunc(keywords, func(keyword string) bool { return strings.Contains(name, keyword) })
	}
	switch {
	case contains("invit"):
		return RobloxNotificationInvites
	case contains("friend", "connection", "follow"):
		return RobloxNotificationFriends
	case contains("group", "community", "shout"):
		return RobloxNotificationCommunities
	case contains("game", "experience", "universe", "place", "badge", "event"):
		return RobloxNotificationExperiences
	default:
		return RobloxNotificationOther
	}
}

// robloxNotificationUniverseID reads universe_id from the analytics payload of
// an experience invite. Roblox sends it as a string, as observed on 2026-11-09.
func robloxNotificationUniverseID(payload json.RawMessage) int64 {
	var analytics struct {
		UniverseID json.Number `json:"universe_id"`
	}
	if json.Unmarshal(payload, &analytics) != nil {
		return 0
	}
	universeID, err := analytics.UniverseID.Int64()
	if err != nil || universeID <= 0 {
		return 0
	}
	return universeID
}

func robloxNotificationState(entry robloxservices.NotificationStreamEntry) (robloxservices.NotificationStreamState, bool) {
	if entry.Content == nil {
		return robloxservices.NotificationStreamState{}, false
	}
	state, ok := entry.Content.States[entry.Content.CurrentState]
	return state, ok
}

func robloxNotificationThumbnailTarget(idType string, targetID int64) (robloxservices.ThumbnailTarget, RobloxNotificationImageKind) {
	if targetID <= 0 {
		return robloxservices.ThumbnailTarget{}, RobloxNotificationImageOther
	}
	// Thumbnail ID types from the Roblox web notification stream in
	// Roblox-Web-Datamining. A game thumbnail ID is a universe ID.
	switch idType {
	case "userThumbnail":
		return robloxservices.ThumbnailTarget{Type: "AvatarHeadShot", TargetID: targetID}, RobloxNotificationImageUser
	case "groupThumbnail":
		return robloxservices.ThumbnailTarget{Type: "GroupIcon", TargetID: targetID}, RobloxNotificationImageGroup
	case "gameThumbnail":
		return robloxservices.ThumbnailTarget{Type: "GameIcon", TargetID: targetID}, RobloxNotificationImageGame
	case "assetThumbnail":
		return robloxservices.ThumbnailTarget{Type: "Asset", TargetID: targetID}, RobloxNotificationImageOther
	case "bundleThumbnail":
		return robloxservices.ThumbnailTarget{Type: "BundleThumbnail", TargetID: targetID}, RobloxNotificationImageOther
	case "gamePassThumbnail":
		return robloxservices.ThumbnailTarget{Type: "GamePass", TargetID: targetID}, RobloxNotificationImageOther
	default:
		return robloxservices.ThumbnailTarget{}, RobloxNotificationImageOther
	}
}

// robloxNotificationDismissAction finds the Dismiss button: a button that hides
// the entry after it runs a server action. A button that only hides the entry
// on the website does not dismiss it on Roblox, so it does not count.
func robloxNotificationDismissAction(entry robloxservices.NotificationStreamEntry) string {
	state, ok := robloxNotificationState(entry)
	if !ok {
		return ""
	}
	for _, button := range state.VisualItems.Button {
		if action := buttonDismissAction(button); action != "" {
			return action
		}
	}
	return ""
}

func buttonDismissAction(button robloxservices.NotificationStreamButton) string {
	serverAction, hides := "", false
	for _, action := range button.Actions {
		switch action.ActionType {
		case "notificationAPI":
			serverAction = action.Path
		case "dismiss":
			hides = true
		}
	}
	if !hides || !isRobloxNotificationToken(serverAction, maxRobloxNotificationActionSize) {
		return ""
	}
	return serverAction
}

func robloxNotificationMainAction(buttons []robloxservices.NotificationStreamButton, universeID int64) *RobloxNotificationAction {
	for _, button := range buttons {
		if buttonDismissAction(button) != "" {
			continue
		}
		for _, action := range button.Actions {
			if action.ActionType != "deeplink" {
				continue
			}
			if mapped := robloxNotificationDeeplink(action.Path, universeID); mapped != nil {
				mapped.Label = strings.TrimSpace(button.Label.Text)
				return mapped
			}
		}
	}
	return nil
}

// robloxNotificationDeeplink maps a notification link to an action that this
// application can run.
func robloxNotificationDeeplink(path string, universeID int64) *RobloxNotificationAction {
	if parsed, err := url.Parse(path); err == nil && roblox.IsRobloxURL(parsed) {
		return &RobloxNotificationAction{LinkURL: parsed.String()}
	}
	route, query, ok := parseRobloxAppLink(path)
	if !ok {
		return nil
	}
	switch route {
	case "", "experiences/start":
		if placeID := positiveQueryID(query, "placeId"); placeID > 0 {
			return &RobloxNotificationAction{Join: &RobloxNotificationJoin{PlaceID: placeID, UniverseID: universeID}}
		}
	case "navigation/chat":
		if userID := positiveQueryID(query, "userId"); userID > 0 {
			return &RobloxNotificationAction{ChatUserID: userID}
		}
	case "navigation/profile":
		if userID := positiveQueryID(query, "userId"); userID > 0 {
			return &RobloxNotificationAction{LinkURL: fmt.Sprintf("https://www.roblox.com/users/%d/profile", userID)}
		}
	case "navigation/group":
		if groupID := positiveQueryID(query, "groupId"); groupID > 0 {
			return &RobloxNotificationAction{LinkURL: fmt.Sprintf("https://www.roblox.com/communities/%d", groupID)}
		}
	}
	return nil
}

// parseRobloxAppLink splits a roblox:// link into its lowercase route and its
// query. Observed links on 2026-11-09: roblox://placeId=1 (no route),
// roblox://navigation/chat?userId=1, roblox://navigation/group?groupId=1, and
// roblox://navigation/profile?userId=1.
func parseRobloxAppLink(path string) (string, url.Values, bool) {
	link, isRobloxLink := strings.CutPrefix(path, "roblox://")
	if !isRobloxLink {
		return "", nil, false
	}
	route, rawQuery, hasQuery := strings.Cut(link, "?")
	if !hasQuery && strings.Contains(route, "=") {
		route, rawQuery = "", route
	}
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return "", nil, false
	}
	return strings.ToLower(strings.Trim(route, "/")), query, true
}

func positiveQueryID(query url.Values, name string) int64 {
	id, err := strconv.ParseInt(query.Get(name), 10, 64)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

// isRobloxNotificationToken accepts the notification IDs (UUIDs) and action
// names (such as GroupShoutIgnoreAction) that go into the action URL path.
func isRobloxNotificationToken(token string, maxLength int) bool {
	if token == "" || len(token) > maxLength {
		return false
	}
	for _, character := range token {
		if ('a' > character || character > 'z') && ('A' > character || character > 'Z') && ('0' > character || character > '9') && character != '-' && character != '_' {
			return false
		}
	}
	return true
}

func robloxNotificationFailure(ctx context.Context, logger *slog.Logger, err error) error {
	return operationFailure(ctx, logger, err, "roblox-notifications", "The notifications request failed. Try again.")
}

func robloxNotificationInputError(message string) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "roblox-notifications", Message: message}
}
