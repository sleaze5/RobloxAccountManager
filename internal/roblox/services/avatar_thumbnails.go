package services

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

// AvatarThumbnailType is the avatar image that a thumbnail customization
// frames. Closeup frames the headshot, and FullBody frames the full-body
// image. Roblox always frames the bust image itself. Verified against the API
// on 2026-11-19.
type AvatarThumbnailType string

const (
	AvatarThumbnailCloseup  AvatarThumbnailType = "Closeup"
	AvatarThumbnailFullBody AvatarThumbnailType = "FullBody"
)

// robloxThumbnailTypeByType holds the numbers of the Roblox
// AvatarThumbnailCustomizationType enum, which the avatar API writes. Roblox
// saves any number above 2 into the FullBody customization, so only these two
// are sent. Verified against the API on 2026-11-19.
var robloxThumbnailTypeByType = map[AvatarThumbnailType]int{
	AvatarThumbnailCloseup:  1,
	AvatarThumbnailFullBody: 2,
}

// AvatarThumbnailCamera is where the camera of a customized thumbnail stands.
// DistanceScale scales the distance of the close-up camera, and is always 1
// for the full-body camera.
type AvatarThumbnailCamera struct {
	FieldOfViewDegrees float64 `json:"fieldOfViewDegrees"`
	YRotationDegrees   float64 `json:"yRotationDegrees"`
	DistanceScale      float64 `json:"distanceScale"`
}

type AvatarThumbnailCameraLimits struct {
	FieldOfViewDegrees AvatarScaleLimit `json:"fieldOfViewDegrees"`
	YRotationDegrees   AvatarScaleLimit `json:"yRotationDegrees"`
	DistanceScale      AvatarScaleLimit `json:"distanceScale"`
}

// The camera limits are the ranges outside of which Roblox rejects a save with
// the codes 5, 6, and 7. The avatar API lists no limits. The full-body camera
// keeps its distance, and a distanceScale of exactly -1 turns a camera off.
// RoSeal uses the same limits, while RoAvatar keeps the close-up distance
// under 2.5. Verified against the API on 2026-11-19.
var (
	closeupCameraLimits = AvatarThumbnailCameraLimits{
		FieldOfViewDegrees: AvatarScaleLimit{Min: 15, Max: 45, Increment: 1},
		YRotationDegrees:   AvatarScaleLimit{Min: -60, Max: 60, Increment: 1},
		DistanceScale:      AvatarScaleLimit{Min: 0.5, Max: 4, Increment: 0.05},
	}
	fullBodyCameraLimits = AvatarThumbnailCameraLimits{
		FieldOfViewDegrees: closeupCameraLimits.FieldOfViewDegrees,
		YRotationDegrees:   closeupCameraLimits.YRotationDegrees,
		DistanceScale:      AvatarScaleLimit{Min: 1, Max: 1, Increment: 1},
	}
	// defaultThumbnailCamera is the camera that Roblox lists for a thumbnail
	// that was never customized, as RoAvatar resets it.
	defaultThumbnailCamera = AvatarThumbnailCamera{FieldOfViewDegrees: 30, YRotationDegrees: 0, DistanceScale: 1}
)

// cameraRangeErrorCodes are the codes with which Roblox rejects the rotation,
// the field of view, and the distance of a camera. It rejects an emote that
// the account does not own with code 0.
var cameraRangeErrorCodes = []int{5, 6, 7}

func cameraLimitsOf(thumbnailType AvatarThumbnailType) AvatarThumbnailCameraLimits {
	if thumbnailType == AvatarThumbnailFullBody {
		return fullBodyCameraLimits
	}
	return closeupCameraLimits
}

// AvatarThumbnailCustomization is how Roblox poses and frames one avatar
// image.
type AvatarThumbnailCustomization struct {
	Type AvatarThumbnailType `json:"type"`
	// Emote is the emote or animation that poses the avatar, or nil when the
	// avatar stands in its idle pose.
	Emote *InventoryItem `json:"emote"`
	// IsCameraCustomized is false while Roblox frames the image with its
	// standard camera. Camera then holds the values that the customized
	// camera starts from.
	IsCameraCustomized bool                        `json:"isCameraCustomized"`
	Camera             AvatarThumbnailCamera       `json:"camera"`
	CameraLimits       AvatarThumbnailCameraLimits `json:"cameraLimits"`
	// IsPoseShownOnlyWithCustomCamera tells that the image ignores the emote
	// while the standard camera frames it, as the headshot does. Roblox still
	// stores the emote. Verified against the API on 2026-11-19.
	IsPoseShownOnlyWithCustomCamera bool `json:"isPoseShownOnlyWithCustomCamera"`
}

