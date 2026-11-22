package appservice

import (
	"context"

	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// maxExplorerImageCount bounds the referenced assets whose images one
// explorer shows.
const maxExplorerImageCount = 100

// GetItemInstanceTree reads the instances of the file of the item, or of each
// asset of a bundle, with their properties in the groups that Roblox Studio
// shows and the images of the assets that they name. Missing groups or images
// leave the tree without them.
func (service *Service) GetItemInstanceTree(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) (robloxservices.ItemInstanceTree, error) {
	const operation = "item-instance-tree"
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, operation)
	if err != nil {
		return robloxservices.ItemInstanceTree{}, err
	}
	defer done()
	logger := service.logs.Module("application.explorer").With("operation", operation, "account_id", accountID, "category", category)
	tree, err := service.users.ItemInstanceTree(ctx, accountID, version, category, itemID)
	if err != nil {
		logger.WarnContext(ctx, "item instances unavailable", "error", err)
		return robloxservices.ItemInstanceTree{}, err
	}
	if err := service.classReflection.GroupProperties(ctx, tree.Models); err != nil {
		logger.WarnContext(ctx, "property groups unavailable", "error", err)
	}
	assetIDs := tree.ReferencedAssetIDs(maxExplorerImageCount)
	targets := make([]robloxservices.ThumbnailTarget, len(assetIDs))
	for index, assetID := range assetIDs {
		targets[index] = robloxservices.ThumbnailTarget{Type: "Asset", TargetID: assetID}
	}
	images, err := service.thumbnails.Images(ctx, targets, robloxservices.SmallThumbnailSize)
	if err != nil {
		logger.WarnContext(ctx, "referenced asset images unavailable", "error", err)
	}
	tree.ImageURLByAssetID = make(map[int64]string, len(images))
	for target, imageURL := range images {
		tree.ImageURLByAssetID[target.TargetID] = imageURL
	}
	return tree, nil
}
