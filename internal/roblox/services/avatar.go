package services

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	maxWornAssetCount   = 100
	maxEmoteSlot        = 8
	maxOutfitNameLength = 100
	outfitPageSize      = 50
)

// AvatarType is the rig that an avatar uses.
type AvatarType string

const (
	AvatarR6  AvatarType = "R6"
	AvatarR15 AvatarType = "R15"
)

// AvatarScales are the body proportions of an R15 avatar as fractions. Roblox
// derives the depth from the width, so the application neither reads nor
// writes it. Verified against the API on 2026-11-14.
type AvatarScales struct {
	Height     float64 `json:"height"`
	Width      float64 `json:"width"`
	Head       float64 `json:"head"`
	Proportion float64 `json:"proportion"`
	BodyType   float64 `json:"bodyType"`
}

// AvatarBodyColors are the colors of the body parts as six uppercase hex
// digits, such as A3A2A5. A color is empty when Roblox lists an invalid one.
type AvatarBodyColors struct {
	Head     string `json:"head"`
	Torso    string `json:"torso"`
	LeftArm  string `json:"leftArm"`
	RightArm string `json:"rightArm"`
	LeftLeg  string `json:"leftLeg"`
	RightLeg string `json:"rightLeg"`
}

// AvatarAsset is an asset that an avatar wears.
type AvatarAsset struct {
	// Category is empty for an asset type that no inventory category lists.
	Category InventoryCategory `json:"category"`
	Item     InventoryItem     `json:"item"`
	// Adjustment is nil when the avatar wears the accessory where it attaches
	// by default.
	Adjustment *AvatarAccessoryAdjustment `json:"adjustment"`
	// Layer is nil for an asset that is not layered.
	Layer *AvatarClothingLayer `json:"layer"`
}

// AvatarEmote is an emote in one of the eight slots of the emote wheel.
type AvatarEmote struct {
	Slot int           `json:"slot"`
	Item InventoryItem `json:"item"`
}

// Avatar is how a user looks now. Assets lists every worn asset, including
// body parts and animations.
type Avatar struct {
	AvatarType AvatarType       `json:"avatarType"`
	Scales     AvatarScales     `json:"scales"`
	BodyColors AvatarBodyColors `json:"bodyColors"`
	Assets     []AvatarAsset    `json:"assets"`
	Emotes     []AvatarEmote    `json:"emotes"`
	// ImageURL is the full-body image, or empty while Roblox renders it.
	ImageURL string `json:"imageUrl"`
}

// ThumbnailTargets names the images of the worn assets and emotes.
func (avatar Avatar) ThumbnailTargets() []ThumbnailTarget {
	targets := make([]ThumbnailTarget, 0, len(avatar.Assets)+len(avatar.Emotes))
	for _, asset := range avatar.Assets {
		targets = append(targets, asset.Item.ThumbnailTarget())
	}
	for _, emote := range avatar.Emotes {
		targets = append(targets, emote.Item.ThumbnailTarget())
	}
	return targets
}

// AddImages sets the image of each worn asset and emote that the map holds.
func (avatar *Avatar) AddImages(images map[ThumbnailTarget]string) {
	for index := range avatar.Assets {
		avatar.Assets[index].Item.ImageURL = images[avatar.Assets[index].Item.ThumbnailTarget()]
	}
	for index := range avatar.Emotes {
		avatar.Emotes[index].Item.ImageURL = images[avatar.Emotes[index].Item.ThumbnailTarget()]
	}
}

// AvatarImageTarget names the full-body image of the user's avatar.
func AvatarImageTarget(userID int64) ThumbnailTarget {
	return ThumbnailTarget{Type: "Avatar", TargetID: userID}
}

// AvatarEmoteSlot puts an emote in a slot of the emote wheel.
type AvatarEmoteSlot struct {
	AssetID int64 `json:"assetId"`
	Slot    int   `json:"slot"`
}

// AvatarChangeAsset is an asset that a changed avatar wears.
type AvatarChangeAsset struct {
	AssetID int64 `json:"assetId"`
	// Adjustment is the placement of a rigid accessory, or nil to keep the
	// placement that Roblox stores. Only an asset that the avatar wears now
	// can change its placement.
	Adjustment *AvatarAccessoryAdjustment `json:"adjustment"`
}

// AvatarChange is the whole avatar that an account saves.
type AvatarChange struct {
	AvatarType AvatarType          `json:"avatarType"`
	Scales     AvatarScales        `json:"scales"`
	BodyColors AvatarBodyColors    `json:"bodyColors"`
	Assets     []AvatarChangeAsset `json:"assets"`
	Emotes     []AvatarEmoteSlot   `json:"emotes"`
}

// AvatarScaleLimit is the range of one scale and the step between values.
type AvatarScaleLimit struct {
	Min       float64 `json:"min"`
	Max       float64 `json:"max"`
	Increment float64 `json:"increment"`
}

type AvatarScaleLimits struct {
	Height     AvatarScaleLimit `json:"height"`
	Width      AvatarScaleLimit `json:"width"`
	Head       AvatarScaleLimit `json:"head"`
	Proportion AvatarScaleLimit `json:"proportion"`
	BodyType   AvatarScaleLimit `json:"bodyType"`
}

// AvatarPaletteColor is a BrickColor that the avatar editor offers.
type AvatarPaletteColor struct {
	BrickColorID int64  `json:"brickColorId"`
	Name         string `json:"name"`
	HexColor     string `json:"hexColor"`
}

