package services

import (
	"context"
	"slices"
	"strconv"
)

// plainAvatarBodyColor is the grey of the plain avatar that wears an item in
// the 3D viewer, the Medium stone grey of Roblox.
const plainAvatarBodyColor = "A3A2A5"

// shapeKeepingAccessoryTypeNames are the accessory types that keep their shape
// on every body, so the 3D viewer shows them on their own. Layered clothing
// and the parts of a head fit the body that wears them.
var shapeKeepingAccessoryTypeNames = []string{
	"Hat", "HairAccessory", "FaceAccessory", "NeckAccessory", "ShoulderAccessory",
	"FrontAccessory", "BackAccessory", "WaistAccessory",
}

// InventoryItemLook is how the 3D viewer of the application shows an item.
// AccessoryAssetID names an accessory that shows on its own. Avatar is an
// avatar that shows the item: a plain avatar wearing it, or the outfit of a
// bundle. When neither is set, the viewer shows the model that Roblox renders
// of the item, such as for models and gear.
type InventoryItemLook struct {
	AccessoryAssetID int64   `json:"accessoryAssetId"`
	Avatar           *Avatar `json:"avatar"`
}

// InventoryItemLook reads how the 3D viewer shows the item.
func (service *Users) InventoryItemLook(ctx context.Context, accountID, version int64, category InventoryCategory, itemID int64) (InventoryItemLook, error) {
	const endpoint = "inventory-item-look"
	if category == InventoryBundles {
		return service.bundleLook(ctx, accountID, version, endpoint, itemID)
	}
	if !canTryOn(category) {
		return InventoryItemLook{}, nil
	}
	typeName, _, err := service.wearableAssetTypeOf(ctx, accountID, version, endpoint, itemID)
	if err != nil {
		return InventoryItemLook{}, err
	}
	switch {
	case slices.Contains(shapeKeepingAccessoryTypeNames, typeName):
		return InventoryItemLook{AccessoryAssetID: itemID}, nil
	case typeName == "Gear":
		// A plain avatar would hold the gear away from the camera, and Roblox
		// renders gear on its own.
		return InventoryItemLook{}, nil
	}
	avatar := plainAvatarWearing(AvatarAsset{Category: category, Item: avatarItem(itemID, "", typeName)})
	return InventoryItemLook{Avatar: &avatar}, nil
}

// bundleLook reads the outfit that Roblox lists to show the bundle.
func (service *Users) bundleLook(ctx context.Context, accountID, version int64, endpoint string, bundleID int64) (InventoryItemLook, error) {
	payload, err := service.readBundleDetails(ctx, accountID, version, endpoint, strconv.FormatInt(bundleID, 10))
	if err != nil {
		return InventoryItemLook{}, err
	}
	for _, bundledItem := range payload.BundledItems {
		if bundledItem.Type == "UserOutfit" && bundledItem.ID > 0 {
			outfit, err := service.AvatarOutfitDetails(ctx, accountID, version, bundledItem.ID)
			if err != nil {
				return InventoryItemLook{}, err
			}
			return InventoryItemLook{Avatar: &outfit}, nil
		}
	}
	return InventoryItemLook{}, nil
}

// plainAvatarWearing is a grey R15 avatar of the default size that wears only
// the asset, so that clothing, faces, and body parts show on a body.
func plainAvatarWearing(asset AvatarAsset) Avatar {
	return Avatar{
		AvatarType: AvatarR15,
		Scales:     AvatarScales{Height: 1, Width: 1, Head: 1},
		BodyColors: AvatarBodyColors{
			Head: plainAvatarBodyColor, Torso: plainAvatarBodyColor,
			LeftArm: plainAvatarBodyColor, RightArm: plainAvatarBodyColor,
			LeftLeg: plainAvatarBodyColor, RightLeg: plainAvatarBodyColor,
		},
		Assets: []AvatarAsset{asset},
	}
}
