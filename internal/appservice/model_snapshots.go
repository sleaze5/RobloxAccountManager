package appservice

import (
	"bytes"
	"image/png"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	modelSnapshotExtension = "png"
	// A frame of the largest 3D view on a 4K screen is well under this size.
	maxModelSnapshotSizeBytes = 64 << 20
)

func (service *Service) ModelSnapshotFileName(subjectName string) string {
	return fileNameOf(subjectName, "3D view", modelSnapshotExtension)
}

// ValidateModelSnapshotPNG checks the size and PNG header without decoding pixels.
func (service *Service) ValidateModelSnapshotPNG(snapshotPNG []byte) error {
	const operation = "model-snapshot"
	if len(snapshotPNG) > maxModelSnapshotSizeBytes {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: operation, Message: "The snapshot is too large to save."}
	}
	if _, err := png.DecodeConfig(bytes.NewReader(snapshotPNG)); err != nil {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: operation, Message: "The snapshot is not a valid image. Try again.", Cause: err}
	}
	return nil
}

func (service *Service) SaveModelSnapshot(path string, snapshotPNG []byte) (string, error) {
	savedPath, err := service.saveChosenFile(path, modelSnapshotExtension, snapshotPNG, "model-snapshot", "application.model-viewer")
	if err == nil {
		service.logs.Module("application.model-viewer").Info("3D view snapshot saved", "operation", "model-snapshot", "size_bytes", len(snapshotPNG))
	}
	return savedPath, err
}
