package services

import (
	"cmp"
	"context"
	"net/url"
	"strconv"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
)

const (
	// Roblox returns at most 400 developer products per page.
	universeDeveloperProductsPageSize = 400
	// maxUniverseDeveloperProducts matches the thumbnails batch limit, so that
	// every listed product can get its icon.
	maxUniverseDeveloperProducts         = maxHeadshotIDs
	maxDeveloperProductNameLength        = 100
	maxDeveloperProductDescriptionLength = 1000
)

// DeveloperProducts lists a universe's developer products without icons.
// Roblox lists them without a signed-in request. Some keys of a listed product
// are in PascalCase. Verified on 2026-11-22.
func (service *Games) DeveloperProducts(ctx context.Context, universeID int64) ([]games.DeveloperProduct, error) {
	const endpoint = "developer-products"
	path := "/developer-products/v2/universes/" + strconv.FormatInt(universeID, 10) + "/developerproducts"
	maxPages := (maxUniverseDeveloperProducts + universeDeveloperProductsPageSize - 1) / universeDeveloperProductsPageSize
	developerProducts := []games.DeveloperProduct{}
	err := readCursorPages(endpoint, maxPages, "This game has too many developer products.", func(cursor string) (string, error) {
		parameters := url.Values{"limit": {strconv.Itoa(universeDeveloperProductsPageSize)}}
		if cursor != "" {
			parameters.Set("cursor", cursor)
		}
		var payload struct {
			DeveloperProducts *[]struct {
				DeveloperProductID int64  `json:"DeveloperProductId"`
				Name               string `json:"Name"`
				DisplayName        string `json:"displayName"`
				DisplayDescription string `json:"displayDescription"`
				IsForSale          bool   `json:"IsForSale"`
				PriceInRobux       *int64 `json:"PriceInRobux"`
			} `json:"developerProducts"`
			NextPageCursor *string `json:"nextPageCursor"`
		}
		if err := service.get(ctx, endpoint, "apis.roblox.com", path, parameters, &payload); err != nil {
			return "", err
		}
		if payload.DeveloperProducts == nil || len(*payload.DeveloperProducts) > universeDeveloperProductsPageSize {
			return "", gameDataError("Roblox returned invalid developer product data.")
		}
		for _, listedProduct := range *payload.DeveloperProducts {
			if !games.ValidID(listedProduct.DeveloperProductID) {
				continue
			}
			developerProduct := games.DeveloperProduct{
				DeveloperProductID: listedProduct.DeveloperProductID,
				Name:               cmp.Or(boundedText(listedProduct.DisplayName, maxDeveloperProductNameLength), boundedText(listedProduct.Name, maxDeveloperProductNameLength)),
				Description:        boundedText(listedProduct.DisplayDescription, maxDeveloperProductDescriptionLength),
			}
			if listedProduct.IsForSale && listedProduct.PriceInRobux != nil && *listedProduct.PriceInRobux >= 0 {
				developerProduct.IsForSale, developerProduct.PriceInRobux = true, *listedProduct.PriceInRobux
			}
			developerProducts = append(developerProducts, developerProduct)
		}
		if len(developerProducts) > maxUniverseDeveloperProducts {
			return "", gameDataError("This game has too many developer products.")
		}
		if payload.NextPageCursor == nil {
			return "", nil
		}
		return *payload.NextPageCursor, nil
	})
	if err != nil {
		return nil, err
	}
	return developerProducts, nil
}