// AvatarRules is what the account may change on its avatar.
type AvatarRules struct {
	ScaleLimits      AvatarScaleLimits    `json:"scaleLimits"`
	BodyColorPalette []AvatarPaletteColor `json:"bodyColorPalette"`
	// MaxWornCountByTypeName holds every asset type that the account may
	// wear, by the type names of InventoryItem.TypeName.
	MaxWornCountByTypeName map[string]int `json:"maxWornCountByTypeName"`
	// IsBodyShapeEnabled tells whether the account may change the proportion
	// and body type scales.
	IsBodyShapeEnabled bool `json:"isBodyShapeEnabled"`
	// AccessoryAdjustmentLimitsByTypeName holds every accessory type that the
	// account may adjust, by the type names of InventoryItem.TypeName.
	AccessoryAdjustmentLimitsByTypeName map[string]AvatarAccessoryAdjustmentLimits `json:"accessoryAdjustmentLimitsByTypeName"`
}

// AvatarOutfit is an outfit that a user saved.
type AvatarOutfit struct {
	OutfitID int64  `json:"outfitId"`
	Name     string `json:"name"`
	ImageURL string `json:"imageUrl"`
}

// ThumbnailTarget names the image of the outfit.
func (outfit AvatarOutfit) ThumbnailTarget() ThumbnailTarget {
	return AvatarOutfitImageTarget(outfit.OutfitID)
}

// AvatarOutfitImageTarget names the image of an outfit. Roblox renders it in
// the small and the large size. Verified on 2026-11-14.
func AvatarOutfitImageTarget(outfitID int64) ThumbnailTarget {
	return ThumbnailTarget{Type: "Outfit", TargetID: outfitID}
}

type AvatarOutfitPage struct {
	Outfits    []AvatarOutfit `json:"outfits"`
	NextCursor string         `json:"nextCursor"`
}

type listedAvatarAsset struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	AssetType struct {
		Name string `json:"name"`
	} `json:"assetType"`
	// Meta is the layering order, puffiness, and placement of layered
	// clothing and adjusted accessories. A save passes it back unchanged.
	Meta json.RawMessage `json:"meta"`
}

// updateAsset passes the layout back unchanged, and leaves it out when Roblox
// lists none.
func (listed listedAvatarAsset) updateAsset() avatarUpdateAsset {
	asset := avatarUpdateAsset{ID: listed.ID}
	if hasMeta(listed.Meta) {
		asset.Meta = listed.Meta
	}
	return asset
}

type listedBodyColors struct {
	HeadColor     string `json:"headColor3"`
	TorsoColor    string `json:"torsoColor3"`
	LeftArmColor  string `json:"leftArmColor3"`
	RightArmColor string `json:"rightArmColor3"`
	LeftLegColor  string `json:"leftLegColor3"`
	RightLegColor string `json:"rightLegColor3"`
}

func (listed listedBodyColors) bodyColors() AvatarBodyColors {
	return AvatarBodyColors{
		Head: avatarColorOf(listed.HeadColor), Torso: avatarColorOf(listed.TorsoColor),
		LeftArm: avatarColorOf(listed.LeftArmColor), RightArm: avatarColorOf(listed.RightArmColor),
		LeftLeg: avatarColorOf(listed.LeftLegColor), RightLeg: avatarColorOf(listed.RightLegColor),
	}
}

// avatarPayload is an avatar as the v2 avatar API lists it. Verified against
// the API on 2026-11-14.
type avatarPayload struct {
	Scales           *AvatarScales        `json:"scales"`
	PlayerAvatarType AvatarType           `json:"playerAvatarType"`
	BodyColors       *listedBodyColors    `json:"bodyColor3s"`
	Assets           *[]listedAvatarAsset `json:"assets"`
	Emotes           []struct {
		AssetID   int64  `json:"assetId"`
		AssetName string `json:"assetName"`
		Position  int    `json:"position"`
	} `json:"emotes"`
}

// isComplete reports whether the payload lists everything that an avatar
// needs. Emotes are optional, because outfits have none.
func (payload avatarPayload) isComplete() bool {
	return payload.Scales != nil && payload.BodyColors != nil && payload.Assets != nil && validAvatarType(payload.PlayerAvatarType)
}

// Avatar reads how the user looks now. Roblox lists the avatar of any user.
func (service *Users) Avatar(ctx context.Context, accountID, version, userID int64) (Avatar, error) {
	var payload avatarPayload
	address := "https://avatar.roblox.com/v2/avatar/users/" + strconv.FormatInt(userID, 10) + "/avatar"
	if err := service.readAvatarDetails(ctx, accountID, version, "avatar", address, &payload); err != nil {
		return Avatar{}, err
	}
	return avatarOf(payload)
}

func avatarOf(payload avatarPayload) (Avatar, error) {
	if !payload.isComplete() {
		return Avatar{}, invalidAvatar("avatar")
	}
	avatar := Avatar{
		AvatarType: payload.PlayerAvatarType, Scales: *payload.Scales, BodyColors: payload.BodyColors.bodyColors(),
		Assets: make([]AvatarAsset, 0, len(*payload.Assets)), Emotes: make([]AvatarEmote, 0, len(payload.Emotes)),
	}
	for _, listedAsset := range *payload.Assets {
		if listedAsset.ID > 0 {
			adjustment, layer := assetLayoutOf(listedAsset.Meta)
			avatar.Assets = append(avatar.Assets, AvatarAsset{
				Category:   inventoryCategoryOfAssetType(listedAsset.AssetType.Name),
				Item:       avatarItem(listedAsset.ID, listedAsset.Name, listedAsset.AssetType.Name),
				Adjustment: adjustment, Layer: layer,
			})
		}
	}
	for _, listedEmote := range payload.Emotes {
		if listedEmote.AssetID > 0 && listedEmote.Position >= 1 && listedEmote.Position <= maxEmoteSlot {
			avatar.Emotes = append(avatar.Emotes, AvatarEmote{Slot: listedEmote.Position, Item: avatarItem(listedEmote.AssetID, listedEmote.AssetName, "EmoteAnimation")})
		}
	}
	return avatar, nil
}

