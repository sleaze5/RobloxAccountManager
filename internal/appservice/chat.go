package appservice

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

const (
	maxChatCursorLength  = 4096
	maxChatMessageLength = 10000
	maxChatMarkAllPages  = 50
)

var (
	chatOperationID     atomic.Uint64
	chatUsernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,20}$`)
)

type ChatParticipant struct {
	UserID      int64  `json:"userId"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

type ChatConversationView struct {
	ID           string            `json:"id"`
	Group        bool              `json:"group"`
	Title        string            `json:"title"`
	ImageURL     string            `json:"imageUrl"`
	Preview      string            `json:"preview"`
	UnreadCount  int64             `json:"unreadCount"`
	UpdatedAtMs  int64             `json:"updatedAtMs"`
	Participants []ChatParticipant `json:"participants"`
}

type ChatMessageView struct {
	ID           string `json:"id"`
	SenderUserID int64  `json:"senderUserId"`
	Text         string `json:"text"`
	CreatedAtMs  int64  `json:"createdAtMs"`
	System       bool   `json:"system"`
	Moderated    bool   `json:"moderated"`
}

type ChatConversationPage struct {
	Conversations []ChatConversationView `json:"conversations"`
	NextCursor    string                 `json:"nextCursor"`
}

// ChatMessagePage lists messages oldest first; NextCursor loads older history.
type ChatMessagePage struct {
	Messages   []ChatMessageView `json:"messages"`
	NextCursor string            `json:"nextCursor"`
}

func (service *Service) GetChatConversations(ctx context.Context, accountID int64, cursor string) (ChatConversationPage, error) {
	if len(cursor) > maxChatCursorLength {
		return ChatConversationPage{}, chatInputError("This chat list has expired. Refresh chats.")
	}
	ctx, version, logger, done, err := service.beginChat(ctx, accountID, "list")
	if err != nil {
		return ChatConversationPage{}, err
	}
	defer done()
	selfID, err := service.chatSelfID(ctx, accountID)
	if err != nil {
		return ChatConversationPage{}, err
	}
	page, err := service.chat.Conversations(ctx, accountID, version, cursor)
	if err != nil {
		return ChatConversationPage{}, chatFailure(logger, err)
	}
	conversations := slices.DeleteFunc(page.Conversations, func(conversation robloxservices.ChatConversation) bool {
		return !realChatConversation(conversation.ID)
	})
	return ChatConversationPage{Conversations: service.chatConversationViews(ctx, logger, selfID, conversations), NextCursor: page.NextCursor}, nil
}

func (service *Service) GetChatMessages(ctx context.Context, accountID int64, conversationID, cursor string) (ChatMessagePage, error) {
	if err := validateChatConversationID(conversationID); err != nil {
		return ChatMessagePage{}, err
	}
	if len(cursor) > maxChatCursorLength {
		return ChatMessagePage{}, chatInputError("This chat history has expired. Reopen the chat.")
	}
	ctx, version, logger, done, err := service.beginChat(ctx, accountID, "messages")
	if err != nil {
		return ChatMessagePage{}, err
	}
	defer done()
	page, err := service.chat.Messages(ctx, accountID, version, conversationID, cursor)
	if err != nil {
		return ChatMessagePage{}, chatFailure(logger, err)
	}
	messages := make([]ChatMessageView, 0, len(page.Messages))
	for _, message := range page.Messages {
		if chatMessageVisible(message) {
			messages = append(messages, chatMessageView(message))
		}
	}
	slices.SortStableFunc(messages, func(left, right ChatMessageView) int {
		return cmp.Compare(left.CreatedAtMs, right.CreatedAtMs)
	})
	return ChatMessagePage{Messages: messages, NextCursor: page.NextCursor}, nil
}

func (service *Service) MarkChatConversationRead(ctx context.Context, accountID int64, conversationID string) error {
	if err := validateChatConversationID(conversationID); err != nil {
		return err
	}
	ctx, version, logger, done, err := service.beginChat(ctx, accountID, "mark-read")
	if err != nil {
		return err
	}
	defer done()
	return service.markChatRead(ctx, logger, accountID, version, []string{conversationID})
}

// MarkAllChatConversationsRead marks every conversation with unread messages and
// returns how many were marked. Roblox's global unread metadata can report zero
// while conversations still have unread messages, so every page is checked.
func (service *Service) MarkAllChatConversationsRead(ctx context.Context, accountID int64) (int, error) {
	ctx, version, logger, done, err := service.beginChat(ctx, accountID, "mark-all-read")
	if err != nil {
		return 0, err
	}
	defer done()
	ids, seen, cursor := []string{}, map[string]bool{}, ""
	for range maxChatMarkAllPages {
		page, err := service.chat.Conversations(ctx, accountID, version, cursor)
		if err != nil {
			return 0, chatFailure(logger, err)
		}
		for _, conversation := range page.Conversations {
			if conversation.UnreadMessageCount > 0 && realChatConversation(conversation.ID) && !seen[conversation.ID] {
				seen[conversation.ID] = true
				ids = append(ids, conversation.ID)
			}
		}
		if cursor = page.NextCursor; cursor == "" {
			break
		}
	}
	for chunk := range slices.Chunk(ids, robloxservices.ChatPageSize) {
		if err := service.markChatRead(ctx, logger, accountID, version, chunk); err != nil {
			return 0, err
		}
	}
	logger.Debug("chat conversations marked read", "count", len(ids))
	return len(ids), nil
}

