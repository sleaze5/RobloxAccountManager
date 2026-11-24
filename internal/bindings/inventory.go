package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/appservice"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

func (service *Service) GetInventoryPage(ctx context.Context, accountID, userID int64, category robloxservices.InventoryCategory, cursor string) (robloxservices.InventoryPage, error) {
	if err := validateAccountAndTargetIDs(accountID, userID, "inventory"); err != nil {
		return robloxservices.InventoryPage{}, err
	}
	return service.core.GetInventoryPage(ctx, accountID, userID, category, cursor)
}

func (service *Service) GetInventoryItemDetails(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) (robloxservices.InventoryItemDetails, error) {
	if err := validateAccountAndTargetIDs(accountID, itemID, "inventory-item-details"); err != nil {
		return robloxservices.InventoryItemDetails{}, err
	}
	return service.core.GetInventoryItemDetails(ctx, accountID, category, itemID)
}

func (service *Service) GetInventoryItemModel(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) (robloxservices.RenderedModel, error) {
	if err := validateAccountAndTargetIDs(accountID, itemID, "inventory-item-model"); err != nil {
		return robloxservices.RenderedModel{}, err
	}
	return service.core.GetInventoryItemModel(ctx, accountID, category, itemID)
}

func (service *Service) TryOnInventoryItem(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) (string, error) {
	if err := validateAccountAndTargetIDs(accountID, itemID, "avatar-try-on"); err != nil {
		return "", err
	}
	return service.core.TryOnInventoryItem(ctx, accountID, category, itemID)
}

func (service *Service) GetInventoryItemTryOnAvatar(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) (robloxservices.Avatar, error) {
	if err := validateAccountAndTargetIDs(accountID, itemID, "avatar-try-on"); err != nil {
		return robloxservices.Avatar{}, err
	}
	return service.core.GetInventoryItemTryOnAvatar(ctx, accountID, category, itemID)
}

func (service *Service) GetInventoryItemLook(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) (robloxservices.InventoryItemLook, error) {
	if err := validateAccountAndTargetIDs(accountID, itemID, "inventory-item-look"); err != nil {
		return robloxservices.InventoryItemLook{}, err
	}
	return service.core.GetInventoryItemLook(ctx, accountID, category, itemID)
}

func (service *Service) GetInventoryRelatedItems(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) ([]robloxservices.CategorizedInventoryItem, error) {
	if err := validateAccountAndTargetIDs(accountID, itemID, "inventory-related-items"); err != nil {
		return nil, err
	}
	return service.core.GetInventoryRelatedItems(ctx, accountID, category, itemID)
}

func (service *Service) GetInventoryAssetItem(ctx context.Context, accountID, assetID int64) (robloxservices.CategorizedInventoryItem, error) {
	if err := validateAccountAndTargetIDs(accountID, assetID, "inventory-asset-item"); err != nil {
		return robloxservices.CategorizedInventoryItem{}, err
	}
	return service.core.GetInventoryAssetItem(ctx, accountID, assetID)
}

// DownloadInventoryItem downloads the file of the item, then asks where to save
// it. An empty path without an error means the user cancelled the dialog.
func (service *Service) DownloadInventoryItem(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64, itemName string) (string, error) {
	const operation = "asset-download"
	if err := validateAccountAndTargetIDs(accountID, itemID, operation); err != nil {
		return "", err
	}
	file, err := service.core.ReadInventoryItemFile(ctx, accountID, category, itemID)
	if err != nil {
		return "", err
	}
	return service.saveDownload(ctx, operation, itemID, itemName, file)
}

// DownloadInventoryItemModel packs the 3D model of the item with its
// textures, then asks where to save it. An empty path without an error means
// the user cancelled the dialog.
func (service *Service) DownloadInventoryItemModel(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64, itemName string) (string, error) {
	const operation = "model-download"
	if err := validateAccountAndTargetIDs(accountID, itemID, operation); err != nil {
		return "", err
	}
	file, err := service.core.ReadInventoryItemModelArchive(ctx, accountID, category, itemID)
	if err != nil {
		return "", err
	}
	return service.saveDownload(ctx, operation, itemID, itemName, file)
}

func (service *Service) saveDownload(ctx context.Context, operation string, itemID int64, itemName string, file appservice.DownloadableFile) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", downloadCanceled(operation, err)
	}
	path, err := chooseSaveFile(service.core.DefaultDownloadDirectory(), service.core.DownloadFileName(itemName, itemID, file.Extension), file.FormatName, "*."+file.Extension)
	if err != nil {
		return "", &roblox.Error{Kind: roblox.KindProtocol, Endpoint: operation, Message: "Could not open the save dialog.", Cause: err}
	}
	if err := ctx.Err(); err != nil {
		return "", downloadCanceled(operation, err)
	}
	if path == "" {
		return "", nil
	}
	return service.core.SaveDownloadedFile(path, operation, file)
}

func downloadCanceled(operation string, cause error) error {
	return &roblox.Error{Kind: roblox.KindCancelled, Endpoint: operation, Message: "The download was canceled.", Cause: cause}
}

func (service *Service) IsInventoryItemOwned(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) (bool, error) {
	if err := validateAccountAndTargetIDs(accountID, itemID, "inventory-item-ownership"); err != nil {
		return false, err
	}
	return service.core.IsInventoryItemOwned(ctx, accountID, category, itemID)
}

func (service *Service) IsInventoryItemFavorite(ctx context.Context, accountID int64, itemType robloxservices.CatalogItemType, itemID int64) (bool, error) {
	if err := validateAccountAndTargetIDs(accountID, itemID, "item-favorite"); err != nil {
		return false, err
	}
	return service.core.IsInventoryItemFavorite(ctx, accountID, itemType, itemID)
}

func (service *Service) SetInventoryItemFavorite(ctx context.Context, accountID int64, itemType robloxservices.CatalogItemType, itemID int64, isFavorite bool) error {
	if err := validateAccountAndTargetIDs(accountID, itemID, "item-favorite-update"); err != nil {
		return err
	}
	return service.core.SetInventoryItemFavorite(ctx, accountID, itemType, itemID, isFavorite)
}

func (service *Service) BuyInventoryItem(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID, expectedPriceInRobux int64) (bool, error) {
	if err := validateAccountAndTargetIDs(accountID, itemID, "inventory-item-purchase"); err != nil {
		return false, err
	}
	return service.core.BuyInventoryItem(ctx, accountID, category, itemID, expectedPriceInRobux)
}

func (service *Service) DeleteInventoryAsset(ctx context.Context, accountID, assetID int64) error {
	if err := validateAccountAndTargetIDs(accountID, assetID, "inventory-asset-delete"); err != nil {
		return err
	}
	return service.core.DeleteInventoryAsset(ctx, accountID, assetID)
}

func (service *Service) GetWornAssetIDs(ctx context.Context, accountID, userID int64) ([]int64, error) {
	if err := validateAccountAndTargetIDs(accountID, userID, "inventory-worn-assets"); err != nil {
		return nil, err
	}
	return service.core.GetWornAssetIDs(ctx, accountID, userID)
}
