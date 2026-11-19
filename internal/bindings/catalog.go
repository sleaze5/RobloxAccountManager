package bindings

import (
	"context"

	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

func (service *Service) SearchCatalog(ctx context.Context, accountID int64, search robloxservices.CatalogSearch, cursor string) (robloxservices.CatalogPage, error) {
	if err := positiveID(accountID, "catalog-search"); err != nil {
		return robloxservices.CatalogPage{}, err
	}
	return service.core.SearchCatalog(ctx, accountID, search, cursor)
}
