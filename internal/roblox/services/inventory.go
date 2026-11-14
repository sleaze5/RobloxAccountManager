package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	inventoryPageSize             = 100
	maxInventoryCursorLength      = 4096
	maxInventoryItemNameLength    = 200
	maxInventoryDescriptionLength = 4000
	maxInventoryResponseBytes     = 2 << 20
)

// InventoryCategory names one group of things that a user owns.
type InventoryCategory string

const (
	InventoryCollectibles    InventoryCategory = "collectibles"
	InventoryAccessories     InventoryCategory = "accessories"
	InventoryClassicClothing InventoryCategory = "classicClothing"
	InventoryLayeredClothing InventoryCategory = "layeredClothing"
	InventoryHeadsAndBodies  InventoryCategory = "headsAndBodies"
	InventoryFaces           InventoryCategory = "faces"
	InventoryBundles         InventoryCategory = "bundles"
	InventoryEmotes          InventoryCategory = "emotes"
	InventoryAnimations      InventoryCategory = "animations"
	InventoryGear            InventoryCategory = "gear"
	InventoryBadges          InventoryCategory = "badges"
	InventoryGamePasses      InventoryCategory = "gamePasses"
	InventoryPlaces          InventoryCategory = "places"
	InventoryModels          InventoryCategory = "models"
	InventoryDecals          InventoryCategory = "decals"
	InventoryAudio           InventoryCategory = "audio"
)

// The asset type names that the inventory v2 API accepts in its assetTypes
// parameter. It rejects the whole request when one name is unknown, and it
// rejects numeric asset type IDs. Verified against the API on 2026-11-14.
var inventoryAssetTypesByCategory = map[InventoryCategory][]string{
	InventoryAccessories:     {"Hat", "HairAccessory", "FaceAccessory", "NeckAccessory", "ShoulderAccessory", "FrontAccessory", "BackAccessory", "WaistAccessory"},
	InventoryClassicClothing: {"TShirt", "Shirt", "Pants"},
	InventoryLayeredClothing: {"TShirtAccessory", "ShirtAccessory", "PantsAccessory", "JacketAccessory", "SweaterAccessory", "ShortsAccessory", "LeftShoeAccessory", "RightShoeAccessory", "DressSkirtAccessory"},
	InventoryHeadsAndBodies:  {"DynamicHead", "Head", "EyebrowAccessory", "EyelashAccessory", "Torso", "RightArm", "LeftArm", "LeftLeg", "RightLeg"},
	InventoryFaces:           {"Face"},
	InventoryEmotes:          {"EmoteAnimation"},
	InventoryAnimations:      {"ClimbAnimation", "FallAnimation", "IdleAnimation", "JumpAnimation", "RunAnimation", "SwimAnimation", "WalkAnimation", "MoodAnimation"},
	InventoryGear:            {"Gear"},
	InventoryPlaces:          {"Place"},
	InventoryModels:          {"Model"},
	InventoryDecals:          {"Decal"},
	InventoryAudio:           {"Audio"},
}

// InventoryItem is one thing that a user owns.
type InventoryItem struct {
	ItemID int64  `json:"itemId"`
	Name   string `json:"name"`
	// TypeName is the Roblox asset type of an asset, such as HairAccessory, or
	// the Roblox bundle type of a bundle, such as DynamicHead.
	TypeName                string                 `json:"typeName"`
	SerialNumber            *int64                 `json:"serialNumber"`
	RecentAveragePrice      *int64                 `json:"recentAveragePrice"`
	ImageURL                string                 `json:"imageUrl"`
	Description             string                 `json:"description"`
	CreatorName             string                 `json:"creatorName"`
	CreatorHasVerifiedBadge bool                   `json:"creatorHasVerifiedBadge"`
	Market                  *InventoryItemMarket   `json:"market"`
	Badge                   *InventoryBadgeDetails `json:"badge"`
	// CatalogItemType is empty for an item that the catalog does not sell.
	CatalogItemType CatalogItemType `json:"catalogItemType"`
	thumbnailType   string
}