// AvatarThumbnails are the customizations of the avatar images of a user,
// with the headshot that the close-up customization frames and the bust
// image, which has no customization. An image is empty while Roblox renders
// it.
type AvatarThumbnails struct {
	Closeup          AvatarThumbnailCustomization `json:"closeup"`
	FullBody         AvatarThumbnailCustomization `json:"fullBody"`
	HeadshotImageURL string                       `json:"headshotImageUrl"`
	BustImageURL     string                       `json:"bustImageUrl"`
}

// AvatarThumbnailChange replaces one customization of the account. An
// EmoteAssetID of 0 removes the emote.
type AvatarThumbnailChange struct {
	Type               AvatarThumbnailType   `json:"type"`
	EmoteAssetID       int64                 `json:"emoteAssetId"`
	IsCameraCustomized bool                  `json:"isCameraCustomized"`
	Camera             AvatarThumbnailCamera `json:"camera"`
}

// AvatarHeadshotImageTarget and AvatarBustImageTarget name the close-up
// images of the user's avatar.
func AvatarHeadshotImageTarget(userID int64) ThumbnailTarget {
	return ThumbnailTarget{Type: "AvatarHeadShot", TargetID: userID}
}

func AvatarBustImageTarget(userID int64) ThumbnailTarget {
	return ThumbnailTarget{Type: "AvatarBust", TargetID: userID}
}

// listedThumbnailCustomization is a customization as the avatar API lists it.
// A negative distanceScale turns the customized camera off. Verified against
// the API on 2026-11-18.
type listedThumbnailCustomization struct {
	ThumbnailType int   `json:"thumbnailType"`
	EmoteAssetID  int64 `json:"emoteAssetId"`
	Camera        struct {
		FieldOfViewDegrees float64 `json:"fieldOfViewDeg"`
		YRotationDegrees   float64 `json:"yRotDeg"`
		DistanceScale      float64 `json:"distanceScale"`
	} `json:"camera"`
}

// AvatarThumbnailCustomizations reads how Roblox poses and frames the avatar
// images of the user. The v4 avatar API lists them for any user with the
// selection type 4. Emotes carry their asset ID only. Verified against the
// API on 2026-11-18.
func (service *Users) AvatarThumbnailCustomizations(ctx context.Context, accountID, version, userID int64) (AvatarThumbnails, error) {
	const endpoint = "avatar-thumbnails"
	var payload struct {
		AvatarConfigurations *struct {
			ThumbnailCustomizations []listedThumbnailCustomization `json:"thumbnailCustomizations"`
		} `json:"avatarConfigurations"`
	}
	address := "https://avatar.roblox.com/v4/avatar/users/" + strconv.FormatInt(userID, 10) + "?selectionTypes=4"
	if err := service.readAvatarDetails(ctx, accountID, version, endpoint, address, &payload); err != nil {
		return AvatarThumbnails{}, err
	}
	if payload.AvatarConfigurations == nil {
		return AvatarThumbnails{}, invalidAvatar(endpoint)
	}
	listed := payload.AvatarConfigurations.ThumbnailCustomizations
	return AvatarThumbnails{
		Closeup:  thumbnailCustomizationOf(AvatarThumbnailCloseup, listed),
		FullBody: thumbnailCustomizationOf(AvatarThumbnailFullBody, listed),
	}, nil
}

