package services

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

// The asset type IDs of the media entries of a game. A game preview video
// lists its poster as its image. Verified on 2026-11-24.
const (
	gameMediaImageAssetTypeID        = 1
	gameMediaYouTubeVideoAssetTypeID = 33
	gameMediaVideoAssetTypeID        = 86
)

const (
	// maxGameMedia bounds the media of a game. Roblox allows fewer.
	maxGameMedia             = 50
	maxGameMediaTextLength   = 200
	youTubeVideoPreviewImage = "hqdefault.jpg"
)

// youTubeVideoIDPattern matches the 11 characters of a YouTube video ID, so
// that an ID never changes the path or host of a link.
var youTubeVideoIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

// ListedGameMedia is a media entry of a game before Roblox renders its
// preview image. ImageID is 0 when the entry has no Roblox image.
type ListedGameMedia struct {
	Media   games.Media
	ImageID int64
}

// Media reads the approved images and videos of the universe in the order
// that its creator set. Roblox lists them without a signed-in request.
// Verified on 2026-11-24.
func (service *Games) Media(ctx context.Context, universeID int64) ([]ListedGameMedia, error) {
	var payload struct {
		Data *[]struct {
			AssetTypeID int64  `json:"assetTypeId"`
			ImageID     int64  `json:"imageId"`
			VideoHash   string `json:"videoHash"`
			VideoTitle  string `json:"videoTitle"`
			VideoID     string `json:"videoId"`
			IsApproved  bool   `json:"approved"`
			AltText     string `json:"altText"`
		} `json:"data"`
	}
	path := "/v2/games/" + strconv.FormatInt(universeID, 10) + "/media"
	if err := service.get(ctx, "game-media", "games.roblox.com", path, nil, &payload); err != nil {
		return nil, err
	}
	if payload.Data == nil || len(*payload.Data) > maxGameMedia {
		return nil, gameDataError("Roblox returned invalid media data.")
	}
	media := []ListedGameMedia{}
	for _, listedMedia := range *payload.Data {
		if !listedMedia.IsApproved {
			continue
		}
		entry := games.Media{
			AltText:    boundedText(listedMedia.AltText, maxGameMediaTextLength),
			VideoTitle: boundedText(listedMedia.VideoTitle, maxGameMediaTextLength),
		}
		switch listedMedia.AssetTypeID {
		case gameMediaImageAssetTypeID:
			entry.Kind = games.MediaKindImage
		case gameMediaVideoAssetTypeID:
			videoAssetID, err := strconv.ParseInt(listedMedia.VideoID, 10, 64)
			if err != nil || videoAssetID <= 0 {
				continue
			}
			entry.Kind, entry.VideoAssetID = games.MediaKindVideo, videoAssetID
		case gameMediaYouTubeVideoAssetTypeID:
			if !youTubeVideoIDPattern.MatchString(listedMedia.VideoHash) {
				continue
			}
			entry.Kind = games.MediaKindYouTubeVideo
			entry.VideoPageURL = (&url.URL{Scheme: "https", Host: "www.youtube.com", Path: "/watch", RawQuery: url.Values{"v": {listedMedia.VideoHash}}.Encode()}).String()
			entry.ImageURL = (&url.URL{Scheme: "https", Host: "i.ytimg.com", Path: "/vi/" + listedMedia.VideoHash + "/" + youTubeVideoPreviewImage}).String()
		default:
			continue
		}
		imageID := int64(0)
		if entry.Kind != games.MediaKindYouTubeVideo && listedMedia.ImageID > 0 {
			imageID = listedMedia.ImageID
		}
		media = append(media, ListedGameMedia{Media: entry, ImageID: imageID})
	}
	return media, nil
}

// listedAssetLocation is an address from which the asset delivery API serves
// the file of an asset.
type listedAssetLocation struct {
	Location string `json:"location"`
}

// VideoURL reads the address from which the app plays a game preview video.
// The address is signed and expires, so the frontend reads it each time a
// video opens. Roblox gives a public video without a signed-in request.
// Verified on 2026-11-24.
func (service *Games) VideoURL(ctx context.Context, videoAssetID int64) (string, error) {
	const operation = "game-video"
	var payload struct {
		AssetTypeID int64                 `json:"assetTypeId"`
		Locations   []listedAssetLocation `json:"locations"`
	}
	parameters := url.Values{"id": {strconv.FormatInt(videoAssetID, 10)}}
	if err := service.get(ctx, operation, "assetdelivery.roblox.com", "/v2/asset/", parameters, &payload); err != nil {
		return "", err
	}
	if payload.AssetTypeID != gameMediaVideoAssetTypeID || len(payload.Locations) == 0 || !isRobloxCDNURL(payload.Locations[0].Location) {
		return "", &roblox.Error{
			Kind: roblox.KindProtocol, Endpoint: operation, Status: http.StatusNotFound, Message: "Roblox has no playable file for this video.",
			Cause: fmt.Errorf("asset type %d with %d locations, first on host %q", payload.AssetTypeID, len(payload.Locations), locationHost(payload.Locations)),
		}
	}
	return payload.Locations[0].Location, nil
}

// locationHost names the host of the first location, for the log of a
// rejected video. The signed query of the address stays out of the log.
func locationHost(locations []listedAssetLocation) string {
	if len(locations) == 0 {
		return ""
	}
	parsed, err := url.Parse(locations[0].Location)
	if err != nil {
		return ""
	}
	return parsed.Host
}
