package appservice

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/logging"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

const (
	maxChatCursorLength  = 4096
	maxChatMessageLength = 10000
	maxChatMarkAllPages  = 50
	// Roblox limits a group chat to 6 members, including the account, and its
	// name to 150 characters. Values from platform-chat-api/v1/metadata
	// (numberOfMembersForPartyChrome and maxConversationTitleLength) on 2026-11-09.
	maxChatGroupMembers    = 6
	maxChatGroupNameLength = 150
)

var chatOperationID atomic.Uint64

type ChatParticipant struct {
	UserID      int64  `json:"userId"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	ImageURL    string `json:"imageUrl"`
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
	Conversations      []ChatConversationView `json:"conversations"`
	NextCursor         string                 `json:"nextCursor"`
	GroupMemberLimit   int                    `json:"groupMemberLimit"`
	GroupNameMaxLength int                    `json:"groupNameMaxLength"`
}

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
		return ChatConversationPage{}, chatFailure(ctx, logger, err)
	}
	conversations := slices.DeleteFunc(page.Conversations, func(conversation robloxservices.ChatConversation) bool {
		return !realChatConversation(conversation.ID)
	})
	return ChatConversationPage{
		Conversations: service.chatConversationViews(ctx, logger, selfID, conversations), NextCursor: page.NextCursor,
		GroupMemberLimit: maxChatGroupMembers, GroupNameMaxLength: maxChatGroupNameLength,
	}, nil
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
		return ChatMessagePage{}, chatFailure(ctx, logger, err)
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

func (service *Service) MarkAllChatConversationsRead(ctx context.Context, accountID int64) (int, error) {
	ctx, version, logger, done, err := service.beginChat(ctx, accountID, "mark-all-read")
	if err != nil {
		return 0, err
	}
	defer done()
	conversationIDs, seen, cursor := []string{}, map[string]bool{}, ""
	for range maxChatMarkAllPages {
		page, err := service.chat.Conversations(ctx, accountID, version, cursor)
		if err != nil {
			return 0, chatFailure(ctx, logger, err)
		}
		for _, conversation := range page.Conversations {
			if conversation.UnreadMessageCount > 0 && realChatConversation(conversation.ID) && !seen[conversation.ID] {
				seen[conversation.ID] = true
				conversationIDs = append(conversationIDs, conversation.ID)
			}
		}
		if cursor = page.NextCursor; cursor == "" {
			break
		}
	}
	for chunk := range slices.Chunk(conversationIDs, robloxservices.ChatPageSize) {
		if err := service.markChatRead(ctx, logger, accountID, version, chunk); err != nil {
			return 0, err
		}
	}
	logger.DebugContext(ctx, "chat conversations marked read", "count", len(conversationIDs))
	return len(conversationIDs), nil
}

// CreateChatConversation opens a direct chat with one user, or creates a group
// chat when several users are chosen or the chat has a name.
func (service *Service) CreateChatConversation(ctx context.Context, accountID int64, userIDs []int64, name string) (ChatConversationView, error) {
	userIDs, name, err := chatMembersInput(userIDs, name, maxChatGroupMembers-1)
	if err != nil {
		return ChatConversationView{}, err
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
	if slices.Contains(userIDs, selfID) {
		return ChatConversationView{}, chatInputError("Choose other users. This account is already in the chat.")
	}
	var conversation robloxservices.ChatConversation
	if len(userIDs) == 1 && name == "" {
		conversation, err = service.chat.CreateDirect(ctx, accountID, version, userIDs[0])
	} else {
		conversation, err = service.chat.CreateGroup(ctx, accountID, version, name, userIDs)
	}
	if err == nil {
		err = chatConversationStatusError(conversation.Status, "Roblox did not create this chat. Try again.")
	}
	if err != nil {
		return ChatConversationView{}, chatFailure(ctx, logger, err)
	}
	logger.InfoContext(ctx, "chat conversation opened", "group", conversation.Type == "group", "members", len(userIDs))
	return service.chatConversationViews(ctx, logger, selfID, []robloxservices.ChatConversation{conversation})[0], nil
}

// AddChatMembers adds users to a group chat and returns the updated chat.
func (service *Service) AddChatMembers(ctx context.Context, accountID int64, conversationID string, userIDs []int64) (ChatConversationView, error) {
	if err := validateChatConversationID(conversationID); err != nil {
		return ChatConversationView{}, err
	}
	userIDs, _, err := chatMembersInput(userIDs, "", maxChatGroupMembers-2)
	if err != nil {
		return ChatConversationView{}, err
	}
	ctx, version, logger, done, err := service.beginChat(ctx, accountID, "add-members")
	if err != nil {
		return ChatConversationView{}, err
	}
	defer done()
	selfID, err := service.chatSelfID(ctx, accountID)
	if err != nil {
		return ChatConversationView{}, err
	}
	current, err := service.chat.Conversation(ctx, accountID, version, conversationID)
	if err != nil {
		return ChatConversationView{}, chatFailure(ctx, logger, err)
	}
	if current.Type != "group" {
		return ChatConversationView{}, chatInputError("Members can only be added to group chats.")
	}
	userIDs = slices.DeleteFunc(userIDs, func(userID int64) bool { return slices.Contains(current.ParticipantUserIDs, userID) })
	if len(userIDs) == 0 {
		return ChatConversationView{}, chatInputError("Everyone you chose is already in this chat.")
	}
	if len(current.ParticipantUserIDs)+len(userIDs) > maxChatGroupMembers {
		return ChatConversationView{}, chatInputError(fmt.Sprintf("A group chat can have up to %d members.", maxChatGroupMembers))
	}
	if err := service.chat.AddUsers(ctx, accountID, version, conversationID, userIDs); err != nil {
		return ChatConversationView{}, chatFailure(ctx, logger, err)
	}
	updated, err := service.chat.Conversation(ctx, accountID, version, conversationID)
	if err != nil {
		return ChatConversationView{}, chatFailure(ctx, logger, err)
	}
	logger.InfoContext(ctx, "chat members added", "count", len(userIDs))
	return service.chatConversationViews(ctx, logger, selfID, []robloxservices.ChatConversation{updated})[0], nil
}

func (service *Service) RenameChatConversation(ctx context.Context, accountID int64, conversationID, name string) (ChatConversationView, error) {
	if err := validateChatConversationID(conversationID); err != nil {
		return ChatConversationView{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" || !utf8.ValidString(name) || utf8.RuneCountInString(name) > maxChatGroupNameLength {
		return ChatConversationView{}, chatInputError(fmt.Sprintf("Enter a name of up to %d characters.", maxChatGroupNameLength))
	}
	ctx, version, logger, done, err := service.beginChat(ctx, accountID, "rename")
	if err != nil {
		return ChatConversationView{}, err
	}
	defer done()
	selfID, err := service.chatSelfID(ctx, accountID)
	if err != nil {
		return ChatConversationView{}, err
	}
	conversation, err := service.chat.Rename(ctx, accountID, version, conversationID, name)
	if err == nil {
		err = chatConversationStatusError(conversation.Status, "Roblox did not rename this chat. Try again.")
	}
	if err != nil {
		return ChatConversationView{}, chatFailure(ctx, logger, err)
	}
	if len(conversation.ParticipantUserIDs) == 0 {
		if conversation, err = service.chat.Conversation(ctx, accountID, version, conversationID); err != nil {
			return ChatConversationView{}, chatFailure(ctx, logger, err)
		}
	}
	logger.InfoContext(ctx, "chat renamed")
	return service.chatConversationViews(ctx, logger, selfID, []robloxservices.ChatConversation{conversation})[0], nil
}

// chatMembersInput checks the users and the optional group name of a chat.
func chatMembersInput(userIDs []int64, name string, maxUsers int) ([]int64, string, error) {
	userIDs = slices.Compact(slices.Sorted(slices.Values(userIDs)))
	if len(userIDs) == 0 || len(userIDs) > maxUsers || slices.ContainsFunc(userIDs, func(userID int64) bool { return userID <= 0 }) {
		return nil, "", chatInputError(fmt.Sprintf("Choose 1 to %d people.", maxUsers))
	}
	name = strings.TrimSpace(name)
	if !utf8.ValidString(name) || utf8.RuneCountInString(name) > maxChatGroupNameLength {
		return nil, "", chatInputError(fmt.Sprintf("Shorten the group name to %d characters or fewer.", maxChatGroupNameLength))
	}
	return userIDs, name, nil
}

// chatConversationStatusError reads the status that Roblox returns with a
// created or renamed chat. An empty status means Roblox accepted it.
func chatConversationStatusError(status, failed string) error {
	switch strings.ToLower(status) {
	case "", "success":
		return nil
	case "moderated":
		return chatInputError("Roblox blocked this chat name. Choose another name.")
	default:
		return chatInputError(failed)
	}
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
		if errors.As(err, &remote) && remote.Status == http.StatusConflict {
			return ChatMessageView{}, chatInputError("This chat changed. Refresh it and try again.")
		}
		return ChatMessageView{}, chatFailure(ctx, logger, err)
	}
	view := chatMessageView(message)
	if view.Text == "" {
		view.Text = text
	}
	if view.CreatedAtMs == 0 {
		view.CreatedAtMs = time.Now().UnixMilli()
	}
	logger.DebugContext(ctx, "chat message sent", "moderated", view.Moderated)
	return view, nil
}

func (service *Service) beginChat(parent context.Context, accountID int64, operation string) (context.Context, int64, *slog.Logger, func(), error) {
	ctx, version, done, err := service.beginAccountSession(parent, accountID, "chat")
	if err != nil {
		return nil, 0, nil, nil, err
	}
	ctx = logging.WithOperation(ctx, fmt.Sprintf("chat-%d", chatOperationID.Add(1)))
	logger := service.logs.Module("application.chat").With("operation", operation)
	return ctx, version, logger, done, nil
}

func (service *Service) chatSelfID(ctx context.Context, accountID int64) (int64, error) {
	account, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return 0, mapRepositoryError(err)
	}
	return account.RobloxUserID, nil
}

func (service *Service) markChatRead(ctx context.Context, logger *slog.Logger, accountID, version int64, conversationIDs []string) error {
	results, err := service.chat.MarkRead(ctx, accountID, version, conversationIDs)
	if err != nil {
		return chatFailure(ctx, logger, err)
	}
	for _, conversationStatus := range results {
		if conversationStatus.Status != "" && !strings.EqualFold(conversationStatus.Status, "success") {
			logger.WarnContext(ctx, "chat mark-read returned an unexpected status", "status", conversationStatus.Status)
		}
	}
	return nil
}

func (service *Service) chatConversationViews(ctx context.Context, logger *slog.Logger, selfID int64, conversations []robloxservices.ChatConversation) []ChatConversationView {
	views := make([]ChatConversationView, 0, len(conversations))
	imageUsers := []int64{}
	for _, conversation := range conversations {
		view := chatConversationView(selfID, conversation)
		for _, participant := range view.Participants {
			if !slices.Contains(imageUsers, participant.UserID) {
				imageUsers = append(imageUsers, participant.UserID)
			}
		}
		views = append(views, view)
	}
	if len(imageUsers) == 0 {
		return views
	}
	images, err := service.headshotImages(ctx, imageUsers)
	if err != nil {
		logger.WarnContext(ctx, "chat avatars unavailable", "error", err)
		return views
	}
	for index := range views {
		view := &views[index]
		for participantIndex := range view.Participants {
			participant := &view.Participants[participantIndex]
			participant.ImageURL = images[participant.UserID]
		}
		if !view.Group && len(view.Participants) > 0 {
			view.ImageURL = view.Participants[0].ImageURL
		}
	}
	return views
}

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

func chatTime(timestamp string) int64 {
	parsed, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		return 0
	}
	return parsed.UnixMilli()
}

func realChatConversation(conversationID string) bool {
	return conversationID != "" && !strings.HasPrefix(conversationID, "friends-")
}

func validateChatConversationID(conversationID string) error {
	// 0x7f is the DEL control character.
	if !realChatConversation(conversationID) || len(conversationID) > 256 || !utf8.ValidString(conversationID) || strings.ContainsFunc(conversationID, func(character rune) bool { return character <= ' ' || character == 0x7f }) {
		return chatInputError("This chat is unavailable. Refresh chats.")
	}
	return nil
}

func chatFailure(ctx context.Context, logger *slog.Logger, err error) error {
	return operationFailure(ctx, logger, err, "chat", "The chat request failed. Try again.")
}

func chatInputError(message string) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "chat", Message: message}
}