// InventoryItemMarket is how Roblox sells an item, in the catalog or as a game
// pass.
type InventoryItemMarket struct {
	IsForSale    bool  `json:"isForSale"`
	PriceInRobux int64 `json:"priceInRobux"`
	IsLimited    bool  `json:"isLimited"`
	// LowestResalePriceInRobux is the cheapest copy of a limited item that
	// another user resells.
	LowestResalePriceInRobux *int64 `json:"lowestResalePriceInRobux"`
	FavoriteCount            *int64 `json:"favoriteCount"`
	// TotalQuantity is how many copies exist of an item that Roblox sells in
	// a limited quantity, and RemainingQuantity is how many it still sells.
	TotalQuantity     *int64 `json:"totalQuantity"`
	RemainingQuantity *int64 `json:"remainingQuantity"`
	// SaleLocation is empty for an item that is not for sale.
	SaleLocation InventorySaleLocation `json:"saleLocation"`
	// OffSaleDeadlineMs is when a timed item goes off sale, or 0.
	OffSaleDeadlineMs int64 `json:"offSaleDeadlineMs"`
}

// InventorySaleLocation is where Roblox sells a catalog item.
type InventorySaleLocation string

const (
	SaleInMarketplace                   InventorySaleLocation = "marketplace"
	SaleInMarketplaceAndAllExperiences  InventorySaleLocation = "marketplaceAndAllExperiences"
	SaleInMarketplaceAndSomeExperiences InventorySaleLocation = "marketplaceAndSomeExperiences"
	SaleInExperiences                   InventorySaleLocation = "experiences"
)

// saleLocationOf reads the catalog saleLocationType, whose values RoSeal
// lists as ShopOnly, MyExperiencesOnly, ShopAndMyExperiences,
// ExperiencesById, ShopAndAllExperiences, ExperiencesDevApiOnly, and
// ShopAndExperiencesById.
func saleLocationOf(saleLocationType string) InventorySaleLocation {
	switch saleLocationType {
	case "ShopOnly":
		return SaleInMarketplace
	case "ShopAndAllExperiences":
		return SaleInMarketplaceAndAllExperiences
	case "ShopAndMyExperiences", "ShopAndExperiencesById":
		return SaleInMarketplaceAndSomeExperiences
	case "MyExperiencesOnly", "ExperiencesById", "ExperiencesDevApiOnly":
		return SaleInExperiences
	}
	return ""
}

type InventoryBadgeDetails struct {
	AwardedCount       int64   `json:"awardedCount"`
	PastDayWinFraction float64 `json:"pastDayWinFraction"`
	AwardingPlaceID    int64   `json:"awardingPlaceId"`
}

// ThumbnailTarget names the image of the item for the thumbnails batch API.
func (item InventoryItem) ThumbnailTarget() ThumbnailTarget {
	return ThumbnailTarget{Type: item.thumbnailType, TargetID: item.ItemID}
}

type InventoryPage struct {
	Items      []InventoryItem `json:"items"`
	NextCursor string          `json:"nextCursor"`
}

