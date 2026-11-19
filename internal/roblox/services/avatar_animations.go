package services

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const avatarAnimationEndpoint = "avatar-animation"

var (
	// assetIDPattern matches a bare asset ID or a content ID, such as
	// rbxassetid://507766388.
	assetIDPattern = regexp.MustCompile(`(?i)^(?:rbxassetid://)?(\d{1,19})$`)
	// assetLinkPattern matches the links of an asset on the Roblox website and
	// the Creator Store, such as https://www.roblox.com/catalog/3576686446/Hello
	// and https://create.roblox.com/store/asset/507766388.
	assetLinkPattern  = regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.|web\.|create\.)?roblox\.com/(?:[a-z]{2}(?:-[a-z]{2})?/)?(?:catalog|library|marketplace/asset|store/asset|dashboard/creations/store)/(\d{1,19})(?:[/?#]|$)`)
	bundleLinkPattern = regexp.MustCompile(`(?i)roblox\.com/(?:[a-z]{2}(?:-[a-z]{2})?/)?bundles/\d`)
)

// playableAnimationTypeNames are the asset types that the 3D avatar plays: an
// emote, an animation of an avatar animation set, and an animation that a
// creator uploaded.
var playableAnimationTypeNames = append([]string{"EmoteAnimation", "Animation"}, inventoryAssetTypesByCategory[InventoryAnimations]...)

// assetIDOfReference reads the asset ID from what the user pasted: an ID, a
// content ID, or a link to the item.
func assetIDOfReference(reference string) (int64, error) {
	trimmed := strings.TrimSpace(reference)
	match := assetIDPattern.FindStringSubmatch(trimmed)
	if match == nil {
		match = assetLinkPattern.FindStringSubmatch(trimmed)
	}
	if match == nil {
		message := "Paste an asset ID or a link to an emote or animation."
		if bundleLinkPattern.MatchString(trimmed) {
			message = "Paste the link of one emote or animation, not of a bundle."
		}
		return 0, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: avatarAnimationEndpoint, Message: message}
	}
	assetID, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil || assetID <= 0 {
		return 0, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: avatarAnimationEndpoint, Message: "The asset ID is invalid.", Cause: err}
	}
	return assetID, nil
}

// AvatarAnimation reads the emote or animation that the reference names, so
// that the 3D avatar can play it.
func (service *Users) AvatarAnimation(ctx context.Context, accountID, version int64, reference string) (InventoryItem, error) {
	assetID, err := assetIDOfReference(reference)
	if err != nil {
		return InventoryItem{}, err
	}
	animation, err := service.animatedAsset(ctx, accountID, version, avatarAnimationEndpoint, assetID)
	if err != nil {
		return InventoryItem{}, err
	}
	if !slices.Contains(playableAnimationTypeNames, animation.TypeName) {
		return InventoryItem{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: avatarAnimationEndpoint, Message: "This item is not an emote or an animation."}
	}
	return animation, nil
}

// animatedAsset reads the name and type of an asset as an item. The economy
// API names any asset, while the catalog lists only items that it sells.
func (service *Users) animatedAsset(ctx context.Context, accountID, version int64, endpoint string, assetID int64) (InventoryItem, error) {
	var payload struct {
		Name        string `json:"Name"`
		AssetTypeID int64  `json:"AssetTypeId"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, endpoint, "https://economy.roblox.com/v2/assets/"+strconv.FormatInt(assetID, 10)+"/details", &payload); err != nil {
		return InventoryItem{}, err
	}
	return avatarItem(assetID, payload.Name, assetTypeNameByID[payload.AssetTypeID]), nil
}

// AddThumbnailEmoteDetails names the emotes of the customizations. A failed
// read leaves the emote without its name.
func (service *Users) AddThumbnailEmoteDetails(ctx context.Context, accountID, version int64, thumbnails *AvatarThumbnails) error {
	var readErrors []error
	for _, customization := range []*AvatarThumbnailCustomization{&thumbnails.Closeup, &thumbnails.FullBody} {
		if customization.Emote == nil {
			continue
		}
		emote, err := service.animatedAsset(ctx, accountID, version, "avatar-thumbnails", customization.Emote.ItemID)
		if err != nil {
			readErrors = append(readErrors, err)
			continue
		}
		*customization.Emote = emote
		customization.EmoteCategory = inventoryCategoryOfAssetType(emote.TypeName)
	}
	return errors.Join(readErrors...)
}
