package services

import (
	"bytes"
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
	maxWornAssetCount     = 100
	maxEmoteSlot          = 8
	maxOutfitNameLength   = 100
	maxOutfitCursorLength = 4096
	outfitPageSize        = 50
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

// AvatarChange is the whole avatar that an account saves.
type AvatarChange struct {
	AvatarType AvatarType        `json:"avatarType"`
	Scales     AvatarScales      `json:"scales"`
	BodyColors AvatarBodyColors  `json:"bodyColors"`
	AssetIDs   []int64           `json:"assetIds"`
	Emotes     []AvatarEmoteSlot `json:"emotes"`
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
}

// AvatarOutfit is an outfit that a user saved.
type AvatarOutfit struct {
	OutfitID int64  `json:"outfitId"`
	Name     string `json:"name"`
	ImageURL string `json:"imageUrl"`
}

// ThumbnailTarget names the image of the outfit.
func (outfit AvatarOutfit) ThumbnailTarget() ThumbnailTarget {
	return ThumbnailTarget{Type: "Outfit", TargetID: outfit.OutfitID}
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
	if trimmedMeta := bytes.TrimSpace(listed.Meta); len(trimmedMeta) > 0 && !bytes.Equal(trimmedMeta, []byte("null")) {
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
	if payload.Scales == nil || payload.BodyColors == nil || payload.Assets == nil || !validAvatarType(payload.PlayerAvatarType) {
		return Avatar{}, invalidAvatar("avatar")
	}
	avatar := Avatar{
		AvatarType: payload.PlayerAvatarType, Scales: *payload.Scales, BodyColors: payload.BodyColors.bodyColors(),
		Assets: make([]AvatarAsset, 0, len(*payload.Assets)), Emotes: make([]AvatarEmote, 0, len(payload.Emotes)),
	}
	for _, listedAsset := range *payload.Assets {
		if listedAsset.ID > 0 {
			avatar.Assets = append(avatar.Assets, AvatarAsset{
				Category: inventoryCategoryOfAssetType(listedAsset.AssetType.Name),
				Item:     avatarItem(listedAsset.ID, listedAsset.Name, listedAsset.AssetType.Name),
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

// AvatarRules reads what the account may change on its avatar.
func (service *Users) AvatarRules(ctx context.Context, accountID, version int64) (AvatarRules, error) {
	type listedScaleLimit struct {
		Min       float64 `json:"min"`
		Max       float64 `json:"max"`
		Increment float64 `json:"increment"`
	}
	var payload struct {
		Scales *struct {
			Height     listedScaleLimit `json:"height"`
			Width      listedScaleLimit `json:"width"`
			Head       listedScaleLimit `json:"head"`
			Proportion listedScaleLimit `json:"proportion"`
			BodyType   listedScaleLimit `json:"bodyType"`
		} `json:"scales"`
		WearableAssetTypes []struct {
			MaxNumber int    `json:"maxNumber"`
			Name      string `json:"name"`
		} `json:"wearableAssetTypes"`
		BodyColorsPalette []struct {
			BrickColorID int64  `json:"brickColorId"`
			HexColor     string `json:"hexColor"`
			Name         string `json:"name"`
		} `json:"bodyColorsPalette"`
		ProportionsAndBodyTypeEnabledForUser bool `json:"proportionsAndBodyTypeEnabledForUser"`
	}
	if err := service.readAvatarDetails(ctx, accountID, version, "avatar-rules", "https://avatar.roblox.com/v1/avatar-rules", &payload); err != nil {
		return AvatarRules{}, err
	}
	if payload.Scales == nil {
		return AvatarRules{}, invalidAvatar("avatar-rules")
	}
	scaleLimitOf := func(listed listedScaleLimit) AvatarScaleLimit {
		return AvatarScaleLimit(listed)
	}
	rules := AvatarRules{
		ScaleLimits: AvatarScaleLimits{
			Height: scaleLimitOf(payload.Scales.Height), Width: scaleLimitOf(payload.Scales.Width), Head: scaleLimitOf(payload.Scales.Head),
			Proportion: scaleLimitOf(payload.Scales.Proportion), BodyType: scaleLimitOf(payload.Scales.BodyType),
		},
		BodyColorPalette:       make([]AvatarPaletteColor, 0, len(payload.BodyColorsPalette)),
		MaxWornCountByTypeName: make(map[string]int, len(payload.WearableAssetTypes)),
		IsBodyShapeEnabled:     payload.ProportionsAndBodyTypeEnabledForUser,
	}
	for _, listedColor := range payload.BodyColorsPalette {
		if hexColor := avatarColorOf(listedColor.HexColor); hexColor != "" {
			rules.BodyColorPalette = append(rules.BodyColorPalette, AvatarPaletteColor{
				BrickColorID: max(listedColor.BrickColorID, 0), Name: boundedText(listedColor.Name, maxInventoryItemNameLength), HexColor: hexColor,
			})
		}
	}
	// The rules name asset types with spaces and hyphens, such as "T-Shirt
	// Accessory", where the avatar and inventory APIs write TShirtAccessory.
	typeNameReplacer := strings.NewReplacer(" ", "", "-", "")
	for _, wearableType := range payload.WearableAssetTypes {
		if wearableType.MaxNumber > 0 {
			rules.MaxWornCountByTypeName[typeNameReplacer.Replace(wearableType.Name)] = wearableType.MaxNumber
		}
	}
	return rules, nil
}

// SaveAvatar replaces the whole avatar of the account in one request, as the
// Roblox avatar editor does. Roblox skips an asset that the account cannot
// wear without reporting it. Verified against the API on 2026-11-14.
func (service *Users) SaveAvatar(ctx context.Context, accountID, version int64, change AvatarChange) error {
	if err := validateAvatarChange(change); err != nil {
		return err
	}
	var current avatarPayload
	if err := service.readAvatarDetails(ctx, accountID, version, "avatar-save", "https://avatar.roblox.com/v2/avatar/avatar", &current); err != nil {
		return err
	}
	if current.Assets == nil {
		return invalidAvatar("avatar-save")
	}
	wornAssetByID := make(map[int64]avatarUpdateAsset, len(*current.Assets))
	for _, listedAsset := range *current.Assets {
		wornAssetByID[listedAsset.ID] = listedAsset.updateAsset()
	}
	assets := make([]avatarUpdateAsset, len(change.AssetIDs))
	for index, assetID := range change.AssetIDs {
		if wornAsset, isWorn := wornAssetByID[assetID]; isWorn {
			assets[index] = wornAsset
		} else {
			assets[index] = avatarUpdateAsset{ID: assetID}
		}
	}
	emotes := make([]avatarUpdateEmote, len(change.Emotes))
	for index, emote := range change.Emotes {
		emotes[index] = avatarUpdateEmote{AssetID: emote.AssetID, Position: emote.Slot}
	}
	return service.updateAvatar(ctx, accountID, version, "avatar-save", avatarUpdate{
		AvatarType: change.AvatarType, Scales: change.Scales, BodyColors: change.BodyColors, Assets: assets, Emotes: &emotes,
	})
}

// WearAvatarOutfit puts the outfit on the avatar of the account. It keeps the
// equipped emotes, because outfits have none.
func (service *Users) WearAvatarOutfit(ctx context.Context, accountID, version, outfitID int64) error {
	var payload struct {
		Scale            *AvatarScales        `json:"scale"`
		PlayerAvatarType AvatarType           `json:"playerAvatarType"`
		BodyColors       *listedBodyColors    `json:"bodyColor3s"`
		Assets           *[]listedAvatarAsset `json:"assets"`
	}
	address := "https://avatar.roblox.com/v3/outfits/" + strconv.FormatInt(outfitID, 10) + "/details"
	if err := service.readAvatarDetails(ctx, accountID, version, "avatar-outfit-wear", address, &payload); err != nil {
		return err
	}
	if payload.Scale == nil || payload.BodyColors == nil || payload.Assets == nil || !validAvatarType(payload.PlayerAvatarType) {
		return invalidAvatar("avatar-outfit-wear")
	}
	assets := make([]avatarUpdateAsset, 0, len(*payload.Assets))
	for _, listedAsset := range *payload.Assets {
		if listedAsset.ID > 0 {
			assets = append(assets, listedAsset.updateAsset())
		}
	}
	return service.updateAvatar(ctx, accountID, version, "avatar-outfit-wear", avatarUpdate{
		AvatarType: payload.PlayerAvatarType, Scales: *payload.Scale, BodyColors: payload.BodyColors.bodyColors(), Assets: assets,
	})
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
	BodyColors       avatarColorsUpdate  `json:"bodyColors"`
	Assets           []avatarUpdateAsset `json:"assets"`
}

type avatarColorsUpdate struct {
	HeadColor     string `json:"headColor"`
	TorsoColor    string `json:"torsoColor"`
	LeftArmColor  string `json:"leftArmColor"`
	RightArmColor string `json:"rightArmColor"`
	LeftLegColor  string `json:"leftLegColor"`
	RightLegColor string `json:"rightLegColor"`
}

func avatarColorsUpdateOf(colors AvatarBodyColors) avatarColorsUpdate {
	return avatarColorsUpdate{
		HeadColor: colors.Head, TorsoColor: colors.Torso, LeftArmColor: colors.LeftArm,
		RightArmColor: colors.RightArm, LeftLegColor: colors.LeftLeg, RightLegColor: colors.RightLeg,
	}
}

// updateAvatar writes the avatar with the v4 avatar API, which names the parts
// that a request changes in updateTypes. Verified on 2026-11-14.
func (service *Users) updateAvatar(ctx context.Context, accountID, version int64, endpoint string, update avatarUpdate) error {
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
			Scales: update.Scales, PlayerAvatarType: update.AvatarType, BodyColors: avatarColorsUpdateOf(update.BodyColors), Assets: update.Assets,
		}},
	}
	if update.Emotes != nil {
		request.UpdateTypes = append(request.UpdateTypes, "UpdateEmotes")
		request.AvatarDefinition.UpdateAvatarConfig = &avatarConfigUpdate{EmoteRequestModels: *update.Emotes}
	}
	response, err := service.writeAccount(ctx, accountID, version, endpoint, http.MethodPatch, "https://avatar.roblox.com/v4/avatar", request)
	if remote, ok := errors.AsType[*roblox.Error](err); ok && remote.Status == http.StatusBadRequest {
		return &roblox.Error{Kind: remote.Kind, Endpoint: endpoint, Status: remote.Status, RobloxCode: remote.RobloxCode, Message: "Roblox rejected these avatar settings. Refresh the avatar and try again.", Cause: err}
	}
	if err != nil {
		return err
	}
	var payload struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(response.Body, &payload); err != nil || !payload.Success {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Status: response.Status, Message: "Roblox did not save the avatar. Try again.", Cause: err}
	}
	return nil
}

// AvatarOutfits reads one page of the outfits that the user created. Roblox
// lists them for any user. Verified against the API on 2026-11-14.
func (service *Users) AvatarOutfits(ctx context.Context, accountID, version, userID int64, cursor string) (AvatarOutfitPage, error) {
	if len(cursor) > maxOutfitCursorLength {
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
	if payload.Data == nil || len(*payload.Data) > outfitPageSize || len(payload.PaginationToken) > maxOutfitCursorLength {
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
	if current.Scales == nil || current.BodyColors == nil || current.Assets == nil || !validAvatarType(current.PlayerAvatarType) {
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
		BodyColors       avatarColorsUpdate  `json:"bodyColors"`
		Assets           []avatarUpdateAsset `json:"assets"`
	}
	type outfitDefinition struct {
		UpdateOutfitModel outfitModel `json:"updateOutfitModel"`
	}
	request := struct {
		OutfitDefinition outfitDefinition `json:"outfitDefinition"`
	}{OutfitDefinition: outfitDefinition{UpdateOutfitModel: outfitModel{
		Name: name, Scale: *current.Scales, PlayerAvatarType: current.PlayerAvatarType,
		BodyColors: avatarColorsUpdateOf(current.BodyColors.bodyColors()), Assets: assets,
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

// validateAvatarItems checks the worn assets and the emote slots of a change.
func validateAvatarItems(change AvatarChange) error {
	if len(change.AssetIDs) > maxWornAssetCount {
		return invalidAvatarChange("The avatar wears too many items.")
	}
	for _, assetID := range change.AssetIDs {
		if assetID <= 0 {
			return invalidAvatarChange("An item on the avatar is invalid. Refresh the avatar and try again.")
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
