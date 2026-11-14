package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

// CatalogItemType is the kind of item that the catalog sells.
type CatalogItemType string

const (
	CatalogAsset  CatalogItemType = "Asset"
	CatalogBundle CatalogItemType = "Bundle"
)

// ValidCatalogItemType reports whether the catalog sells this kind of item.
func ValidCatalogItemType(itemType CatalogItemType) bool {
	return itemType == CatalogAsset || itemType == CatalogBundle
}

func catalogFavoriteAddress(userID int64, itemType CatalogItemType, itemID int64) string {
	collection := "assets"
	if itemType == CatalogBundle {
		collection = "bundles"
	}
	return "https://catalog.roblox.com/v1/favorites/users/" + strconv.FormatInt(userID, 10) + "/" + collection + "/" + strconv.FormatInt(itemID, 10) + "/favorite"
}

// IsItemFavorite reads whether the user favorited the catalog item. Roblox
// answers null for an item that is not a favorite. Verified on 2026-11-14.
func (service *Users) IsItemFavorite(ctx context.Context, accountID, version, userID int64, itemType CatalogItemType, itemID int64) (bool, error) {
	var payload json.RawMessage
	if err := service.readAccountDetails(ctx, accountID, version, "item-favorite", catalogFavoriteAddress(userID, itemType, itemID), &payload); err != nil {
		return false, err
	}
	return !bytes.Equal(bytes.TrimSpace(payload), []byte("null")), nil
}

func (service *Users) SetItemFavorite(ctx context.Context, accountID, version, userID int64, itemType CatalogItemType, itemID int64, isFavorite bool) error {
	method := http.MethodDelete
	if isFavorite {
		method = http.MethodPost
	}
	_, err := service.writeAccount(ctx, accountID, version, "item-favorite-update", method, catalogFavoriteAddress(userID, itemType, itemID), nil)
	return err
}

// DeleteInventoryAsset removes the asset from the inventory of the account.
// Roblox refuses limited assets and assets that Roblox created.
func (service *Users) DeleteInventoryAsset(ctx context.Context, accountID, version, assetID int64) error {
	_, err := service.writeAccount(ctx, accountID, version, "inventory-asset-delete", http.MethodDelete, "https://inventory.roblox.com/v2/inventory/asset/"+strconv.FormatInt(assetID, 10), nil)
	if remote, ok := errors.AsType[*roblox.Error](err); ok && remote.Status == http.StatusBadRequest {
		return &roblox.Error{Kind: remote.Kind, Endpoint: remote.Endpoint, Status: remote.Status, RobloxCode: remote.RobloxCode, Message: "Roblox does not allow deleting this item.", Cause: err}
	}
	return err
}
