package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
)

func (service *Service) ListGameMedia(ctx context.Context, universeID int64) ([]games.Media, error) {
	return service.core.ListGameMedia(ctx, universeID)
}

// DownloadGameImage downloads the original of a game media image, then asks
// where to save it. An empty path without an error means the user cancelled
// the dialog. fileName names the file without its extension.
func (service *Service) DownloadGameImage(ctx context.Context, accountID, imageAssetID int64, fileName string) (string, error) {
	const operation = "game-image-download"
	if err := validateAccountAndTargetIDs(accountID, imageAssetID, operation); err != nil {
		return "", err
	}
	file, err := service.core.ReadGameImageFile(ctx, accountID, imageAssetID)
	if err != nil {
		return "", err
	}
	return service.saveDownload(ctx, operation, imageAssetID, fileName, file)
}

// DownloadGameVideo downloads the file of a game preview video, then asks
// where to save it. An empty path without an error means the user cancelled
// the dialog. fileName names the file without its extension.
func (service *Service) DownloadGameVideo(ctx context.Context, videoAssetID int64, fileName string) (string, error) {
	const operation = "game-video-download"
	file, err := service.core.ReadGameVideoFile(ctx, videoAssetID)
	if err != nil {
		return "", err
	}
	return service.saveDownload(ctx, operation, videoAssetID, fileName, file)
}