func (service *Service) CreateChatConversation(ctx context.Context, accountID int64, username string) (ChatConversationView, error) {
	username = strings.TrimPrefix(strings.TrimSpace(username), "@")
	if !chatUsernamePattern.MatchString(username) {
		return ChatConversationView{}, chatInputError("Enter a Roblox username of 3 to 20 letters, numbers, or underscores.")
	}
	ctx, version, logger, done, err := service.beginChat(ctx, accountID, "create")
	if err != nil {
		return ChatConversationView{}, err
	}
	defer done()
	selfID, err := service.chatSelfID(ctx, accountID)
	if err != nil {
		return ChatConversationView{}, err
	}
	userID, err := service.users.UserIDFromUsername(ctx, username)
	if err != nil {
		return ChatConversationView{}, chatFailure(logger, err)
	}
	if userID == selfID {
		return ChatConversationView{}, chatInputError("Enter another user's username.")
	}
	conversation, err := service.chat.CreateDirect(ctx, accountID, version, userID)
	if err != nil {
		return ChatConversationView{}, chatFailure(logger, err)
	}
	logger.Info("chat conversation opened")
	return service.chatConversationViews(ctx, logger, selfID, []robloxservices.ChatConversation{conversation})[0], nil
}

func (service *Service) SendChatMessage(ctx context.Context, accountID int64, conversationID, text string) (ChatMessageView, error) {
	if err := validateChatConversationID(conversationID); err != nil {
		return ChatMessageView{}, err
	}
	text = strings.TrimSpace(text)
	if text == "" || !utf8.ValidString(text) {
		return ChatMessageView{}, chatInputError("Enter a message.")
	}
	if utf8.RuneCountInString(text) > maxChatMessageLength {
		return ChatMessageView{}, chatInputError("This message is too long.")
	}
	ctx, version, logger, done, err := service.beginChat(ctx, accountID, "send")
	if err != nil {
		return ChatMessageView{}, err
	}
	defer done()
	message, err := service.chat.Send(ctx, accountID, version, conversationID, text)
	if err != nil {
		var remote *roblox.Error
		if errors.As(err, &remote) && strings.Contains(strings.ToLower(remote.Message), "too long") {
			return ChatMessageView{}, chatInputError("This message is too long.")
		}
		if errors.As(err, &remote) && remote.Status == 409 {
			return ChatMessageView{}, chatInputError("This chat changed. Refresh it and try again.")
		}
		return ChatMessageView{}, chatFailure(logger, err)
	}
	view := chatMessageView(message)
	if view.Text == "" {
		view.Text = text
	}
	if view.CreatedAtMs == 0 {
		view.CreatedAtMs = time.Now().UnixMilli()
	}
	logger.Debug("chat message sent", "moderated", view.Moderated)
	return view, nil
}

func (service *Service) beginChat(parent context.Context, accountID int64, operation string) (context.Context, int64, *slog.Logger, func(), error) {
	ctx, version, done, err := service.beginAccountSession(parent, accountID, "chat")
	if err != nil {
		return nil, 0, nil, nil, err
	}
	logger := service.logs.Module("application.chat").With("operation_id", fmt.Sprintf("chat-%d", chatOperationID.Add(1)), "operation", operation)
	return ctx, version, logger, done, nil
}

func (service *Service) chatSelfID(ctx context.Context, accountID int64) (int64, error) {
	account, err := service.repo.GetView(ctx, accountID)
	if err != nil {
		return 0, mapRepositoryError(err)
	}
	return account.RobloxUserID, nil
}

func (service *Service) markChatRead(ctx context.Context, logger *slog.Logger, accountID, version int64, ids []string) error {
	results, err := service.chat.MarkRead(ctx, accountID, version, ids)
	if err != nil {
		return chatFailure(logger, err)
	}
	for _, result := range results {
		if result.Status != "" && !strings.EqualFold(result.Status, "success") {
			logger.Warn("chat mark-read returned an unexpected status", "status", result.Status)
		}
	}
	return nil
}

func (service *Service) chatConversationViews(ctx context.Context, logger *slog.Logger, selfID int64, conversations []robloxservices.ChatConversation) []ChatConversationView {
	views := make([]ChatConversationView, 0, len(conversations))
	imageUsers := []int64{}
	for _, conversation := range conversations {
		view := chatConversationView(selfID, conversation)
		if !view.Group && len(view.Participants) > 0 {
			imageUsers = append(imageUsers, view.Participants[0].UserID)
		}
		views = append(views, view)
	}
	if len(imageUsers) == 0 {
		return views
	}
	headshots, err := service.avatars.AvatarHeadshots(ctx, imageUsers)
	if err != nil {
		logger.Warn("chat avatars unavailable", "error", err)
		return views
	}
	images := make(map[int64]string, len(headshots))
	for _, headshot := range headshots {
		images[headshot.RobloxUserID] = headshot.ImageURL
	}
	for index := range views {
		if !views[index].Group && len(views[index].Participants) > 0 {
			views[index].ImageURL = images[views[index].Participants[0].UserID]
		}
	}
	return views
}

