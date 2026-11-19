package services

import (
	"context"
	"net/url"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	// catalogPageSize is one of the page sizes that the catalog search API
	// accepts: 10, 28, 30, 60, or 120. It rejects any other size. Verified on
	// 2026-11-19.
	catalogPageSize             = 30
	maxCatalogKeywordLength     = 100
	maxCatalogCreatorNameLength = 50
	catalogSearchEndpoint       = "catalog-search"
)

// CatalogCategory names one group of items that the Marketplace sells.
type CatalogCategory string

const (
	CatalogAll                 CatalogCategory = "all"
	CatalogAccessories         CatalogCategory = "accessories"
	CatalogHeadAccessories     CatalogCategory = "headAccessories"
	CatalogFaceAccessories     CatalogCategory = "faceAccessories"
	CatalogNeckAccessories     CatalogCategory = "neckAccessories"
	CatalogShoulderAccessories CatalogCategory = "shoulderAccessories"
	CatalogFrontAccessories    CatalogCategory = "frontAccessories"
	CatalogBackAccessories     CatalogCategory = "backAccessories"
	CatalogWaistAccessories    CatalogCategory = "waistAccessories"
	CatalogGear                CatalogCategory = "gear"
	CatalogBody                CatalogCategory = "body"
	CatalogHair                CatalogCategory = "hair"
	CatalogHeads               CatalogCategory = "heads"
	CatalogFullBodies          CatalogCategory = "fullBodies"
	CatalogMakeup              CatalogCategory = "makeup"
	CatalogClothing            CatalogCategory = "clothing"
	CatalogShirts              CatalogCategory = "shirts"
	CatalogTShirts             CatalogCategory = "tShirts"
	CatalogSweaters            CatalogCategory = "sweaters"
	CatalogJackets             CatalogCategory = "jackets"
	CatalogPants               CatalogCategory = "pants"
	CatalogShorts              CatalogCategory = "shorts"
	CatalogDressesAndSkirts    CatalogCategory = "dressesAndSkirts"
	CatalogBodysuits           CatalogCategory = "bodysuits"
	CatalogShoes               CatalogCategory = "shoes"
	CatalogClassicShirts       CatalogCategory = "classicShirts"
	CatalogClassicTShirts      CatalogCategory = "classicTShirts"
	CatalogClassicPants        CatalogCategory = "classicPants"
	CatalogAnimations          CatalogCategory = "animations"
	CatalogEmotes              CatalogCategory = "emotes"
	CatalogAnimationBundles    CatalogCategory = "animationBundles"
	CatalogBackgrounds         CatalogCategory = "backgrounds"
)

// The taxonomy IDs that the Marketplace sends in the Taxonomy parameter of the
// search API, from its navigation-menu-items API. Verified on 2026-11-19.
var catalogTaxonomyByCategory = map[CatalogCategory]string{
	CatalogAll:                 "tZsUsd2BqGViQrJ9Vs3Wah",
	CatalogAccessories:         "wNYJso48d1XnhMyFWT3oX3",
	CatalogHeadAccessories:     "thukPQUFe5EHxRrtsgq43u",
	CatalogFaceAccessories:     "b5rkQhGSfwhfTK177y6ERa",
	CatalogNeckAccessories:     "tvckLAbtDAGwF9Y2sph2Ce",
	CatalogShoulderAccessories: "8i1sggNTrxoN799hpRrR1S",
	CatalogFrontAccessories:    "kj535RzJxHTCnXPBLo5KaE",
	CatalogBackAccessories:     "7yGywzp5JeDsNjhXWJqSw3",
	CatalogWaistAccessories:    "6jYQPDtkVwQZcKtvVpyYAN",
	CatalogGear:                "gUtxkR951g3VPgRkb5ay8u",
	CatalogBody:                "mwkR3QvDBVzYk4o9RAVVa2",
	CatalogHair:                "d6a7M7r9MMrpf7z7VxW9ZS",
	CatalogHeads:               "s5NLP7BS9PRLKvUVca4XKg",
	CatalogFullBodies:          "kfHzJwaNYM8s2CWRqYUhRa",
	CatalogMakeup:              "dBM565GC3YYTdi9UKvr87Z",
	CatalogClothing:            "5G3bZScC9Hxp2D2EUtGSNm",
	CatalogShirts:              "pJ71PxerdfEuarTNRtSZYs",
	CatalogTShirts:             "fLRqNzGqjX7MzcqeMro9hc",
	CatalogSweaters:            "31M6WgEMmyq9TTfk3pUUpZ",
	CatalogJackets:             "kPZpEVNdProGcqMbj1jDKJ",
	CatalogPants:               "1MvRtnnsy2FJWmkErSBxBa",
	CatalogShorts:              "etAPg889P243JyjdbZCXhw",
	CatalogDressesAndSkirts:    "oSSCBSqkQPZu6HataAUAxB",
	CatalogBodysuits:           "u5jaNLyf2ZhvR95GS37ui5",
	CatalogShoes:               "uLRgNoJ1awZkhpVw9WyvKo",
	CatalogClassicShirts:       "2a2rf9qyeTd8W5iegK2Prc",
	CatalogClassicTShirts:      "43H35nJKHzCQM63q3wPaJG",
	CatalogClassicPants:        "49cFNAJWuwiJJVUSADG6DR",
	CatalogAnimations:          "9fPpW3SJ6UkieQU5jJ4dgP",
	CatalogEmotes:              "ioNxAT977DFP2hMnAJbsbF",
	CatalogAnimationBundles:    "whf6kUVBwk2xdwKUmRYN6G",
	CatalogBackgrounds:         "ijFKKMZq6jV4gkJQC4v7L1",
}

