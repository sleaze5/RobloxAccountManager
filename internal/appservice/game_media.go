package appservice

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// ListGameMedia reads the images and videos that the game shows on its page.
// An entry whose preview image Roblox cannot render keeps an empty ImageURL.
func (service *Service) ListGameMedia(ctx context.Context, universeID int64) ([]games.Media, error) {
	const operation = "game-media"
	if err := validGameID(universeID); err != nil {
		return nil, err
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return nil, err
	}
	defer done()
	listedMedia, err := service.games.Media(ctx, universeID)
	if err != nil {
		return nil, service.gamesError(ctx, operation, err)
	}
	targets := make([]robloxservices.ThumbnailTarget, 0, len(listedMedia))
	for _, entry := range listedMedia {
		if entry.ImageID > 0 {
			targets = append(targets, robloxservices.ThumbnailTarget{Type: "GameThumbnail", TargetID: entry.ImageID})
		}
	}
	images := map[robloxservices.ThumbnailTarget]string{}
	if len(targets) > 0 {
		if images, err = service.thumbnails.Images(ctx, targets, robloxservices.WideThumbnailSize); err != nil {
			// The media still lists without its images, which the frontend
			// shows as unavailable.
			service.logs.Module("application.games").WarnContext(ctx, "game media images unavailable", "operation", operation, "universe_id", universeID, "images", len(targets), "error", err)
			images = map[robloxservices.ThumbnailTarget]string{}
		}
	}
	media := make([]games.Media, len(listedMedia))
	for index, entry := range listedMedia {
		media[index] = entry.Media
		if entry.ImageID > 0 {
			media[index].ImageURL = images[robloxservices.ThumbnailTarget{Type: "GameThumbnail", TargetID: entry.ImageID}]
		}
	}
	return media, nil
}

// GameVideoPath is the address from which the frontend plays a game preview
// video, by its asset ID.
const GameVideoPath = "/game-videos/{videoAssetID}"

// shownGameVideo keeps the file of the last video that the player opened, so
// that each byte range that it seeks to comes from memory. The player shows
// one video at a time.
type shownGameVideo struct {
	mutex        sync.Mutex
	videoAssetID int64
	contents     []byte
}

// ServeGameVideo serves a game preview video to the player from the app's
// own address, with the byte ranges that the player seeks to.
func (service *Service) ServeGameVideo(writer http.ResponseWriter, request *http.Request) {
	const operation = "game-video"
	videoAssetID, err := strconv.ParseInt(request.PathValue("videoAssetID"), 10, 64)
	if err != nil || validGameID(videoAssetID) != nil {
		http.Error(writer, "The video ID is invalid.", http.StatusBadRequest)
		return
	}
	ctx, done, err := service.beginGames(request.Context())
	if err != nil {
		http.Error(writer, "Unlock the vault first.", http.StatusForbidden)
		return
	}
	defer done()
	contents, err := service.readGameVideo(ctx, videoAssetID)
	if err != nil {
		service.logs.Module("application.games").WarnContext(ctx, "game video unavailable", "operation", operation, "video_asset_id", videoAssetID, "error", err)
		http.Error(writer, "Roblox has no playable file for this video.", http.StatusBadGateway)
		return
	}
	// Roblox serves game preview videos as WebM, and names them only as
	// binary data.
	writer.Header().Set("Content-Type", "video/webm")
	http.ServeContent(writer, request, "", time.Time{}, bytes.NewReader(contents))
}

// readGameVideo returns the file of the video, which it downloads unless the
// player opened the same video last.
func (service *Service) readGameVideo(ctx context.Context, videoAssetID int64) ([]byte, error) {
	shown := &service.shownGameVideo
	shown.mutex.Lock()
	defer shown.mutex.Unlock()
	if shown.videoAssetID == videoAssetID {
		return shown.contents, nil
	}
	contents, err := service.games.VideoFile(ctx, videoAssetID)
	if err != nil {
		return nil, fmt.Errorf("download game video: %w", err)
	}
	shown.videoAssetID, shown.contents = videoAssetID, contents
	return contents, nil
}

// ReadGameImageFile downloads the original of a game media image as the
// account sees it.
func (service *Service) ReadGameImageFile(ctx context.Context, accountID, imageAssetID int64) (DownloadableFile, error) {
	const operation = "game-image"
	if err := validGameID(imageAssetID); err != nil {
		return DownloadableFile{}, err
	}
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, operation)
	if err != nil {
		return DownloadableFile{}, err
	}
	defer done()
	contents, format, err := service.users.ImageFile(ctx, accountID, version, imageAssetID)
	if err != nil {
		return DownloadableFile{}, service.gamesError(ctx, operation, err)
	}
	return DownloadableFile{Contents: contents, Extension: string(format), FormatName: formatNameByAssetFileFormat[format]}, nil
}

// ReadGameVideoFile downloads the file of a game preview video, which the
// player may already hold.
func (service *Service) ReadGameVideoFile(ctx context.Context, videoAssetID int64) (DownloadableFile, error) {
	const operation = "game-video"
	if err := validGameID(videoAssetID); err != nil {
		return DownloadableFile{}, err
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return DownloadableFile{}, err
	}
	defer done()
	contents, err := service.readGameVideo(ctx, videoAssetID)
	if err != nil {
		return DownloadableFile{}, service.gamesError(ctx, operation, err)
	}
	return DownloadableFile{Contents: contents, Extension: "webm", FormatName: "WebM video"}, nil
}
