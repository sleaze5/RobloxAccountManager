package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	notificationStreamBaseURL = "https://notifications.roblox.com/v2/stream-notifications/"
	// Roblox rejects get-recent with more than 20 rows ("You cannot access that
	// many notification stream events at a time."). Verified on 2026-11-09.
	NotificationStreamPageSize      = 20
	notificationStreamResponseBytes = 4 << 20
)

// NotificationStreamEntry is one entry of the notification stream that the bell
// on the Roblox website shows. Sendr entries describe their own layout: each
// state lists the text, thumbnail, and buttons to show.
type NotificationStreamEntry struct {
	ID        string                     `json:"id"`
	EventDate string                     `json:"eventDate"`
	Content   *NotificationStreamContent `json:"content"`
}

type NotificationStreamContent struct {
	NotificationType string                             `json:"notificationType"`
	CurrentState     string                             `json:"currentState"`
	States           map[string]NotificationStreamState `json:"states"`
	// ClientEventsPayload holds analytics IDs, such as universe_id on an
	// experience invite. Its shape differs per notification type.
	ClientEventsPayload json.RawMessage `json:"clientEventsPayload"`
}

type NotificationStreamState struct {
	VisualItems NotificationStreamVisualItems `json:"visualItems"`
}

type NotificationStreamVisualItems struct {
	TextBody  []NotificationStreamTextBody  `json:"textBody"`
	Button    []NotificationStreamButton    `json:"button"`
	Thumbnail []NotificationStreamThumbnail `json:"thumbnail"`
}

type NotificationStreamText struct {
	Text string `json:"text"`
}

type NotificationStreamTextBody struct {
	Title   *NotificationStreamText    `json:"title"`
	Label   *NotificationStreamText    `json:"label"`
	Actions []NotificationStreamAction `json:"actions"`
}

type NotificationStreamButton struct {
	Label   NotificationStreamText     `json:"label"`
	Actions []NotificationStreamAction `json:"actions"`
}

// NotificationStreamAction is one step of a button. The website runs the steps
// in order: "notificationAPI" posts Path to the action endpoint, "dismiss" hides
// the entry, and "deeplink" opens a roblox:// or https:// Path.
type NotificationStreamAction struct {
	ActionType string `json:"actionType"`
	Path       string `json:"path"`
}

type NotificationStreamThumbnail struct {
	ID     string `json:"id"`
	IDType string `json:"idType"`
}

type NotificationStream struct {
	client *roblox.Client
}

func NewNotificationStream(client *roblox.Client) *NotificationStream {
	return &NotificationStream{client: client}
}

// Recent reads up to NotificationStreamPageSize entries, newest first, starting
// at startIndex.
func (service *NotificationStream) Recent(ctx context.Context, accountID, version int64, startIndex int) ([]NotificationStreamEntry, error) {
	endpointURL, _ := url.Parse(notificationStreamBaseURL + "get-recent")
	endpointURL.RawQuery = url.Values{
		"startIndex": {strconv.Itoa(startIndex)},
		"maxRows":    {strconv.Itoa(NotificationStreamPageSize)},
	}.Encode()
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: "notification-stream", AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodGet, URL: endpointURL, MaxResponseSize: notificationStreamResponseBytes,
		Retry: roblox.RetryPolicy{MaxAttempts: 3, RetryServerErrors: true, RetryRateLimit: true},
	})
	if err != nil {
		return nil, err
	}
	var entries []NotificationStreamEntry
	if err := json.Unmarshal(response.Body, &entries); err != nil {
		return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "notification-stream", Status: response.Status, Message: "Roblox returned invalid notification data.", Cause: err}
	}
	return entries, nil
}

// RunAction runs the server step of a notification button, such as
// FriendRequestAcceptedDismissAction. Running a dismiss action removes the entry
// from the stream for every client. Request shape from the Roblox web
// notification stream in Roblox-Web-Datamining, verified on 2026-11-09.
func (service *NotificationStream) RunAction(ctx context.Context, accountID, version int64, notificationID, action string) error {
	endpointURL, _ := url.Parse(notificationStreamBaseURL + "action/" + url.PathEscape(notificationID) + "/" + url.PathEscape(action))
	_, err := service.client.Do(ctx, roblox.Request{
		Endpoint: "notification-action", AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodPost, URL: endpointURL, Body: []byte("{}"),
		ContentType: "application/json", RequiresCSRF: true, MaxResponseSize: notificationStreamResponseBytes,
		Retry: roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
	})
	return err
}

// ClearUnread marks every notification as seen, as the Roblox website does
// when it opens the notification stream. The response shape was captured on
// 2026-10-15.
func (service *NotificationStream) ClearUnread(ctx context.Context, accountID, version int64) error {
	endpointURL, _ := url.Parse(notificationStreamBaseURL + "clear-unread")
	_, err := service.client.Do(ctx, roblox.Request{
		Endpoint: "notification-clear-unread", AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodPost, URL: endpointURL, Body: []byte("{}"),
		ContentType: "application/json", RequiresCSRF: true, MaxResponseSize: notificationStreamResponseBytes,
		Retry: roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
	})
	return err
}
