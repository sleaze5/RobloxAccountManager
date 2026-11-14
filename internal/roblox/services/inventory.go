package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	inventoryPageSize          = 100
	maxInventoryCursorLength   = 4096
	maxInventoryItemNameLength = 200
	maxInventoryResponseBytes  = 2 << 20
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
	TypeName           string `json:"typeName"`
	SerialNumber       *int64 `json:"serialNumber"`
	RecentAveragePrice *int64 `json:"recentAveragePrice"`
	ImageURL           string `json:"imageUrl"`
	PageURL            string `json:"pageUrl"`
	thumbnailType      string
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
// first, without images.
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
		return service.assetsPage(ctx, accountID, version, user, assetTypes, category == InventoryPlaces, cursor)
	}
}

func (service *Users) assetsPage(ctx context.Context, accountID, version int64, user string, assetTypes []string, isPlace bool, cursor string) (InventoryPage, error) {
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
	pageURL := "https://www.roblox.com/catalog/"
	if isPlace {
		pageURL = "https://www.roblox.com/games/"
	}
	items := make([]InventoryItem, 0, len(*payload.Data))
	for _, listedAsset := range *payload.Data {
		items = append(items, InventoryItem{
			ItemID: listedAsset.AssetID, Name: boundedText(listedAsset.Name, maxInventoryItemNameLength), TypeName: listedAsset.AssetType,
			PageURL: pageURL + strconv.FormatInt(listedAsset.AssetID, 10), thumbnailType: "Asset",
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
			PageURL: "https://www.roblox.com/catalog/" + strconv.FormatInt(listedCollectible.AssetID, 10), thumbnailType: "Asset",
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
			ID   int64  `json:"id"`
			Name string `json:"name"`
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
		items = append(items, InventoryItem{
			ItemID: listedBadge.ID, Name: boundedText(listedBadge.Name, maxInventoryItemNameLength),
			PageURL: "https://www.roblox.com/badges/" + strconv.FormatInt(listedBadge.ID, 10), thumbnailType: "BadgeIcon",
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
			PageURL: "https://www.roblox.com/bundles/" + strconv.FormatInt(listedBundle.ID, 10), thumbnailType: "BundleThumbnail",
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
			GamePassID int64  `json:"gamePassId"`
			Name       string `json:"name"`
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
		items = append(items, InventoryItem{
			ItemID: listedGamePass.GamePassID, Name: boundedText(listedGamePass.Name, maxInventoryItemNameLength),
			PageURL: "https://www.roblox.com/game-pass/" + strconv.FormatInt(listedGamePass.GamePassID, 10), thumbnailType: "GamePass",
		})
	}
	var nextCursor *string
	if len(items) == inventoryPageSize {
		lastGamePassID := strconv.FormatInt(items[len(items)-1].ItemID, 10)
		nextCursor = &lastGamePassID
	}
	return inventoryPageOf(items, nextCursor, "inventory-game-passes")
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