// CatalogSortOrder is the order of the search results.
type CatalogSortOrder string

const (
	CatalogSortByRelevance         CatalogSortOrder = "relevance"
	CatalogSortByMostFavorited     CatalogSortOrder = "mostFavorited"
	CatalogSortByBestselling       CatalogSortOrder = "bestselling"
	CatalogSortByRecentlyPublished CatalogSortOrder = "recentlyPublished"
	CatalogSortByHighestPrice      CatalogSortOrder = "highestPrice"
	CatalogSortByLowestPrice       CatalogSortOrder = "lowestPrice"
)

// The SortType values of the search API.
var catalogSortTypeBySortOrder = map[CatalogSortOrder]string{
	CatalogSortByRelevance:         "0",
	CatalogSortByMostFavorited:     "1",
	CatalogSortByBestselling:       "2",
	CatalogSortByRecentlyPublished: "3",
	CatalogSortByLowestPrice:       "4",
	CatalogSortByHighestPrice:      "5",
}

// CatalogSortPeriod is the time over which the most favorited and bestselling
// orders count favorites and sales.
type CatalogSortPeriod string

const (
	CatalogPastDay  CatalogSortPeriod = "pastDay"
	CatalogPastWeek CatalogSortPeriod = "pastWeek"
	CatalogAllTime  CatalogSortPeriod = "allTime"
)

// The SortAggregation values of the search API.
var catalogSortAggregationBySortPeriod = map[CatalogSortPeriod]string{
	CatalogPastDay:  "1",
	CatalogPastWeek: "3",
	CatalogAllTime:  "5",
}

// CatalogSalesType narrows the results to limited or timed items.
type CatalogSalesType string

const (
	CatalogAnySalesType CatalogSalesType = "any"
	CatalogLimited      CatalogSalesType = "limited"
	CatalogTimed        CatalogSalesType = "timed"
)

// The SalesTypeFilter values of the search API.
var catalogSalesTypeFilterBySalesType = map[CatalogSalesType]string{
	CatalogAnySalesType: "1",
	CatalogLimited:      "2",
	CatalogTimed:        "4",
}

// CatalogCreatorType says whether CreatorName names a user or a group.
type CatalogCreatorType string

const (
	CatalogCreatorUser  CatalogCreatorType = "User"
	CatalogCreatorGroup CatalogCreatorType = "Group"
)

// The CreatorType values of the search API.
var catalogCreatorTypeFilterByCreatorType = map[CatalogCreatorType]string{
	CatalogCreatorUser:  "1",
	CatalogCreatorGroup: "2",
}

