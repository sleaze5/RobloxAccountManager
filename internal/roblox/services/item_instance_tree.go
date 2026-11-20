package services

import (
	"context"
	"errors"
	"strconv"
	"sync"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox/modelfile"
)

const (
	// maxExploredInstanceCount bounds the instances that one explorer shows
	// across every model, so that a large model stays responsive.
	maxExploredInstanceCount = 5000
	itemInstanceTreeEndpoint = "item-instance-tree"
)

// ExploredModel is the instance tree of the file of one asset.
type ExploredModel struct {
	AssetID int64 `json:"assetId"`
	// Name names an asset of a bundle. It is empty for the file of the item
	// itself.
	Name  string          `json:"name"`
	Model modelfile.Model `json:"model"`
	// UnavailableMessage tells why the file of an asset of a bundle could not
	// be read. Its model is then empty.
	UnavailableMessage string `json:"unavailableMessage"`
}

// ItemInstanceTree holds the instances of the file of an item, or of each
// asset of a bundle.
type ItemInstanceTree struct {
	Models []ExploredModel `json:"models"`
	// ImageURLByAssetID holds the small images of the assets that the
	// properties name, such as meshes and textures, for the assets that
	// Roblox renders.
	ImageURLByAssetID map[int64]string `json:"imageUrlByAssetId"`
}

// canExplore reports whether the items of the category are model files, or
// bundles of them, whose instances the explorer shows. Images, meshes, audio,
// and places are not.
func canExplore(category InventoryCategory) bool {
	switch category {
	case InventoryBundles, InventoryEmotes, InventoryAnimations, InventoryDecals:
		return true
	default:
		return hasDependencies(category)
	}
}

// ItemInstanceTree downloads and reads the file of the item, or of each
// asset of a bundle.
func (service *Users) ItemInstanceTree(ctx context.Context, accountID, version int64, category InventoryCategory, itemID int64) (ItemInstanceTree, error) {
	if !canExplore(category) {
		return ItemInstanceTree{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: itemInstanceTreeEndpoint, Message: "This item has no instances to explore."}
	}
	if category != InventoryBundles {
		file, err := service.downloadExploredFile(ctx, accountID, version, itemID)
		if err != nil {
			return ItemInstanceTree{}, err
		}
		model, err := decodeExploredFile(file, 1, maxExploredInstanceCount)
		if err != nil {
			return ItemInstanceTree{}, err
		}
		return ItemInstanceTree{Models: []ExploredModel{{AssetID: itemID, Model: model}}}, nil
	}
	return service.bundleInstanceTree(ctx, accountID, version, itemID)
}

// bundleInstanceTree reads the file of each asset of the bundle at once. An
// asset whose file fails shows why, unless every one fails.
func (service *Users) bundleInstanceTree(ctx context.Context, accountID, version, bundleID int64) (ItemInstanceTree, error) {
	payload, err := service.readBundleDetails(ctx, accountID, version, itemInstanceTreeEndpoint, strconv.FormatInt(bundleID, 10))
	if err != nil {
		return ItemInstanceTree{}, err
	}
	var models []ExploredModel
	for _, bundledItem := range payload.BundledItems {
		if bundledItem.Type == "Asset" && bundledItem.ID > 0 && len(models) < maxRelatedItemCount {
			models = append(models, ExploredModel{AssetID: bundledItem.ID, Name: boundedText(bundledItem.Name, maxInventoryItemNameLength)})
		}
	}
	files := make([][]byte, len(models))
	failures := make([]error, len(models))
	var group sync.WaitGroup
	for index, model := range models {
		group.Go(func() {
			files[index], failures[index] = service.downloadExploredFile(ctx, accountID, version, model.AssetID)
		})
	}
	group.Wait()
	// The files decode in bundle order, so that instance IDs stay unique
	// across the models and the instance limit applies to them in order.
	nextInstanceID, remainingInstanceCount, failureCount := 1, maxExploredInstanceCount, 0
	for index := range models {
		if failures[index] == nil {
			models[index].Model, failures[index] = decodeExploredFile(files[index], nextInstanceID, remainingInstanceCount)
		}
		if failures[index] != nil {
			failureCount++
			models[index].UnavailableMessage = unavailableAssetMessage(failures[index])
			continue
		}
		keptCount := countInstances(models[index].Model.Instances)
		nextInstanceID += keptCount
		remainingInstanceCount -= keptCount
	}
	if failureCount == len(models) && failureCount > 0 {
		// The first failure explains the others, which fail the same way.
		return ItemInstanceTree{}, failures[0]
	}
	return ItemInstanceTree{Models: models}, nil
}

// downloadExploredFile downloads the file of the asset in the form that
// Roblox stores, which the Download file action saves too.
func (service *Users) downloadExploredFile(ctx context.Context, accountID, version, assetID int64) ([]byte, error) {
	file, _, err := service.downloadAssetFile(ctx, accountID, version, itemInstanceTreeEndpoint, assetID, AssetRepresentation{}, maxAssetFileBytes)
	if remote, isRefused := assetFileRefusal(err); isRefused {
		return nil, &roblox.Error{Kind: remote.Kind, Endpoint: itemInstanceTreeEndpoint, Status: remote.Status, Message: "Roblox does not let this account open the file of this item.", Cause: err}
	}
	return file, err
}

func decodeExploredFile(file []byte, firstInstanceID, maxInstanceCount int) (modelfile.Model, error) {
	if !modelfile.IsBinary(file) && !modelfile.IsXML(file) {
		return modelfile.Model{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: itemInstanceTreeEndpoint, Message: "The file of this item holds no instances."}
	}
	model, err := modelfile.Decode(file, firstInstanceID, maxInstanceCount)
	if err != nil {
		return modelfile.Model{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: itemInstanceTreeEndpoint, Message: "Roblox returned a file that the app cannot read.", Cause: err}
	}
	return model, nil
}

// unavailableAssetMessage tells why the file of an asset of a bundle could not
// be read, in the words of the error that Roblox or the decoder returned.
func unavailableAssetMessage(err error) string {
	if remote, ok := errors.AsType[*roblox.Error](err); ok && remote.Message != "" {
		return remote.Message
	}
	return "Could not read the file of this asset."
}

func countInstances(instances []modelfile.Instance) int {
	count := len(instances)
	for _, instance := range instances {
		count += countInstances(instance.Children)
	}
	return count
}

// ReferencedAssetIDs lists the assets that the properties of the tree name,
// without repeating one, up to maxCount.
func (tree ItemInstanceTree) ReferencedAssetIDs(maxCount int) []int64 {
	seen := map[int64]bool{}
	var assetIDs []int64
	var visit func(instances []modelfile.Instance)
	visit = func(instances []modelfile.Instance) {
		for _, instance := range instances {
			for _, property := range instance.Properties {
				if property.AssetID > 0 && !seen[property.AssetID] && len(assetIDs) < maxCount {
					seen[property.AssetID] = true
					assetIDs = append(assetIDs, property.AssetID)
				}
			}
			visit(instance.Children)
		}
	}
	for _, model := range tree.Models {
		visit(model.Model.Instances)
	}
	return assetIDs
}
