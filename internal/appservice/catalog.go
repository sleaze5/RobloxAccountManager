package appservice

import (
	"context"

	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// SearchCatalog reads one page of the Marketplace items that match the
// search, as the account sees them, with item images. A missing image leaves
// the item without it.
func (service *Service) SearchCatalog(ctx context.Context, accountID int64, search robloxservices.CatalogSearch, cursor string) (robloxservices.CatalogPage, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "catalog-search")
	if err != nil {
		return robloxservices.CatalogPage{}, err
	}
	defer done()
	logger := service.logs.Module("application.catalog").With("operation", "catalog-search", "account_id", accountID, "category", search.Category)
	page, err := service.users.SearchCatalog(ctx, accountID, version, search, cursor)
	if err != nil {
		logger.WarnContext(ctx, "catalog search failed", "error", err)
		return robloxservices.CatalogPage{}, err
	}
	items := make([]robloxservices.InventoryItem, len(page.Items))
	for index, categorizedItem := range page.Items {
		items[index] = categorizedItem.Item
	}
	service.addItemImages(ctx, items, logger)
	for index := range page.Items {
		page.Items[index].Item = items[index]
	}
	return page, nil
}