// CatalogSearch is what to search the Marketplace for.
type CatalogSearch struct {
	Keyword  string          `json:"keyword"`
	Category CatalogCategory `json:"category"`
	// CreatorName is empty to list items of every creator.
	CreatorName string             `json:"creatorName"`
	CreatorType CatalogCreatorType `json:"creatorType"`
	// MinPriceInRobux and MaxPriceInRobux are nil for no bound.
	MinPriceInRobux *int64            `json:"minPriceInRobux"`
	MaxPriceInRobux *int64            `json:"maxPriceInRobux"`
	SortOrder       CatalogSortOrder  `json:"sortOrder"`
	SortPeriod      CatalogSortPeriod `json:"sortPeriod"`
	SalesType       CatalogSalesType  `json:"salesType"`
	// ShouldIncludeHiddenItems lists the items that are off sale, which the
	// Marketplace hides from its results unless asked.
	ShouldIncludeHiddenItems bool `json:"shouldIncludeHiddenItems"`
}

type CatalogPage struct {
	Items      []CategorizedInventoryItem `json:"items"`
	NextCursor string                     `json:"nextCursor"`
}

// validateCatalogSearch checks every field of the search, so that the search
// API never receives a value it does not know.
func validateCatalogSearch(search CatalogSearch) error {
	switch {
	case utf8.RuneCountInString(search.Keyword) > maxCatalogKeywordLength:
		return invalidCatalogSearch("Shorten the search to 100 characters or fewer.")
	case catalogTaxonomyByCategory[search.Category] == "":
		return invalidCatalogSearch("Select a Marketplace category.")
	case utf8.RuneCountInString(search.CreatorName) > maxCatalogCreatorNameLength:
		return invalidCatalogSearch("Shorten the creator name to 50 characters or fewer.")
	case search.CreatorName != "" && catalogCreatorTypeFilterByCreatorType[search.CreatorType] == "":
		return invalidCatalogSearch("Select whether the creator is a user or a group.")
	case catalogSortTypeBySortOrder[search.SortOrder] == "":
		return invalidCatalogSearch("Select a sort order.")
	case catalogSortAggregationBySortPeriod[search.SortPeriod] == "":
		return invalidCatalogSearch("Select a time period.")
	case catalogSalesTypeFilterBySalesType[search.SalesType] == "":
		return invalidCatalogSearch("Select a sales type.")
	}
	return validateCatalogPriceRange(search.MinPriceInRobux, search.MaxPriceInRobux)
}

func validateCatalogPriceRange(minPriceInRobux, maxPriceInRobux *int64) error {
	switch {
	case (minPriceInRobux != nil && *minPriceInRobux < 0) || (maxPriceInRobux != nil && *maxPriceInRobux < 0):
		return invalidCatalogSearch("Enter prices of 0 or more.")
	case minPriceInRobux != nil && maxPriceInRobux != nil && *minPriceInRobux > *maxPriceInRobux:
		return invalidCatalogSearch("Enter a minimum price that is not above the maximum price.")
	}
	return nil
}

func invalidCatalogSearch(message string) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: catalogSearchEndpoint, Message: message}
}

// catalogSearchEntry is one result of the search API. It lists the same
// market fields as the details API, plus the type and the price status.
type catalogSearchEntry struct {
	catalogItemDetails

	Name       string `json:"name"`
	AssetType  int64  `json:"assetType"`
	BundleType int64  `json:"bundleType"`
	// PriceStatus is "Off Sale" for an item that is off sale, and "No
	// Resellers" for a sold out limited item that nobody resells. The API
	// writes a price of 0 for both. Verified on 2026-11-19.
	PriceStatus string `json:"priceStatus"`
}

// bundleTypeNameByID names the Roblox bundle types by the IDs of the
// BundleType enum, with the names that the bundles API writes.
var bundleTypeNameByID = map[int64]string{
	1: "BodyParts", 2: "AvatarAnimations", 3: "Shoes", 4: "DynamicHead",
}

