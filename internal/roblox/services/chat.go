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
	chatBaseURL      = "https://apis.roblox.com/platform-chat-api/v1/"
	ChatPageSize     = 20
	chatResponseSize = 4 << 20
)

type ChatUser struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

type ChatMessagePiece struct {
	Content string `json:"content"`
}

type ChatMessage struct {
	ID             string             `json:"id"`
	SenderUserID   int64              `json:"sender_user_id"`
	CreatedAt      string             `json:"created_at"`
	Content        *string            `json:"content"`
	PreviewContent *string            `json:"preview_content"`
	Pieces         []ChatMessagePiece `json:"pieces"`
	Type           string             `json:"type"`
	Status         string             `json:"status"`
	Visibility     string             `json:"visibility"`
	IsPreviewable  bool               `json:"is_previewable"`
}

type ChatConversation struct {
	ID                 string              `json:"id"`
	Type               string              `json:"type"`
	Name               string              `json:"name"`
	ParticipantUserIDs []int64             `json:"participant_user_ids"`
	UserData           map[string]ChatUser `json:"user_data"`
	Messages           []ChatMessage       `json:"messages"`
	PreviewMessage     *ChatMessage        `json:"preview_message"`
	UnreadMessageCount int64               `json:"unread_message_count"`
	UpdatedAt          string              `json:"updated_at"`
}

type ChatConversationPage struct {
	Conversations []ChatConversation `json:"conversations"`
	NextCursor    string             `json:"next_cursor"`
}

type ChatMessagePage struct {
	Messages   []ChatMessage `json:"messages"`
	NextCursor string        `json:"next_cursor"`
}

type ChatMarkResult struct {
	ConversationID string `json:"conversation_id"`
	Status         string `json:"status"`
}

type Chat struct {
	client *roblox.Client
}

func NewChat(client *roblox.Client) *Chat {
	return &Chat{client: client}
}

func (service *Chat) Conversations(ctx context.Context, accountID, version int64, cursor string) (ChatConversationPage, error) {
	query := url.Values{"include_user_data": {"true"}, "pageSize": {strconv.Itoa(ChatPageSize)}}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	var page ChatConversationPage
	err := service.get(ctx, accountID, version, "chat-conversations", "get-user-conversations", query, &page)
	return page, err
}

func (service *Chat) Messages(ctx context.Context, accountID, version int64, conversationID, cursor string) (ChatMessagePage, error) {
	query := url.Values{"conversation_id": {conversationID}}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	var page ChatMessagePage
	err := service.get(ctx, accountID, version, "chat-messages", "get-conversation-messages", query, &page)
	return page, err
}

func (service *Chat) MarkRead(ctx context.Context, accountID, version int64, conversationIDs []string) ([]ChatMarkResult, error) {
	var result struct {
		Results []ChatMarkResult `json:"results"`
	}
	err := service.post(ctx, accountID, version, "chat-mark-read", "mark-conversations", true, map[string]any{"conversation_ids": conversationIDs}, &result)
	return result.Results, err
}

func (service *Chat) CreateDirect(ctx context.Context, accountID, version, userID int64) (ChatConversation, error) {
	var result struct {
		Conversations []ChatConversation `json:"conversations"`
	}
	body := map[string]any{
		"conversations":     []map[string]any{{"type": "one_to_one", "participant_user_ids": []int64{userID}}},
		"include_user_data": true,
	}
	if err := service.post(ctx, accountID, version, "chat-create", "create-conversations", false, body, &result); err != nil {
		return ChatConversation{}, err
	}
	if len(result.Conversations) == 0 || result.Conversations[0].ID == "" {
		return ChatConversation{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "chat-create", Message: "Roblox did not return the new chat."}
	}
	return result.Conversations[0], nil
}

func (service *Chat) Send(ctx context.Context, accountID, version int64, conversationID, content string) (ChatMessage, error) {
	var result ChatMessagePage
	body := map[string]any{"conversation_id": conversationID, "messages": []map[string]string{{"content": content}}}
	if err := service.post(ctx, accountID, version, "chat-send", "send-messages", false, body, &result); err != nil {
		return ChatMessage{}, err
	}
	if len(result.Messages) == 0 {
		return ChatMessage{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "chat-send", Message: "Roblox did not confirm the sent message. Refresh the chat to check."}
	}
	return result.Messages[0], nil
}

func (service *Chat) get(ctx context.Context, accountID, version int64, endpoint, path string, query url.Values, target any) error {
	uri, _ := url.Parse(chatBaseURL + path)
	uri.RawQuery = query.Encode()
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: endpoint, AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodGet, URL: uri, MaxResponseSize: chatResponseSize,
		Retry: roblox.RetryPolicy{MaxAttempts: 3, RetryServerErrors: true, RetryRateLimit: true},
	})
	if err != nil {
		return err
	}
	return decodeChat(endpoint, response, target)
}

func (service *Chat) post(ctx context.Context, accountID, version int64, endpoint, path string, idempotent bool, payload, target any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "The chat request could not be encoded.", Cause: err}
	}
	uri, _ := url.Parse(chatBaseURL + path)
	retry := roblox.RetryPolicy{MaxAttempts: 1}
	if idempotent {
		retry = roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true}
	}
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: endpoint, AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodPost, URL: uri, Body: body,
		ContentType: "application/json", RequiresCSRF: true, MaxResponseSize: chatResponseSize, Retry: retry,
	})
	if err != nil {
		return err
	}
	return decodeChat(endpoint, response, target)
}

func decodeChat(endpoint string, response *roblox.Response, target any) error {
	if err := json.Unmarshal(response.Body, target); err != nil {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Status: response.Status, Message: "Roblox returned invalid chat data.", Cause: err}
	}
	return nil
}
