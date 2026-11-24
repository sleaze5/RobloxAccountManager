package services

import (
	"context"
	"strconv"
)

// avatarBackgroundSelectionType is the selection type with which the avatar
// API lists only the background that a user equipped. Verified on 2026-11-24.
const avatarBackgroundSelectionType = "5"

const maxAvatarBackgroundNameLength = 200

// AvatarBackground is the Marketplace background that a user equipped, which
// Roblox shows behind the avatar on the profile. ImageURL is 16:9, and is
// empty when Roblox has not rendered it.
type AvatarBackground struct {
	AssetID  int64  `json:"assetId"`
	Name     string `json:"name"`
	ImageURL string `json:"imageUrl"`
}

// AvatarPortrait is how a profile shows a user: the full-body avatar image in
// front of the background that the user equipped. FullBodyImageURL is empty
// when Roblox has not rendered it. Background is nil when the user equipped
// none, or when IsBackgroundUnavailable reports that it could not be read.
type AvatarPortrait struct {
	FullBodyImageURL        string            `json:"fullBodyImageUrl"`
	Background              *AvatarBackground `json:"background"`
	IsBackgroundUnavailable bool              `json:"isBackgroundUnavailable"`
}

// ThumbnailTarget names the image of the background.
func (background AvatarBackground) ThumbnailTarget() ThumbnailTarget {
	return ThumbnailTarget{Type: "AvatarBackground", TargetID: background.AssetID}
}

// EquippedAvatarBackground reads the background that the user equipped, or nil
// when the user has none.
func (service *Users) EquippedAvatarBackground(ctx context.Context, accountID, version, userID int64) (*AvatarBackground, error) {
	const endpoint = "avatar-background"
	var payload struct {
		AvatarConfigurations *struct {
			Background *struct {
				BackgroundAsset *struct {
					ID   int64  `json:"id"`
					Name string `json:"name"`
				} `json:"backgroundAsset"`
			} `json:"background"`
		} `json:"avatarConfigurations"`
	}
	address := "https://avatar.roblox.com/v4/avatar/users/" + strconv.FormatInt(userID, 10) + "?selectionTypes=" + avatarBackgroundSelectionType
	if err := service.readAvatarDetails(ctx, accountID, version, endpoint, address, &payload); err != nil {
		return nil, err
	}
	if payload.AvatarConfigurations == nil {
		return nil, invalidAvatar(endpoint)
	}
	background := payload.AvatarConfigurations.Background
	if background == nil || background.BackgroundAsset == nil || background.BackgroundAsset.ID <= 0 {
		return nil, nil
	}
	return &AvatarBackground{
		AssetID: background.BackgroundAsset.ID,
		Name:    boundedText(background.BackgroundAsset.Name, maxAvatarBackgroundNameLength),
	}, nil
}