// chatConversationView lists the other participants first so a direct chat's
// partner is always Participants[0].
func chatConversationView(selfID int64, conversation robloxservices.ChatConversation) ChatConversationView {
	view := ChatConversationView{
		ID:           conversation.ID,
		Group:        conversation.Type == "group",
		UnreadCount:  max(conversation.UnreadMessageCount, 0),
		UpdatedAtMs:  chatTime(conversation.UpdatedAt),
		Participants: []ChatParticipant{},
	}
	var self *ChatParticipant
	for _, userID := range conversation.ParticipantUserIDs {
		user := conversation.UserData[strconv.FormatInt(userID, 10)]
		participant := ChatParticipant{UserID: userID, Username: user.Name, DisplayName: cmp.Or(user.DisplayName, user.Name)}
		if userID == selfID {
			self = &participant
			continue
		}
		view.Participants = append(view.Participants, participant)
	}
	names := make([]string, 0, len(view.Participants))
	for _, participant := range view.Participants {
		if participant.DisplayName != "" {
			names = append(names, participant.DisplayName)
		}
	}
	if view.Group {
		view.Title = cmp.Or(strings.TrimSpace(conversation.Name), strings.Join(names, ", "), "Group chat")
	} else {
		view.Title = cmp.Or(strings.Join(names, ", "), "Unknown user")
	}
	if self != nil {
		view.Participants = append(view.Participants, *self)
	}
	if preview := chatPreviewMessage(conversation); preview != nil {
		view.Preview = chatPreviewText(*preview)
		if createdAt := chatTime(preview.CreatedAt); createdAt > 0 {
			view.UpdatedAtMs = createdAt
		}
	}
	return view
}

func chatPreviewMessage(conversation robloxservices.ChatConversation) *robloxservices.ChatMessage {
	if conversation.PreviewMessage != nil {
		return conversation.PreviewMessage
	}
	var latest *robloxservices.ChatMessage
	for index := range conversation.Messages {
		message := &conversation.Messages[index]
		if message.IsPreviewable && chatMessageVisible(*message) && (latest == nil || chatTime(message.CreatedAt) > chatTime(latest.CreatedAt)) {
			latest = message
		}
	}
	return latest
}

func chatPreviewText(message robloxservices.ChatMessage) string {
	if !chatMessageVisible(message) {
		return ""
	}
	return chatMessageText(message)
}

// chatMessageVisible drops moderated ("hidden") and placeholder ("invalid")
// history entries. Send responses report "invalid" for messages that later
// appear as visible, so they are not filtered.
func chatMessageVisible(message robloxservices.ChatMessage) bool {
	return message.Visibility != "hidden" && message.Visibility != "invalid"
}

func chatMessageView(message robloxservices.ChatMessage) ChatMessageView {
	text := chatPiecesText(message.Pieces)
	if len(message.Pieces) == 0 {
		text = chatMessageText(message)
	}
	return ChatMessageView{
		ID:           message.ID,
		SenderUserID: message.SenderUserID,
		Text:         text,
		CreatedAtMs:  chatTime(message.CreatedAt),
		System:       message.Type == "system",
		Moderated:    strings.EqualFold(message.Status, "moderated"),
	}
}

func chatMessageText(message robloxservices.ChatMessage) string {
	if message.PreviewContent != nil {
		return *message.PreviewContent
	}
	if message.Content != nil {
		return *message.Content
	}
	return chatPiecesText(message.Pieces)
}

func chatPiecesText(pieces []robloxservices.ChatMessagePiece) string {
	var text strings.Builder
	for _, piece := range pieces {
		text.WriteString(piece.Content)
	}
	return text.String()
}

func chatTime(value string) int64 {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return 0
	}
	return parsed.UnixMilli()
}

// realChatConversation excludes the friend placeholders Roblox lists before a
// conversation with that friend exists; they have a null or "friends-" ID.
func realChatConversation(id string) bool {
	return id != "" && !strings.HasPrefix(id, "friends-")
}

func validateChatConversationID(id string) error {
	if !realChatConversation(id) || len(id) > 256 || !utf8.ValidString(id) || strings.ContainsFunc(id, func(r rune) bool { return r <= ' ' || r == 0x7f }) {
		return chatInputError("This chat is unavailable. Refresh chats.")
	}
	return nil
}

func chatFailure(logger *slog.Logger, err error) error {
	var remote *roblox.Error
	if !errors.As(err, &remote) {
		logger.Warn("chat operation failed", "error", err)
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "chat", Message: "The chat request failed. Try again.", Cause: err}
	}
	logger.Warn("chat operation failed", "endpoint", remote.Endpoint, "error_kind", remote.Kind, "status", remote.Status, "roblox_code", remote.RobloxCode)
	return err
}

func chatInputError(message string) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "chat", Message: message}
}
