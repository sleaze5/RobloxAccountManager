package appservice

import (
	"context"

	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// GetInventoryPage reads one page of what the user owns in the category, as
// the account sees it, with item images and catalog details. A missing image
// or catalog detail leaves the item without it.
func (service *Service) GetInventoryPage(ctx context.Context, accountID, userID int64, category robloxservices.InventoryCategory, cursor string) (robloxservices.InventoryPage, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "inventory")
	if err != nil {
		return robloxservices.InventoryPage{}, err
	}
	defer done()
	logger := service.logs.Module("application.inventory").With("operation", "inventory", "account_id", accountID, "category", category)
	page, err := service.users.InventoryPage(ctx, accountID, version, userID, category, cursor)
	if err != nil {
		logger.WarnContext(ctx, "inventory page unavailable", "error", err)
		return robloxservices.InventoryPage{}, err
	}
	if err := service.users.AddCatalogDetails(ctx, accountID, version, page.Items); err != nil {
		logger.WarnContext(ctx, "inventory catalog details unavailable", "error", err)
	}
	targets := make([]robloxservices.ThumbnailTarget, len(page.Items))
	for index, item := range page.Items {
		targets[index] = item.ThumbnailTarget()
	}
	images, err := service.thumbnails.Images(ctx, targets)
	if err != nil {
		logger.WarnContext(ctx, "inventory images unavailable", "error", err)
	}
	for index := range page.Items {
		page.Items[index].ImageURL = images[targets[index]]
	}
	return page, nil
}