// inventoryCategoryOfAssetType finds the inventory category that lists the
// asset type, such as accessories for HairAccessory.
func inventoryCategoryOfAssetType(typeName string) InventoryCategory {
	for category, assetTypes := range inventoryAssetTypesByCategory {
		if slices.Contains(assetTypes, typeName) {
			return category
		}
	}
	return ""
}

// AddAvatarCatalogDetails adds the catalog details of each worn asset.
func (service *Users) AddAvatarCatalogDetails(ctx context.Context, accountID, version int64, avatar *Avatar) error {
	items := make([]InventoryItem, len(avatar.Assets))
	for index, asset := range avatar.Assets {
		items[index] = asset.Item
	}
	err := service.AddCatalogDetails(ctx, accountID, version, items)
	for index := range avatar.Assets {
		avatar.Assets[index].Item = items[index]
	}
	return err
}

func avatarItem(assetID int64, name, typeName string) InventoryItem {
	return InventoryItem{
		ItemID: assetID, Name: boundedText(name, maxInventoryItemNameLength), TypeName: boundedText(typeName, maxInventoryItemNameLength),
		thumbnailType: "Asset", CatalogItemType: CatalogAsset,
	}
}

type avatarRulesPayload struct {
	Scales             *AvatarScaleLimits `json:"scales"`
	WearableAssetTypes []struct {
		MaxNumber int    `json:"maxNumber"`
		ID        int64  `json:"id"`
		Name      string `json:"name"`
	} `json:"wearableAssetTypes"`
	BodyColorsPalette []struct {
		BrickColorID int64  `json:"brickColorId"`
		HexColor     string `json:"hexColor"`
		Name         string `json:"name"`
	} `json:"bodyColorsPalette"`
	ProportionsAndBodyTypeEnabledForUser bool                              `json:"proportionsAndBodyTypeEnabledForUser"`
	AccessoryRefinementTypes             []int64                           `json:"accessoryRefinementTypes"`
	AccessoryRefinementLowerBounds       map[string]listedRefinementBounds `json:"accessoryRefinementLowerBounds"`
	AccessoryRefinementUpperBounds       map[string]listedRefinementBounds `json:"accessoryRefinementUpperBounds"`
}

// The rules name asset types with spaces and hyphens, such as "T-Shirt
// Accessory", where the avatar and inventory APIs write TShirtAccessory.
var ruleTypeNameReplacer = strings.NewReplacer(" ", "", "-", "")

func (service *Users) readAvatarRules(ctx context.Context, accountID, version int64, endpoint string) (avatarRulesPayload, error) {
	var payload avatarRulesPayload
	err := service.readAvatarDetails(ctx, accountID, version, endpoint, "https://avatar.roblox.com/v1/avatar-rules", &payload)
	return payload, err
}

// AvatarRules reads what the account may change on its avatar.
func (service *Users) AvatarRules(ctx context.Context, accountID, version int64) (AvatarRules, error) {
	payload, err := service.readAvatarRules(ctx, accountID, version, "avatar-rules")
	if err != nil {
		return AvatarRules{}, err
	}
	if payload.Scales == nil {
		return AvatarRules{}, invalidAvatar("avatar-rules")
	}
	rules := AvatarRules{
		ScaleLimits:                         *payload.Scales,
		BodyColorPalette:                    make([]AvatarPaletteColor, 0, len(payload.BodyColorsPalette)),
		MaxWornCountByTypeName:              make(map[string]int, len(payload.WearableAssetTypes)),
		IsBodyShapeEnabled:                  payload.ProportionsAndBodyTypeEnabledForUser,
		AccessoryAdjustmentLimitsByTypeName: accessoryAdjustmentLimitsOf(payload),
	}
	for _, listedColor := range payload.BodyColorsPalette {
		if hexColor := avatarColorOf(listedColor.HexColor); hexColor != "" {
			rules.BodyColorPalette = append(rules.BodyColorPalette, AvatarPaletteColor{
				BrickColorID: max(listedColor.BrickColorID, 0), Name: boundedText(listedColor.Name, maxInventoryItemNameLength), HexColor: hexColor,
			})
		}
	}
	for _, wearableType := range payload.WearableAssetTypes {
		if wearableType.MaxNumber > 0 {
			rules.MaxWornCountByTypeName[ruleTypeNameReplacer.Replace(wearableType.Name)] = wearableType.MaxNumber
		}
	}
	return rules, nil
}

// TryOnChange returns the avatar of the account with the asset put on, as a
// change that a preview renders.
func (service *Users) TryOnChange(ctx context.Context, accountID, version int64, category InventoryCategory, assetID int64) (AvatarChange, error) {
	const endpoint = "avatar-try-on"
	if !canTryOn(category) {
		return AvatarChange{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "This item can't be tried on."}
	}
	typeName, maxWornCount, err := service.wearableAssetTypeOf(ctx, accountID, version, endpoint, assetID)
	if err != nil {
		return AvatarChange{}, err
	}
	var current avatarPayload
	if err := service.readAvatarDetails(ctx, accountID, version, endpoint, "https://avatar.roblox.com/v2/avatar/avatar", &current); err != nil {
		return AvatarChange{}, err
	}
	avatar, err := avatarOf(current)
	if err != nil {
		return AvatarChange{}, err
	}
	assetIDs := assetIDsWearing(avatar.Assets, assetID, typeName, maxWornCount)
	assets := make([]AvatarChangeAsset, len(assetIDs))
	for index, wornAssetID := range assetIDs {
		assets[index] = AvatarChangeAsset{AssetID: wornAssetID}
	}
	return AvatarChange{AvatarType: avatar.AvatarType, Scales: avatar.Scales, BodyColors: avatar.BodyColors, Assets: assets}, nil
}

