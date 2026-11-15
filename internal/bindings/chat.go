package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/appservice"
)

func (service *Service) GetChatConversations(ctx context.Context, accountID int64, cursor string) (appservice.ChatConversationPage, error) {
	if err := positiveID(accountID, "chat"); err != nil {
		return appservice.ChatConversationPage{}, err
	}
	return service.core.GetChatConversations(ctx, accountID, cursor)
}

func (service *Service) GetChatMessages(ctx context.Context, accountID int64, conversationID, cursor string) (appservice.ChatMessagePage, error) {
	if err := positiveID(accountID, "chat"); err != nil {
		return appservice.ChatMessagePage{}, err
	}
	return service.core.GetChatMessages(ctx, accountID, conversationID, cursor)
}

func (service *Service) MarkChatConversationRead(ctx context.Context, accountID int64, conversationID string) error {
	if err := positiveID(accountID, "chat"); err != nil {
		return err
	}
	return service.core.MarkChatConversationRead(ctx, accountID, conversationID)
}

func (service *Service) MarkAllChatConversationsRead(ctx context.Context, accountID int64) (int, error) {
	if err := positiveID(accountID, "chat"); err != nil {
		return 0, err
	}
	return service.core.MarkAllChatConversationsRead(ctx, accountID)
}

func (service *Service) CreateChatConversation(ctx context.Context, accountID int64, userIDs []int64, name string) (appservice.ChatConversationView, error) {
	if err := positiveID(accountID, "chat"); err != nil {
		return appservice.ChatConversationView{}, err
	}
	return service.core.CreateChatConversation(ctx, accountID, userIDs, name)
}

func (service *Service) AddChatMembers(ctx context.Context, accountID int64, conversationID string, userIDs []int64) (appservice.ChatConversationView, error) {
	if err := positiveID(accountID, "chat"); err != nil {
		return appservice.ChatConversationView{}, err
	}
	return service.core.AddChatMembers(ctx, accountID, conversationID, userIDs)
}

func (service *Service) RenameChatConversation(ctx context.Context, accountID int64, conversationID, name string) (appservice.ChatConversationView, error) {
	if err := positiveID(accountID, "chat"); err != nil {
		return appservice.ChatConversationView{}, err
	}
	return service.core.RenameChatConversation(ctx, accountID, conversationID, name)
}

func (service *Service) SendChatMessage(ctx context.Context, accountID int64, conversationID, text string) (appservice.ChatMessageView, error) {
	if err := positiveID(accountID, "chat"); err != nil {
		return appservice.ChatMessageView{}, err
	}
	return service.core.SendChatMessage(ctx, accountID, conversationID, text)
}