// SearchCatalog reads one page of the Marketplace items that match the
// search, without images.
func (service *Users) SearchCatalog(ctx context.Context, accountID, version int64, search CatalogSearch, cursor string) (CatalogPage, error) {
	if len(cursor) > maxCursorLength {
		return CatalogPage{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: catalogSearchEndpoint, Message: "This page has expired. Refresh the search."}
	}
	if err := validateCatalogSearch(search); err != nil {
		return CatalogPage{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var payload struct {
		NextPageCursor *string               `json:"nextPageCursor"`
		Data           *[]catalogSearchEntry `json:"data"`
	}
	address := &url.URL{Scheme: "https", Host: "catalog.roblox.com", Path: "/v2/search/items/details", RawQuery: catalogSearchQuery(search, cursor).Encode()}
	if err := service.readLargeAccountDetails(ctx, accountID, version, catalogSearchEndpoint, address, &payload, "Roblox returned invalid Marketplace data."); err != nil {
		return CatalogPage{}, err
	}
	if payload.Data == nil || len(*payload.Data) > catalogPageSize || (payload.NextPageCursor != nil && len(*payload.NextPageCursor) > maxCursorLength) {
		return CatalogPage{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: catalogSearchEndpoint, Message: "Roblox returned invalid Marketplace data."}
	}
	page := CatalogPage{Items: make([]CategorizedInventoryItem, 0, len(*payload.Data))}
	for _, listedEntry := range *payload.Data {
		if categorizedItem, ok := catalogItemOf(listedEntry); ok {
			page.Items = append(page.Items, categorizedItem)
		}
	}
	if payload.NextPageCursor != nil {
		page.NextCursor = *payload.NextPageCursor
	}
	return page, nil
}

func catalogSearchQuery(search CatalogSearch, cursor string) url.Values {
	query := url.Values{
		"Limit":           {strconv.Itoa(catalogPageSize)},
		"SortType":        {catalogSortTypeBySortOrder[search.SortOrder]},
		"SalesTypeFilter": {catalogSalesTypeFilterBySalesType[search.SalesType]},
	}
	// The search API returns a single page for the taxonomy of every item
	// when it sorts by relevance without a keyword, but pages through every
	// item without a taxonomy. Verified on 2026-11-19.
	if search.Category != CatalogAll {
		query.Set("Taxonomy", catalogTaxonomyByCategory[search.Category])
	}
	if search.SortOrder == CatalogSortByMostFavorited || search.SortOrder == CatalogSortByBestselling {
		query.Set("SortAggregation", catalogSortAggregationBySortPeriod[search.SortPeriod])
	}
	if search.Keyword != "" {
		query.Set("Keyword", search.Keyword)
	}
	if search.CreatorName != "" {
		query.Set("CreatorName", search.CreatorName)
		query.Set("CreatorType", catalogCreatorTypeFilterByCreatorType[search.CreatorType])
	}
	if search.MinPriceInRobux != nil {
		query.Set("MinPrice", strconv.FormatInt(*search.MinPriceInRobux, 10))
	}
	if search.MaxPriceInRobux != nil {
		query.Set("MaxPrice", strconv.FormatInt(*search.MaxPriceInRobux, 10))
	}
	if search.ShouldIncludeHiddenItems {
		query.Set("IncludeNotForSale", "true")
	}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	return query
}

// catalogItemOf converts a search result into an item with its inventory
// category, so that the item details show it as they show an owned item.
func catalogItemOf(listedEntry catalogSearchEntry) (CategorizedInventoryItem, bool) {
	if listedEntry.ID <= 0 {
		return CategorizedInventoryItem{}, false
	}
	item := InventoryItem{
		ItemID: listedEntry.ID, Name: boundedText(listedEntry.Name, maxInventoryItemNameLength),
		Description:             boundedText(listedEntry.Description, maxInventoryDescriptionLength),
		CreatorName:             boundedText(listedEntry.CreatorName, maxInventoryItemNameLength),
		CreatorHasVerifiedBadge: listedEntry.CreatorHasVerifiedBadge,
		CatalogItemType:         listedEntry.ItemType,
	}
	item.setCreator(listedEntry.CreatorTargetID, listedEntry.CreatorType)
	var category InventoryCategory
	switch listedEntry.ItemType {
	case CatalogAsset:
		item.TypeName, item.thumbnailType = assetTypeNameByID[listedEntry.AssetType], "Asset"
		category = inventoryCategoryOfAssetTypeOrOtherAssets(item.TypeName)
	case CatalogBundle:
		item.TypeName, item.thumbnailType = bundleTypeNameByID[listedEntry.BundleType], "BundleThumbnail"
		category = InventoryBundles
	default:
		return CategorizedInventoryItem{}, false
	}
	if listedEntry.PriceStatus == "Off Sale" || listedEntry.PriceStatus == "No Resellers" {
		listedEntry.IsOffSale = true
	}
	item.Market = catalogMarket(listedEntry.catalogItemDetails)
	return CategorizedInventoryItem{Category: category, Item: item}, true
}