// wearableAssetTypeOf reads the type name of the asset and how many assets of
// the type an avatar may wear, or fails when an avatar cannot wear it.
func (service *Users) wearableAssetTypeOf(ctx context.Context, accountID, version int64, endpoint string, assetID int64) (string, int, error) {
	rules, err := service.readAvatarRules(ctx, accountID, version, endpoint)
	if err != nil {
		return "", 0, err
	}
	var assetDetails struct {
		AssetTypeID int64 `json:"AssetTypeId"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, endpoint, "https://economy.roblox.com/v2/assets/"+strconv.FormatInt(assetID, 10)+"/details", &assetDetails); err != nil {
		return "", 0, err
	}
	for _, wearableType := range rules.WearableAssetTypes {
		if wearableType.ID == assetDetails.AssetTypeID && wearableType.MaxNumber > 0 {
			return ruleTypeNameReplacer.Replace(wearableType.Name), wearableType.MaxNumber, nil
		}
	}
	return "", 0, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Roblox doesn't let avatars wear this item."}
}

// assetIDsWearing lists the worn assets with the asset put on. When the avatar
// already wears as many assets of the type as Roblox allows, the asset
// replaces the ones that the avatar lists first.
func assetIDsWearing(wornAssets []AvatarAsset, assetID int64, typeName string, maxWornCount int) []int64 {
	sameTypeCount := 0
	for _, asset := range wornAssets {
		if asset.Item.TypeName == typeName && asset.Item.ItemID != assetID {
			sameTypeCount++
		}
	}
	replacedCount := max(sameTypeCount-maxWornCount+1, 0)
	assetIDs := make([]int64, 0, len(wornAssets)+1)
	for _, asset := range wornAssets {
		isReplaced := asset.Item.TypeName == typeName && asset.Item.ItemID != assetID && replacedCount > 0
		if isReplaced {
			replacedCount--
		}
		if !isReplaced && asset.Item.ItemID != assetID {
			assetIDs = append(assetIDs, asset.Item.ItemID)
		}
	}
	return append(assetIDs, assetID)
}

// SaveAvatar replaces the whole avatar of the account in one request, as the
// Roblox avatar editor does. It returns the assets that Roblox lists as
// invalid. Roblox can also skip an asset that the account cannot wear without
// reporting it. Verified against the API on 2026-11-14.
func (service *Users) SaveAvatar(ctx context.Context, accountID, version int64, change AvatarChange) ([]int64, error) {
	const endpoint = "avatar-save"
	if err := validateAvatarChange(change); err != nil {
		return nil, err
	}
	assets, changedAdjustments, err := service.avatarUpdateAssetsOf(ctx, accountID, version, endpoint, change.Assets)
	if err != nil {
		return nil, err
	}
	if len(changedAdjustments) > 0 {
		// The rules can change between the read that showed the controls and
		// the save, so each changed adjustment is checked against the current
		// rules.
		if err := service.validateAccessoryAdjustments(ctx, accountID, version, endpoint, changedAdjustments); err != nil {
			return nil, err
		}
	}
	emotes := make([]avatarUpdateEmote, len(change.Emotes))
	for index, emote := range change.Emotes {
		emotes[index] = avatarUpdateEmote{AssetID: emote.AssetID, Position: emote.Slot}
	}
	return service.updateAvatar(ctx, accountID, version, endpoint, avatarUpdate{
		AvatarType: change.AvatarType, Scales: change.Scales, BodyColors: change.BodyColors, Assets: assets, Emotes: &emotes,
	})
}

// changedAccessoryAdjustment is an adjustment that differs from the one that
// Roblox stores for a worn accessory.
type changedAccessoryAdjustment struct {
	typeName   string
	adjustment AvatarAccessoryAdjustment
}

// validateAccessoryAdjustments checks that the account may adjust each
// accessory type, and that each axis stays within its current range.
func (service *Users) validateAccessoryAdjustments(ctx context.Context, accountID, version int64, endpoint string, changedAdjustments []changedAccessoryAdjustment) error {
	rulesPayload, err := service.readAvatarRules(ctx, accountID, version, endpoint)
	if err != nil {
		return err
	}
	limitsByTypeName := accessoryAdjustmentLimitsOf(rulesPayload)
	for _, changed := range changedAdjustments {
		limits, isAdjustable := limitsByTypeName[changed.typeName]
		if !isAdjustable {
			return invalidAvatarChange("Roblox doesn't let this account adjust one of the accessories. Reset it and try again.")
		}
		if !limits.contains(changed.adjustment) {
			return invalidAvatarChange("An accessory adjustment is outside the range that Roblox allows. Refresh the avatar and try again.")
		}
	}
	return nil
}

// avatarUpdateAssetsOf lists the assets with the meta that Roblox keeps for
// the ones the account wears, such as the order of layered clothing, so that a
// change keeps it. It writes each changed adjustment into the meta, and lists
// the changed adjustments. A changed adjustment of an asset that the avatar
// does not wear now means that the avatar changed since the page read it.
func (service *Users) avatarUpdateAssetsOf(ctx context.Context, accountID, version int64, endpoint string, changeAssets []AvatarChangeAsset) ([]avatarUpdateAsset, []changedAccessoryAdjustment, error) {
	var current avatarPayload
	if err := service.readAvatarDetails(ctx, accountID, version, endpoint, "https://avatar.roblox.com/v2/avatar/avatar", &current); err != nil {
		return nil, nil, err
	}
	if current.Assets == nil {
		return nil, nil, invalidAvatar(endpoint)
	}
	wornAssetByID := make(map[int64]listedAvatarAsset, len(*current.Assets))
	for _, listedAsset := range *current.Assets {
		wornAssetByID[listedAsset.ID] = listedAsset
	}
	assets := make([]avatarUpdateAsset, len(changeAssets))
	var changedAdjustments []changedAccessoryAdjustment
	for index, changeAsset := range changeAssets {
		wornAsset, isWorn := wornAssetByID[changeAsset.AssetID]
		if !isWorn {
			if changeAsset.Adjustment != nil && !changeAsset.Adjustment.equals(defaultAccessoryAdjustment) {
				return nil, nil, avatarChangedError(endpoint)
			}
			assets[index] = avatarUpdateAsset{ID: changeAsset.AssetID}
			continue
		}
		assets[index] = wornAsset.updateAsset()
		if changeAsset.Adjustment == nil {
			continue
		}
		storedAdjustment, _ := assetLayoutOf(wornAsset.Meta)
		if storedAdjustment == nil {
			storedAdjustment = &defaultAccessoryAdjustment
		}
		if changeAsset.Adjustment.equals(*storedAdjustment) {
			continue
		}
		meta, err := metaWithAdjustment(wornAsset.Meta, *changeAsset.Adjustment)
		if err != nil {
			return nil, nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Roblox returned invalid avatar data.", Cause: err}
		}
		assets[index].Meta = meta
		changedAdjustments = append(changedAdjustments, changedAccessoryAdjustment{
			typeName: ruleTypeNameReplacer.Replace(wornAsset.AssetType.Name), adjustment: *changeAsset.Adjustment,
		})
	}
	return assets, changedAdjustments, nil
}

func avatarChangedError(endpoint string) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "The avatar changed since it was loaded. Refresh it and adjust the accessory again."}
}

// AvatarPreviewFormat is what the render API makes of an avatar, by its
// thumbnail type.
type AvatarPreviewFormat string

const (
	AvatarPreviewImage AvatarPreviewFormat = "2dWebp"
	// AvatarPreviewModel is a 3D model, whose address names a manifest that
	// RenderedModel downloads.
	AvatarPreviewModel AvatarPreviewFormat = "3d"
)

// RenderAvatarPreview asks Roblox to render the change in the format without
// saving it. It returns the address of the render, or an empty address while
// Roblox renders it, until a later call with the same change finds it.
func (service *Users) RenderAvatarPreview(ctx context.Context, accountID, version int64, change AvatarChange, format AvatarPreviewFormat) (string, error) {
	const endpoint = "avatar-preview"
	if err := validateAvatarChange(change); err != nil {
		return "", err
	}
	// A preview leaves the bounds to Roblox, because it saves nothing and
	// repeats until Roblox renders it.
	assets, _, err := service.avatarUpdateAssetsOf(ctx, accountID, version, endpoint, change.Assets)
	if err != nil {
		return "", err
	}
	// The render API takes the avatar in the form of the v1 avatar API, with
	// the depth that the avatar editor derives from the width.
	type renderScales struct {
		AvatarScales

		Depth float64 `json:"depth"`
	}
	type renderAvatarType struct {
		PlayerAvatarType AvatarType `json:"playerAvatarType"`
	}
	type renderDefinition struct {
		Assets           []avatarUpdateAsset `json:"assets"`
		BodyColors       v4BodyColors        `json:"bodyColors"`
		Scales           renderScales        `json:"scales"`
		PlayerAvatarType renderAvatarType    `json:"playerAvatarType"`
	}
	type thumbnailConfig struct {
		ThumbnailID   int    `json:"thumbnailId"`
		ThumbnailType string `json:"thumbnailType"`
		Size          string `json:"size"`
	}
	request := struct {
		ThumbnailConfig  thumbnailConfig  `json:"thumbnailConfig"`
		AvatarDefinition renderDefinition `json:"avatarDefinition"`
	}{
		ThumbnailConfig: thumbnailConfig{ThumbnailID: 3, ThumbnailType: string(format), Size: LargeThumbnailSize},
		AvatarDefinition: renderDefinition{
			Assets: assets, BodyColors: v4BodyColorsOf(change.BodyColors),
			Scales:           renderScales{AvatarScales: change.Scales, Depth: change.Scales.Width},
			PlayerAvatarType: renderAvatarType{PlayerAvatarType: change.AvatarType},
		},
	}
	response, err := service.writeAccount(ctx, accountID, version, endpoint, http.MethodPost, "https://avatar.roblox.com/v1/avatar/render", request)
	if remote, ok := errors.AsType[*roblox.Error](err); ok && remote.Status == http.StatusBadRequest {
		return "", &roblox.Error{Kind: remote.Kind, Endpoint: endpoint, Status: remote.Status, RobloxCode: remote.RobloxCode, Message: "Roblox couldn't render this preview.", Cause: err}
	}
	if err != nil {
		return "", err
	}
	var payload struct {
		State    string `json:"state"`
		ImageURL string `json:"imageUrl"`
	}
	if err := json.Unmarshal(response.Body, &payload); err != nil {
		return "", &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Status: response.Status, Message: "Roblox returned an invalid avatar preview.", Cause: err}
	}
	switch {
	case strings.EqualFold(payload.State, "Pending"):
		return "", nil
	case !strings.EqualFold(payload.State, "Completed") || !safeThumbnailURL(payload.ImageURL):
		return "", &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Status: response.Status, Message: "Roblox couldn't render this preview."}
	}
	return payload.ImageURL, nil
}

// RedrawAvatarImage asks Roblox to render the avatar images of the account
// again. Roblox limits how often it accepts the request.
func (service *Users) RedrawAvatarImage(ctx context.Context, accountID, version int64) error {
	_, err := service.writeAccount(ctx, accountID, version, "avatar-redraw", http.MethodPost, "https://avatar.roblox.com/v1/avatar/redraw-thumbnail", nil)
	return err
}

// outfitPayload is an outfit as the v4 outfit details API lists it, which
// replaces the deprecated v3 API. Roblox lists the outfits of any user, and
// writes the avatar type as a name and the body colors without a leading #.
// Verified against the API on 2026-11-17.
type outfitPayload struct {
	OutfitModel *struct {
		Scale            *AvatarScales        `json:"scale"`
		PlayerAvatarType AvatarType           `json:"playerAvatarType"`
		BodyColors       *v4BodyColors        `json:"bodyColors"`
		Assets           *[]listedAvatarAsset `json:"assets"`
	} `json:"outfitModel"`
}

// readOutfit reads the outfit as an avatar without emotes.
func (service *Users) readOutfit(ctx context.Context, accountID, version int64, endpoint string, outfitID int64) (avatarPayload, error) {
	var payload outfitPayload
	address := "https://avatar.roblox.com/v4/outfits/" + strconv.FormatInt(outfitID, 10) + "/details"
	if err := service.readAvatarDetails(ctx, accountID, version, endpoint, address, &payload); err != nil {
		return avatarPayload{}, err
	}
	if payload.OutfitModel == nil || payload.OutfitModel.BodyColors == nil {
		return avatarPayload{}, invalidAvatar(endpoint)
	}
	model := payload.OutfitModel
	bodyColors := listedBodyColors(*model.BodyColors)
	outfit := avatarPayload{Scales: model.Scale, PlayerAvatarType: model.PlayerAvatarType, BodyColors: &bodyColors, Assets: model.Assets}
	if !outfit.isComplete() {
		return avatarPayload{}, invalidAvatar(endpoint)
	}
	return outfit, nil
}

// AvatarOutfitDetails reads how the outfit looks, as an avatar without emotes.
func (service *Users) AvatarOutfitDetails(ctx context.Context, accountID, version, outfitID int64) (Avatar, error) {
	outfit, err := service.readOutfit(ctx, accountID, version, "avatar-outfit-details", outfitID)
	if err != nil {
		return Avatar{}, err
	}
	return avatarOf(outfit)
}

// WearAvatarOutfit puts the outfit on the avatar of the account. It keeps the
// equipped emotes, because outfits have none.
func (service *Users) WearAvatarOutfit(ctx context.Context, accountID, version, outfitID int64) error {
	outfit, err := service.readOutfit(ctx, accountID, version, "avatar-outfit-wear", outfitID)
	if err != nil {
		return err
	}
	assets := make([]avatarUpdateAsset, 0, len(*outfit.Assets))
	for _, listedAsset := range *outfit.Assets {
		if listedAsset.ID > 0 {
			assets = append(assets, listedAsset.updateAsset())
		}
	}
	// The page reads the avatar again after the outfit is put on, which shows
	// any asset that Roblox lists as invalid.
	_, err = service.updateAvatar(ctx, accountID, version, "avatar-outfit-wear", avatarUpdate{
		AvatarType: outfit.PlayerAvatarType, Scales: *outfit.Scales, BodyColors: outfit.BodyColors.bodyColors(), Assets: assets,
	})
	return err
}

type avatarUpdateAsset struct {
	ID   int64           `json:"id"`
	Meta json.RawMessage `json:"meta,omitempty"`
}

type avatarUpdateEmote struct {
	AssetID  int64 `json:"assetId"`
	Position int   `json:"position"`
}

// avatarUpdate is the avatar that an update writes. Emotes is nil when the
// update keeps the equipped emotes.
type avatarUpdate struct {
	AvatarType AvatarType
	Scales     AvatarScales
	BodyColors AvatarBodyColors
	Assets     []avatarUpdateAsset
	Emotes     *[]avatarUpdateEmote
}

type avatarModelUpdate struct {
	Scales           AvatarScales        `json:"scales"`
	PlayerAvatarType AvatarType          `json:"playerAvatarType"`
	BodyColors       v4BodyColors        `json:"bodyColors"`
	Assets           []avatarUpdateAsset `json:"assets"`
}

// v4BodyColors are the body colors as the v4 avatar and outfit APIs read and
// write them.
type v4BodyColors struct {
	HeadColor     string `json:"headColor"`
	TorsoColor    string `json:"torsoColor"`
	LeftArmColor  string `json:"leftArmColor"`
	RightArmColor string `json:"rightArmColor"`
	LeftLegColor  string `json:"leftLegColor"`
	RightLegColor string `json:"rightLegColor"`
}

func v4BodyColorsOf(colors AvatarBodyColors) v4BodyColors {
	return v4BodyColors{
		HeadColor: colors.Head, TorsoColor: colors.Torso, LeftArmColor: colors.LeftArm,
		RightArmColor: colors.RightArm, LeftLegColor: colors.LeftLeg, RightLegColor: colors.RightLeg,
	}
}

// updateAvatar writes the avatar with the v4 avatar API, which names the parts
// that a request changes in updateTypes, and returns the assets that Roblox
// lists as invalid. Verified on 2026-11-14.
func (service *Users) updateAvatar(ctx context.Context, accountID, version int64, endpoint string, update avatarUpdate) ([]int64, error) {
	type avatarConfigUpdate struct {
		EmoteRequestModels []avatarUpdateEmote `json:"emoteRequestModels"`
	}
	type avatarDefinition struct {
		UpdateAvatarModel  avatarModelUpdate   `json:"updateAvatarModel"`
		UpdateAvatarConfig *avatarConfigUpdate `json:"updateAvatarConfig,omitempty"`
	}
	request := struct {
		UpdateTypes      []string         `json:"updateTypes"`
		AvatarDefinition avatarDefinition `json:"avatarDefinition"`
	}{
		UpdateTypes: []string{"UpdateAvatarType", "UpdateScales", "UpdateBodyColors", "UpdateAssets"},
		AvatarDefinition: avatarDefinition{UpdateAvatarModel: avatarModelUpdate{
			Scales: update.Scales, PlayerAvatarType: update.AvatarType, BodyColors: v4BodyColorsOf(update.BodyColors), Assets: update.Assets,
		}},
	}
	if update.Emotes != nil {
		request.UpdateTypes = append(request.UpdateTypes, "UpdateEmotes")
		request.AvatarDefinition.UpdateAvatarConfig = &avatarConfigUpdate{EmoteRequestModels: *update.Emotes}
	}
	response, err := service.writeAccount(ctx, accountID, version, endpoint, http.MethodPatch, "https://avatar.roblox.com/v4/avatar", request)
	if remote, ok := errors.AsType[*roblox.Error](err); ok && remote.Status == http.StatusBadRequest {
		return nil, &roblox.Error{Kind: remote.Kind, Endpoint: endpoint, Status: remote.Status, RobloxCode: remote.RobloxCode, Message: "Roblox rejected these avatar settings. Refresh the avatar and try again.", Cause: err}
	}
	if err != nil {
		return nil, err
	}
	// Roblox lists the assets that it did not put on in validation, and also
	// at the top level while it moves the field. success is false when it
	// skipped any part of the change.
	type invalidAsset struct {
		ID int64 `json:"id"`
	}
	var payload struct {
		Success       bool           `json:"success"`
		InvalidAssets []invalidAsset `json:"invalidAssets"`
		Validation    struct {
			InvalidAssets []invalidAsset `json:"invalidAssets"`
		} `json:"validation"`
	}
	if err := json.Unmarshal(response.Body, &payload); err != nil {
		return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Status: response.Status, Message: "Roblox did not save the avatar. Try again.", Cause: err}
	}
	rejectedAssetIDs := []int64{}
	for _, rejected := range slices.Concat(payload.InvalidAssets, payload.Validation.InvalidAssets) {
		if rejected.ID > 0 && !slices.Contains(rejectedAssetIDs, rejected.ID) {
			rejectedAssetIDs = append(rejectedAssetIDs, rejected.ID)
		}
	}
	if !payload.Success && len(rejectedAssetIDs) == 0 {
		return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Status: response.Status, Message: "Roblox did not save the avatar. Try again."}
	}
	return rejectedAssetIDs, nil
}

// AvatarOutfits reads one page of the outfits that the user created. Roblox
// lists them for any user. Verified against the API on 2026-11-14.
func (service *Users) AvatarOutfits(ctx context.Context, accountID, version, userID int64, cursor string) (AvatarOutfitPage, error) {
	if len(cursor) > maxCursorLength {
		return AvatarOutfitPage{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "avatar-outfits", Message: "This page has expired. Refresh the outfits."}
	}
	query := url.Values{"outfitType": {"Avatar"}, "itemsPerPage": {strconv.Itoa(outfitPageSize)}, "isEditable": {"true"}}
	if cursor != "" {
		query.Set("paginationToken", cursor)
	}
	var payload struct {
		Data *[]struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
		PaginationToken string `json:"paginationToken"`
	}
	address := "https://avatar.roblox.com/v2/avatar/users/" + strconv.FormatInt(userID, 10) + "/outfits?" + query.Encode()
	if err := service.readAvatarDetails(ctx, accountID, version, "avatar-outfits", address, &payload); err != nil {
		return AvatarOutfitPage{}, err
	}
	if payload.Data == nil || len(*payload.Data) > outfitPageSize || len(payload.PaginationToken) > maxCursorLength {
		return AvatarOutfitPage{}, invalidAvatar("avatar-outfits")
	}
	page := AvatarOutfitPage{Outfits: make([]AvatarOutfit, 0, len(*payload.Data)), NextCursor: payload.PaginationToken}
	for _, listedOutfit := range *payload.Data {
		if listedOutfit.ID > 0 {
			page.Outfits = append(page.Outfits, AvatarOutfit{OutfitID: listedOutfit.ID, Name: boundedText(listedOutfit.Name, maxOutfitNameLength)})
		}
	}
	return page, nil
}

// CreateAvatarOutfit saves the current avatar of the account as an outfit.
func (service *Users) CreateAvatarOutfit(ctx context.Context, accountID, version int64, name string) error {
	name, err := validOutfitName(name, "avatar-outfit-create")
	if err != nil {
		return err
	}
	var current avatarPayload
	if err := service.readAvatarDetails(ctx, accountID, version, "avatar-outfit-create", "https://avatar.roblox.com/v2/avatar/avatar", &current); err != nil {
		return err
	}
	if !current.isComplete() {
		return invalidAvatar("avatar-outfit-create")
	}
	assets := make([]avatarUpdateAsset, len(*current.Assets))
	for index, listedAsset := range *current.Assets {
		assets[index] = listedAsset.updateAsset()
	}
	type outfitModel struct {
		Name             string              `json:"name"`
		Scale            AvatarScales        `json:"scale"`
		PlayerAvatarType AvatarType          `json:"playerAvatarType"`
		BodyColors       v4BodyColors        `json:"bodyColors"`
		Assets           []avatarUpdateAsset `json:"assets"`
	}
	type outfitDefinition struct {
		UpdateOutfitModel outfitModel `json:"updateOutfitModel"`
	}
	request := struct {
		OutfitDefinition outfitDefinition `json:"outfitDefinition"`
	}{OutfitDefinition: outfitDefinition{UpdateOutfitModel: outfitModel{
		Name: name, Scale: *current.Scales, PlayerAvatarType: current.PlayerAvatarType,
		BodyColors: v4BodyColorsOf(current.BodyColors.bodyColors()), Assets: assets,
	}}}
	_, err = service.writeAccount(ctx, accountID, version, "avatar-outfit-create", http.MethodPost, "https://avatar.roblox.com/v4/outfits/create", request)
	return outfitWriteError(err, "avatar-outfit-create")
}

func (service *Users) RenameAvatarOutfit(ctx context.Context, accountID, version, outfitID int64, name string) error {
	name, err := validOutfitName(name, "avatar-outfit-rename")
	if err != nil {
		return err
	}
	type outfitModel struct {
		Name string `json:"name"`
	}
	type outfitDefinition struct {
		UpdateOutfitModel outfitModel `json:"updateOutfitModel"`
	}
	request := struct {
		UpdateTypes      []string         `json:"updateTypes"`
		OutfitDefinition outfitDefinition `json:"outfitDefinition"`
	}{UpdateTypes: []string{"UpdateName"}, OutfitDefinition: outfitDefinition{UpdateOutfitModel: outfitModel{Name: name}}}
	_, err = service.writeAccount(ctx, accountID, version, "avatar-outfit-rename", http.MethodPatch, "https://avatar.roblox.com/v4/outfits/"+strconv.FormatInt(outfitID, 10), request)
	return outfitWriteError(err, "avatar-outfit-rename")
}

func (service *Users) DeleteAvatarOutfit(ctx context.Context, accountID, version, outfitID int64) error {
	_, err := service.writeAccount(ctx, accountID, version, "avatar-outfit-delete", http.MethodPost, "https://avatar.roblox.com/v1/outfits/"+strconv.FormatInt(outfitID, 10)+"/delete", nil)
	return err
}

// outfitWriteError explains the outfit errors that the Roblox avatar editor
// names: code 1 for too many outfits, and code 4 on create or code 0 on rename
// for a name that Roblox rejects.
func outfitWriteError(err error, endpoint string) error {
	remote, ok := errors.AsType[*roblox.Error](err)
	if !ok || remote.Status != http.StatusBadRequest {
		return err
	}
	var message string
	switch {
	case remote.RobloxCode == 1:
		message = "This account has the most outfits that Roblox allows. Delete an outfit first."
	case (endpoint == "avatar-outfit-create" && remote.RobloxCode == 4) || (endpoint == "avatar-outfit-rename" && remote.RobloxCode == 0):
		message = "Roblox rejected this outfit name. Try another name."
	default:
		return err
	}
	return &roblox.Error{Kind: remote.Kind, Endpoint: endpoint, Status: remote.Status, RobloxCode: remote.RobloxCode, Message: message, Cause: err}
}

func validOutfitName(name, endpoint string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || !utf8.ValidString(name) || utf8.RuneCountInString(name) > maxOutfitNameLength {
		return "", &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Enter an outfit name of up to 100 characters."}
	}
	return name, nil
}

func validateAvatarChange(change AvatarChange) error {
	if !validAvatarType(change.AvatarType) {
		return invalidAvatarChange("Choose R6 or R15.")
	}
	for _, scale := range []float64{change.Scales.Height, change.Scales.Width, change.Scales.Head, change.Scales.Proportion, change.Scales.BodyType} {
		if math.IsNaN(scale) || scale < 0 || scale > 2 {
			return invalidAvatarChange("A body scale is out of range. Refresh the avatar and try again.")
		}
	}
	colors := change.BodyColors
	for _, color := range []string{colors.Head, colors.Torso, colors.LeftArm, colors.RightArm, colors.LeftLeg, colors.RightLeg} {
		if avatarColorOf(color) != color || color == "" {
			return invalidAvatarChange("Enter each body color as 6 hex digits, such as A3A2A5.")
		}
	}
	return validateAvatarItems(change)
}

// validateAvatarItems checks the worn assets, their adjustments, and the emote
// slots of a change.
func validateAvatarItems(change AvatarChange) error {
	if len(change.Assets) > maxWornAssetCount {
		return invalidAvatarChange("The avatar wears too many items.")
	}
	for _, asset := range change.Assets {
		if asset.AssetID <= 0 {
			return invalidAvatarChange("An item on the avatar is invalid. Refresh the avatar and try again.")
		}
		if asset.Adjustment != nil && !asset.Adjustment.isFinite() {
			return invalidAvatarChange("An accessory adjustment is invalid. Reset it and try again.")
		}
	}
	isSlotTaken := make(map[int]bool, len(change.Emotes))
	for _, emote := range change.Emotes {
		if emote.AssetID <= 0 || emote.Slot < 1 || emote.Slot > maxEmoteSlot || isSlotTaken[emote.Slot] {
			return invalidAvatarChange("An emote slot is invalid. Refresh the avatar and try again.")
		}
		isSlotTaken[emote.Slot] = true
	}
	return nil
}

func invalidAvatarChange(message string) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "avatar-save", Message: message}
}

func validAvatarType(avatarType AvatarType) bool {
	return avatarType == AvatarR6 || avatarType == AvatarR15
}

// avatarColorOf writes a Roblox color as six uppercase hex digits. Roblox
// stores the case it receives and accepts a leading #, and the rules list
// palette colors with one. Verified on 2026-11-14.
func avatarColorOf(rawColor string) string {
	hexColor := strings.ToUpper(strings.TrimPrefix(strings.TrimSpace(rawColor), "#"))
	if len(hexColor) != 6 || strings.Trim(hexColor, "0123456789ABCDEF") != "" {
		return ""
	}
	return hexColor
}

func (service *Users) readAvatarDetails(ctx context.Context, accountID, version int64, endpoint, address string, target any) error {
	endpointURL, _ := url.Parse(address)
	return service.readLargeAccountDetails(ctx, accountID, version, endpoint, endpointURL, target, "Roblox returned invalid avatar data.")
}

func invalidAvatar(endpoint string) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Roblox returned invalid avatar data."}
}