// InventoryPage reads one page of what the user owns in the category, newest
// first, without images or catalog details.
func (service *Users) InventoryPage(ctx context.Context, accountID, version, userID int64, category InventoryCategory, cursor string) (InventoryPage, error) {
	if len(cursor) > maxInventoryCursorLength {
		return InventoryPage{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "inventory", Message: "This page has expired. Refresh the inventory."}
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	user := strconv.FormatInt(userID, 10)
	switch category {
	case InventoryCollectibles:
		return service.collectiblesPage(ctx, accountID, version, user, cursor)
	case InventoryBadges:
		return service.badgesPage(ctx, accountID, version, user, cursor)
	case InventoryBundles:
		return service.bundlesPage(ctx, accountID, version, user, cursor)
	case InventoryGamePasses:
		return service.gamePassesPage(ctx, accountID, version, user, cursor)
	default:
		assetTypes, ok := inventoryAssetTypesByCategory[category]
		if !ok {
			return InventoryPage{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "inventory", Message: "Choose an inventory category."}
		}
		isCatalogCategory := category != InventoryPlaces && category != InventoryModels && category != InventoryDecals && category != InventoryAudio
		return service.assetsPage(ctx, accountID, version, user, assetTypes, isCatalogCategory, cursor)
	}
}

func (service *Users) assetsPage(ctx context.Context, accountID, version int64, user string, assetTypes []string, isCatalogCategory bool, cursor string) (InventoryPage, error) {
	var payload struct {
		NextPageCursor *string `json:"nextPageCursor"`
		Data           *[]struct {
			AssetID   int64  `json:"assetId"`
			Name      string `json:"name"`
			AssetType string `json:"assetType"`
		} `json:"data"`
	}
	query := pagedInventoryQuery(cursor)
	query.Set("assetTypes", strings.Join(assetTypes, ","))
	address := &url.URL{Scheme: "https", Host: "inventory.roblox.com", Path: "/v2/users/" + user + "/inventory", RawQuery: query.Encode()}
	if err := service.readInventory(ctx, accountID, version, "inventory-assets", address, &payload); err != nil {
		return InventoryPage{}, err
	}
	if payload.Data == nil {
		return InventoryPage{}, invalidInventory("inventory-assets")
	}
	var catalogItemType CatalogItemType
	if isCatalogCategory {
		catalogItemType = CatalogAsset
	}
	items := make([]InventoryItem, 0, len(*payload.Data))
	for _, listedAsset := range *payload.Data {
		items = append(items, InventoryItem{
			ItemID: listedAsset.AssetID, Name: boundedText(listedAsset.Name, maxInventoryItemNameLength), TypeName: listedAsset.AssetType,
			thumbnailType: "Asset", CatalogItemType: catalogItemType,
		})
	}
	return inventoryPageOf(items, payload.NextPageCursor, "inventory-assets")
}

func (service *Users) collectiblesPage(ctx context.Context, accountID, version int64, user, cursor string) (InventoryPage, error) {
	var payload struct {
		NextPageCursor *string `json:"nextPageCursor"`
		Data           *[]struct {
			AssetID            int64  `json:"assetId"`
			Name               string `json:"name"`
			SerialNumber       *int64 `json:"serialNumber"`
			RecentAveragePrice *int64 `json:"recentAveragePrice"`
		} `json:"data"`
	}
	address := &url.URL{Scheme: "https", Host: "inventory.roblox.com", Path: "/v1/users/" + user + "/assets/collectibles", RawQuery: pagedInventoryQuery(cursor).Encode()}
	if err := service.readInventory(ctx, accountID, version, "inventory-collectibles", address, &payload); err != nil {
		return InventoryPage{}, err
	}
	if payload.Data == nil {
		return InventoryPage{}, invalidInventory("inventory-collectibles")
	}
	items := make([]InventoryItem, 0, len(*payload.Data))
	for _, listedCollectible := range *payload.Data {
		item := InventoryItem{
			ItemID: listedCollectible.AssetID, Name: boundedText(listedCollectible.Name, maxInventoryItemNameLength),
			thumbnailType: "Asset", CatalogItemType: CatalogAsset,
		}
		if listedCollectible.SerialNumber != nil && *listedCollectible.SerialNumber > 0 {
			item.SerialNumber = listedCollectible.SerialNumber
		}
		if listedCollectible.RecentAveragePrice != nil && *listedCollectible.RecentAveragePrice > 0 {
			item.RecentAveragePrice = listedCollectible.RecentAveragePrice
		}
		items = append(items, item)
	}
	return inventoryPageOf(items, payload.NextPageCursor, "inventory-collectibles")
}

func (service *Users) badgesPage(ctx context.Context, accountID, version int64, user, cursor string) (InventoryPage, error) {
	var payload struct {
		NextPageCursor *string `json:"nextPageCursor"`
		Data           *[]struct {
			ID          int64  `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Creator     struct {
				Name string `json:"name"`
			} `json:"creator"`
			Awarder struct {
				ID   int64  `json:"id"`
				Type string `json:"type"`
			} `json:"awarder"`
			Statistics struct {
				AwardedCount      int64   `json:"awardedCount"`
				WinRatePercentage float64 `json:"winRatePercentage"`
			} `json:"statistics"`
		} `json:"data"`
	}
	address := &url.URL{Scheme: "https", Host: "badges.roblox.com", Path: "/v1/users/" + user + "/badges", RawQuery: pagedInventoryQuery(cursor).Encode()}
	if err := service.readInventory(ctx, accountID, version, "inventory-badges", address, &payload); err != nil {
		return InventoryPage{}, err
	}
	if payload.Data == nil {
		return InventoryPage{}, invalidInventory("inventory-badges")
	}
	items := make([]InventoryItem, 0, len(*payload.Data))
	for _, listedBadge := range *payload.Data {
		badge := &InventoryBadgeDetails{
			AwardedCount:       max(listedBadge.Statistics.AwardedCount, 0),
			PastDayWinFraction: min(max(listedBadge.Statistics.WinRatePercentage, 0), 1),
		}
		if listedBadge.Awarder.Type == "Place" && listedBadge.Awarder.ID > 0 {
			badge.AwardingPlaceID = listedBadge.Awarder.ID
		}
		items = append(items, InventoryItem{
			ItemID: listedBadge.ID, Name: boundedText(listedBadge.Name, maxInventoryItemNameLength),
			Description: boundedText(listedBadge.Description, maxInventoryDescriptionLength),
			CreatorName: boundedText(listedBadge.Creator.Name, maxInventoryItemNameLength),
			Badge:       badge, thumbnailType: "BadgeIcon",
		})
	}
	return inventoryPageOf(items, payload.NextPageCursor, "inventory-badges")
}

func (service *Users) bundlesPage(ctx context.Context, accountID, version int64, user, cursor string) (InventoryPage, error) {
	var payload struct {
		NextPageCursor *string `json:"nextPageCursor"`
		Data           *[]struct {
			ID         int64  `json:"id"`
			Name       string `json:"name"`
			BundleType string `json:"bundleType"`
		} `json:"data"`
	}
	address := &url.URL{Scheme: "https", Host: "catalog.roblox.com", Path: "/v1/users/" + user + "/bundles", RawQuery: pagedInventoryQuery(cursor).Encode()}
	if err := service.readInventory(ctx, accountID, version, "inventory-bundles", address, &payload); err != nil {
		return InventoryPage{}, err
	}
	if payload.Data == nil {
		return InventoryPage{}, invalidInventory("inventory-bundles")
	}
	items := make([]InventoryItem, 0, len(*payload.Data))
	for _, listedBundle := range *payload.Data {
		items = append(items, InventoryItem{
			ItemID: listedBundle.ID, Name: boundedText(listedBundle.Name, maxInventoryItemNameLength), TypeName: listedBundle.BundleType,
			thumbnailType: "BundleThumbnail", CatalogItemType: CatalogBundle,
		})
	}
	return inventoryPageOf(items, payload.NextPageCursor, "inventory-bundles")
}

// gamePassesPage pages by the ID of the last pass of the previous page,
// because the game passes API has no cursor. Verified on 2026-11-14.
func (service *Users) gamePassesPage(ctx context.Context, accountID, version int64, user, cursor string) (InventoryPage, error) {
	query := url.Values{"count": {strconv.Itoa(inventoryPageSize)}}
	if cursor != "" {
		if _, err := strconv.ParseInt(cursor, 10, 64); err != nil {
			return InventoryPage{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "inventory-game-passes", Message: "This page has expired. Refresh the inventory.", Cause: err}
		}
		query.Set("exclusiveStartId", cursor)
	}
	var payload struct {
		GamePasses *[]struct {
			GamePassID  int64  `json:"gamePassId"`
			Name        string `json:"name"`
			Description string `json:"description"`
			IsForSale   bool   `json:"isForSale"`
			Price       *int64 `json:"price"`
			Creator     struct {
				Name string `json:"name"`
			} `json:"creator"`
		} `json:"gamePasses"`
	}
	address := &url.URL{Scheme: "https", Host: "apis.roblox.com", Path: "/game-passes/v1/users/" + user + "/game-passes", RawQuery: query.Encode()}
	if err := service.readInventory(ctx, accountID, version, "inventory-game-passes", address, &payload); err != nil {
		return InventoryPage{}, err
	}
	if payload.GamePasses == nil {
		return InventoryPage{}, invalidInventory("inventory-game-passes")
	}
	items := make([]InventoryItem, 0, len(*payload.GamePasses))
	for _, listedGamePass := range *payload.GamePasses {
		market := &InventoryItemMarket{}
		if listedGamePass.IsForSale && listedGamePass.Price != nil && *listedGamePass.Price >= 0 {
			market.IsForSale, market.PriceInRobux = true, *listedGamePass.Price
		}
		items = append(items, InventoryItem{
			ItemID: listedGamePass.GamePassID, Name: boundedText(listedGamePass.Name, maxInventoryItemNameLength),
			Description: boundedText(listedGamePass.Description, maxInventoryDescriptionLength),
			CreatorName: boundedText(listedGamePass.Creator.Name, maxInventoryItemNameLength),
			Market:      market, thumbnailType: "GamePass",
		})
	}
	var nextCursor *string
	if len(items) == inventoryPageSize {
		lastGamePassID := strconv.FormatInt(items[len(items)-1].ItemID, 10)
		nextCursor = &lastGamePassID
	}
	return inventoryPageOf(items, nextCursor, "inventory-game-passes")
}

type catalogItemKey struct {
	ItemType CatalogItemType `json:"itemType"`
	ID       int64           `json:"id"`
}

type catalogItemDetails struct {
	ID                      int64           `json:"id"`
	ItemType                CatalogItemType `json:"itemType"`
	Description             string          `json:"description"`
	CreatorName             string          `json:"creatorName"`
	CreatorHasVerifiedBadge bool            `json:"creatorHasVerifiedBadge"`
	Price                   *int64          `json:"price"`
	IsOffSale               bool            `json:"isOffSale"`
	LowestResalePrice       *int64          `json:"lowestResalePrice"`
	FavoriteCount           *int64          `json:"favoriteCount"`
	ItemRestrictions        []string        `json:"itemRestrictions"`
	TotalQuantity           *int64          `json:"totalQuantity"`
	UnitsAvailable          *int64          `json:"unitsAvailableForConsumption"`
	SaleLocationType        string          `json:"saleLocationType"`
	OffSaleDeadline         *string         `json:"offSaleDeadline"`
}

// AddCatalogDetails adds the description, creator, and price that the catalog
// shows for each item that it sells. The catalog leaves out an item that it
// does not sell.
func (service *Users) AddCatalogDetails(ctx context.Context, accountID, version int64, items []InventoryItem) error {
	requestedItems := make([]catalogItemKey, 0, len(items))
	for _, item := range items {
		if item.CatalogItemType != "" {
			requestedItems = append(requestedItems, catalogItemKey{ItemType: item.CatalogItemType, ID: item.ItemID})
		}
	}
	if len(requestedItems) == 0 {
		return nil
	}
	listedItems, err := service.readCatalogDetails(ctx, accountID, version, requestedItems)
	if err != nil {
		return err
	}
	listedItemByKey := make(map[catalogItemKey]catalogItemDetails, len(listedItems))
	for _, listedItem := range listedItems {
		listedItemByKey[catalogItemKey{ItemType: listedItem.ItemType, ID: listedItem.ID}] = listedItem
	}
	for index := range items {
		listedItem, ok := listedItemByKey[catalogItemKey{ItemType: items[index].CatalogItemType, ID: items[index].ItemID}]
		if items[index].CatalogItemType == "" || !ok {
			continue
		}
		items[index].Market = catalogMarket(listedItem)
		items[index].Description = boundedText(listedItem.Description, maxInventoryDescriptionLength)
		items[index].CreatorName = boundedText(listedItem.CreatorName, maxInventoryItemNameLength)
		items[index].CreatorHasVerifiedBadge = listedItem.CreatorHasVerifiedBadge
	}
	return nil
}

// readCatalogDetails reads up to 120 items in one request, more than one
// inventory page holds. Verified against the API on 2026-11-14.
func (service *Users) readCatalogDetails(ctx context.Context, accountID, version int64, requestedItems []catalogItemKey) ([]catalogItemDetails, error) {
	body, err := json.Marshal(map[string]any{"items": requestedItems})
	if err != nil {
		return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "inventory-catalog-details", Message: "The catalog request could not be created.", Cause: err}
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: "inventory-catalog-details", AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodPost, URL: &url.URL{Scheme: "https", Host: "catalog.roblox.com", Path: "/v1/catalog/items/details"},
		Body: body, ContentType: "application/json", RequiresCSRF: true, MaxResponseSize: maxInventoryResponseBytes,
		Retry: roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
	})
	if err != nil {
		return nil, err
	}
	var payload struct {
		Data *[]catalogItemDetails `json:"data"`
	}
	if err := json.Unmarshal(response.Body, &payload); err != nil || payload.Data == nil {
		return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "inventory-catalog-details", Status: response.Status, Message: "Roblox returned invalid catalog data.", Cause: err}
	}
	return *payload.Data, nil
}

func catalogMarket(listedItem catalogItemDetails) *InventoryItemMarket {
	// Roblox keeps the last price of an item that is off sale.
	market := &InventoryItemMarket{
		IsForSale: !listedItem.IsOffSale && listedItem.Price != nil && *listedItem.Price >= 0,
		IsLimited: slices.ContainsFunc(listedItem.ItemRestrictions, func(restriction string) bool {
			return restriction == "Limited" || restriction == "LimitedUnique" || restriction == "Collectible"
		}),
	}
	if market.IsForSale {
		market.PriceInRobux = *listedItem.Price
	}
	if market.IsLimited && listedItem.LowestResalePrice != nil && *listedItem.LowestResalePrice > 0 {
		market.LowestResalePriceInRobux = listedItem.LowestResalePrice
	}
	if listedItem.FavoriteCount != nil && *listedItem.FavoriteCount >= 0 {
		market.FavoriteCount = listedItem.FavoriteCount
	}
	addCatalogAvailability(market, listedItem)
	return market
}

// addCatalogAvailability adds how many copies the catalog sells, where, and
// until when.
func addCatalogAvailability(market *InventoryItemMarket, listedItem catalogItemDetails) {
	// Roblox writes a total quantity of 0 for an item without a limit.
	if listedItem.TotalQuantity != nil && *listedItem.TotalQuantity > 0 {
		market.TotalQuantity = listedItem.TotalQuantity
		if market.IsForSale && listedItem.UnitsAvailable != nil && *listedItem.UnitsAvailable >= 0 {
			market.RemainingQuantity = listedItem.UnitsAvailable
		}
	}
	if market.IsForSale {
		market.SaleLocation = saleLocationOf(listedItem.SaleLocationType)
		if listedItem.OffSaleDeadline != nil {
			market.OffSaleDeadlineMs = robloxTimeMs(*listedItem.OffSaleDeadline)
		}
	}
}

func pagedInventoryQuery(cursor string) url.Values {
	query := url.Values{"limit": {strconv.Itoa(inventoryPageSize)}, "sortOrder": {"Desc"}}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	return query
}

// inventoryPageOf drops entries without a valid ID and checks the page bounds.
func inventoryPageOf(listedItems []InventoryItem, nextCursor *string, endpoint string) (InventoryPage, error) {
	if len(listedItems) > inventoryPageSize || (nextCursor != nil && len(*nextCursor) > maxInventoryCursorLength) {
		return InventoryPage{}, invalidInventory(endpoint)
	}
	page := InventoryPage{Items: make([]InventoryItem, 0, len(listedItems))}
	for _, listedItem := range listedItems {
		if listedItem.ItemID > 0 {
			page.Items = append(page.Items, listedItem)
		}
	}
	if nextCursor != nil {
		page.NextCursor = *nextCursor
	}
	return page, nil
}

// readInventory reports a private inventory with its own message, because
// Roblox answers 403 for an inventory that the account may not view.
func (service *Users) readInventory(ctx context.Context, accountID, version int64, endpoint string, address *url.URL, target any) error {
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: endpoint, AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodGet, URL: address, MaxResponseSize: maxInventoryResponseBytes,
		Retry: roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
	})
	if remote, ok := errors.AsType[*roblox.Error](err); ok && remote.Kind == roblox.KindForbidden && remote.Status == http.StatusForbidden {
		return &roblox.Error{Kind: roblox.KindForbidden, Endpoint: endpoint, Status: remote.Status, RobloxCode: remote.RobloxCode, Message: "This inventory is private.", Cause: err}
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(response.Body, target); err != nil {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Status: response.Status, Message: "Roblox returned invalid inventory data.", Cause: err}
	}
	return nil
}

func invalidInventory(endpoint string) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Roblox returned invalid inventory data."}
}

// WornAssetIDs lists the assets that the user wears now, including the parts
// of worn bundles. Roblox lists them for any user. Verified on 2026-11-14.
func (service *Users) WornAssetIDs(ctx context.Context, accountID, version, userID int64) ([]int64, error) {
	var payload struct {
		AssetIDs *[]int64 `json:"assetIds"`
	}
	address := "https://avatar.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/currently-wearing"
	if err := service.readAccountDetails(ctx, accountID, version, "inventory-worn-assets", address, &payload); err != nil {
		return nil, err
	}
	if payload.AssetIDs == nil {
		return nil, invalidInventory("inventory-worn-assets")
	}
	return *payload.AssetIDs, nil
}
