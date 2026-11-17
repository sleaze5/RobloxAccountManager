package services

import (
	"context"
	"slices"
	"strconv"
)

// plainAvatarBodyColor is the grey of the plain avatar that wears an item in
// the 3D viewer, the Medium stone grey of Roblox.
const plainAvatarBodyColor = "A3A2A5"

// InventoryItemLook is how the 3D viewer of the application shows an item.
// AccessoryAssetID names an accessory that shows on its own. Avatar is an
// avatar that shows the item: a plain avatar wearing it, the outfit of a
// bundle, the saved avatar of the account wearing an animation, or the saved
// avatar that plays an emote. When neither is set, the viewer shows the model
// that Roblox renders of the item, such as for models and gear.
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
	// An emote plays on the saved avatar of the account, and an animation on
	// the saved avatar wearing it in place of the animation of the same kind.
	if category == InventoryEmotes {
		return avatarLook(service.savedAvatar(ctx, accountID, version, endpoint))
	}
	if category == InventoryAnimations {
		return avatarLook(service.savedAvatarWearing(ctx, accountID, version, endpoint, category, itemID))
	}
	if !canTryOn(category) {
		return InventoryItemLook{}, nil
	}
	typeName, _, err := service.wearableAssetTypeOf(ctx, accountID, version, endpoint, itemID)
	if err != nil {
		return InventoryItemLook{}, err
	}
	switch {
	case slices.Contains(inventoryAssetTypesByCategory[InventoryAccessories], typeName):
		// These accessories keep their shape on every body, so they show on
		// their own. Layered clothing and the parts of a head fit the body
		// that wears them.
		return InventoryItemLook{AccessoryAssetID: itemID}, nil
	case slices.Contains(inventoryAssetTypesByCategory[InventoryGear], typeName):
		// A plain avatar would hold the gear away from the camera, and Roblox
		// renders gear on its own.
		return InventoryItemLook{}, nil
	}
	return avatarLook(plainAvatarWearing(AvatarAsset{Category: category, Item: avatarItem(itemID, "", typeName)}), nil)
}

// avatarLook shows the item on the avatar, or fails with the error that
// reading the avatar returned.
func avatarLook(avatar Avatar, err error) (InventoryItemLook, error) {
	if err != nil {
		return InventoryItemLook{}, err
	}
	return InventoryItemLook{Avatar: &avatar}, nil
}

// bundleLook reads the outfit that Roblox lists to show the bundle. A bundle
// without one shows the model that Roblox renders.
func (service *Users) bundleLook(ctx context.Context, accountID, version int64, endpoint string, bundleID int64) (InventoryItemLook, error) {
	payload, err := service.readBundleDetails(ctx, accountID, version, endpoint, strconv.FormatInt(bundleID, 10))
	if err != nil {
		return InventoryItemLook{}, err
	}
	outfitID, hasOutfit := payload.outfitID()
	if !hasOutfit {
		return InventoryItemLook{}, nil
	}
	return avatarLook(service.AvatarOutfitDetails(ctx, accountID, version, outfitID))
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
