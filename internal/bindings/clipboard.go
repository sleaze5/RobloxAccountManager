package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/appservice"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func (service *Service) CopyAccountFields(ctx context.Context, accountIDs []int64, field appservice.AccountCopyField) error {
	if err := accountSelection(accountIDs, "account-copy"); err != nil {
		return err
	}
	app := application.Get()
	if app == nil {
		return clipboardUnavailable()
	}
	return service.core.CopyAccountFields(ctx, accountIDs, field, app.Clipboard.SetText)
}

func clipboardUnavailable() error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "account-clipboard", Message: "The clipboard is unavailable. Restart the app and try again."}
}
