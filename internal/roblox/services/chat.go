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
	// Status reports whether a created or renamed conversation was accepted:
	// "success", "moderated" when Roblox blocked the name, or "failed".
	Status string `json:"status"`
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
	var payload struct {
		Results []ChatMarkResult `json:"results"`
	}
	err := service.post(ctx, accountID, version, "chat-mark-read", "mark-conversations", true, map[string]any{"conversation_ids": conversationIDs}, &payload)
	return payload.Results, err
}

func (service *Chat) CreateDirect(ctx context.Context, accountID, version, userID int64) (ChatConversation, error) {
	return service.create(ctx, accountID, version, map[string]any{"type": "one_to_one", "participant_user_ids": []int64{userID}})
}

// CreateGroup creates a group conversation. Request shape from the Roblox web
// chat client in Roblox-Web-Datamining.
func (service *Chat) CreateGroup(ctx context.Context, accountID, version int64, name string, userIDs []int64) (ChatConversation, error) {
	return service.create(ctx, accountID, version, map[string]any{"type": "group", "name": name, "participant_user_ids": userIDs})
}

func (service *Chat) create(ctx context.Context, accountID, version int64, conversation map[string]any) (ChatConversation, error) {
	var payload struct {
		Conversations []ChatConversation `json:"conversations"`
	}
	body := map[string]any{"conversations": []map[string]any{conversation}, "include_user_data": true}
	if err := service.post(ctx, accountID, version, "chat-create", "create-conversations", false, body, &payload); err != nil {
		return ChatConversation{}, err
	}
	return firstChatConversation("chat-create", payload.Conversations, "Roblox did not return the new chat.")
}

// Conversation reads one conversation by ID, with its participants.
func (service *Chat) Conversation(ctx context.Context, accountID, version int64, conversationID string) (ChatConversation, error) {
	var payload struct {
		Conversations []ChatConversation `json:"conversations"`
	}
	body := map[string]any{"ids": []string{conversationID}, "include_participants": true, "include_user_data": true}
	if err := service.post(ctx, accountID, version, "chat-conversation", "get-conversations", true, body, &payload); err != nil {
		return ChatConversation{}, err
	}
	return firstChatConversation("chat-conversation", payload.Conversations, "Roblox did not return this chat. Refresh chats.")
}

func (service *Chat) AddUsers(ctx context.Context, accountID, version int64, conversationID string, userIDs []int64) error {
	var payload any
	return service.post(ctx, accountID, version, "chat-add-users", "add-users", false, map[string]any{"conversation_id": conversationID, "user_ids": userIDs}, &payload)
}

func (service *Chat) Rename(ctx context.Context, accountID, version int64, conversationID, name string) (ChatConversation, error) {
	var payload struct {
		Conversations []ChatConversation `json:"conversations"`
	}
	body := map[string]any{"conversations": []map[string]any{{"id": conversationID, "name": name}}}
	if err := service.post(ctx, accountID, version, "chat-rename", "update-conversations", true, body, &payload); err != nil {
		return ChatConversation{}, err
	}
	return firstChatConversation("chat-rename", payload.Conversations, "Roblox did not confirm the new name. Refresh chats to check.")
}

func firstChatConversation(endpoint string, conversations []ChatConversation, missing string) (ChatConversation, error) {
	if len(conversations) == 0 || conversations[0].ID == "" {
		return ChatConversation{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: missing}
	}
	return conversations[0], nil
}

func (service *Chat) Send(ctx context.Context, accountID, version int64, conversationID, content string) (ChatMessage, error) {
	var payload ChatMessagePage
	body := map[string]any{"conversation_id": conversationID, "messages": []map[string]string{{"content": content}}}
	if err := service.post(ctx, accountID, version, "chat-send", "send-messages", false, body, &payload); err != nil {
		return ChatMessage{}, err
	}
	if len(payload.Messages) == 0 {
		return ChatMessage{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "chat-send", Message: "Roblox did not confirm the sent message. Refresh the chat to check."}
	}
	return payload.Messages[0], nil
}

func (service *Chat) get(ctx context.Context, accountID, version int64, endpoint, path string, query url.Values, target any) error {
	endpointURL, _ := url.Parse(chatBaseURL + path)
	endpointURL.RawQuery = query.Encode()
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: endpoint, AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodGet, URL: endpointURL, MaxResponseSize: chatResponseSize,
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
	endpointURL, _ := url.Parse(chatBaseURL + path)
	var retry roblox.RetryPolicy
	if idempotent {
		retry = roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true}
	}
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: endpoint, AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodPost, URL: endpointURL, Body: body,
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