// thumbnailCustomizationOf finds the customization of the type, or the
// standard one when Roblox lists none. A camera that is off keeps the
// values that Roblox stores, within the limits, so that turning it on starts
// from them.
func thumbnailCustomizationOf(thumbnailType AvatarThumbnailType, listed []listedThumbnailCustomization) AvatarThumbnailCustomization {
	limits := cameraLimitsOf(thumbnailType)
	customization := AvatarThumbnailCustomization{
		Type: thumbnailType, Camera: defaultThumbnailCamera, CameraLimits: limits,
		IsPoseShownOnlyWithCustomCamera: thumbnailType == AvatarThumbnailCloseup,
	}
	listedIndex := slices.IndexFunc(listed, func(entry listedThumbnailCustomization) bool {
		return entry.ThumbnailType == robloxThumbnailTypeByType[thumbnailType]
	})
	if listedIndex < 0 {
		return customization
	}
	entry := listed[listedIndex]
	if entry.EmoteAssetID > 0 {
		// Most thumbnails pose with an emote. The asset details replace the
		// type when the customization uses an animation instead.
		emote := avatarItem(entry.EmoteAssetID, "", "EmoteAnimation")
		customization.Emote = &emote
	}
	customization.IsCameraCustomized = entry.Camera.DistanceScale >= 0
	customization.Camera = AvatarThumbnailCamera{
		FieldOfViewDegrees: limits.FieldOfViewDegrees.clamp(entry.Camera.FieldOfViewDegrees),
		YRotationDegrees:   limits.YRotationDegrees.clamp(entry.Camera.YRotationDegrees),
		DistanceScale:      defaultThumbnailCamera.DistanceScale,
	}
	if customization.IsCameraCustomized {
		customization.Camera.DistanceScale = limits.DistanceScale.clamp(entry.Camera.DistanceScale)
	}
	return customization
}

func (limit AvatarScaleLimit) clamp(scale float64) float64 {
	return min(max(scale, limit.Min), limit.Max)
}

func (limit AvatarScaleLimit) contains(scale float64) bool {
	return scale >= limit.Min && scale <= limit.Max
}

// SaveAvatarThumbnail replaces one customization of the account. Roblox
// renders the changed image within about 10 seconds without a redraw. The API
// has no request that deletes a customization, so an emote ID of 0 with the
// camera off returns the image to its standard look, as RoAvatar resets it.
// Roblox accepts only emotes and animations that the account owns. Verified
// against the API on 2026-11-19.
func (service *Users) SaveAvatarThumbnail(ctx context.Context, accountID, version int64, change AvatarThumbnailChange) error {
	const endpoint = "avatar-thumbnail-save"
	robloxThumbnailType, isKnownType := robloxThumbnailTypeByType[change.Type]
	if !isKnownType {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Select the headshot or the full-body image."}
	}
	if change.EmoteAssetID < 0 {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Select a valid emote."}
	}
	limits := cameraLimitsOf(change.Type)
	if !limits.FieldOfViewDegrees.contains(change.Camera.FieldOfViewDegrees) || !limits.YRotationDegrees.contains(change.Camera.YRotationDegrees) ||
		(change.IsCameraCustomized && !limits.DistanceScale.contains(change.Camera.DistanceScale)) {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "A camera setting is outside the range that Roblox allows."}
	}
	distanceScale := -1.0
	if change.IsCameraCustomized {
		distanceScale = change.Camera.DistanceScale
	}
	type cameraUpdate struct {
		FieldOfViewDegrees float64 `json:"fieldOfViewDeg"`
		YRotationDegrees   float64 `json:"yRotDeg"`
		DistanceScale      float64 `json:"distanceScale"`
	}
	request := struct {
		ThumbnailType int          `json:"thumbnailType"`
		EmoteAssetID  int64        `json:"emoteAssetId"`
		Camera        cameraUpdate `json:"camera"`
	}{
		ThumbnailType: robloxThumbnailType, EmoteAssetID: change.EmoteAssetID,
		Camera: cameraUpdate{FieldOfViewDegrees: change.Camera.FieldOfViewDegrees, YRotationDegrees: change.Camera.YRotationDegrees, DistanceScale: distanceScale},
	}
	_, err := service.writeAccount(ctx, accountID, version, endpoint, http.MethodPost, "https://avatar.roblox.com/v1/avatar/thumbnail-customization", request)
	if remote, ok := errors.AsType[*roblox.Error](err); ok && remote.Status == http.StatusBadRequest {
		message := "Roblox rejected the pose. Select an emote or animation that the account owns."
		if slices.Contains(cameraRangeErrorCodes, remote.RobloxCode) {
			message = "A camera setting is outside the range that Roblox allows."
		}
		return &roblox.Error{Kind: remote.Kind, Endpoint: endpoint, Status: remote.Status, RobloxCode: remote.RobloxCode, Message: message, Cause: err}
	}
	return err
}
