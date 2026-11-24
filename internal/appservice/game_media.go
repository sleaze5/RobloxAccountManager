package appservice

import (
	"context"

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

// GetGameVideoURL reads the signed address from which the frontend plays a
// game preview video.
func (service *Service) GetGameVideoURL(ctx context.Context, videoAssetID int64) (string, error) {
	const operation = "game-video"
	if err := validGameID(videoAssetID); err != nil {
		return "", err
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return "", err
	}
	defer done()
	videoURL, err := service.games.VideoURL(ctx, videoAssetID)
	return videoURL, service.gamesError(ctx, operation, err)
}
