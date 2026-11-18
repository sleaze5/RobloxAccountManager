package appservice

import (
	"bytes"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	modelSnapshotExtension = "png"
	// A frame of the largest 3D view on a 4K screen is well under this size.
	maxModelSnapshotSizeBytes = 64 << 20
)

var pngSignature = []byte("\x89PNG\r\n\x1a\n")

// ModelSnapshotFileName names the snapshot of a 3D view after the subject that
// it shows.
func (service *Service) ModelSnapshotFileName(subjectName string) string {
	return fileNameOf(subjectName, "3D view", modelSnapshotExtension)
}

// ValidateModelSnapshotPNG checks that a snapshot of a 3D view is a PNG image of
// a size that a frame can have.
func (service *Service) ValidateModelSnapshotPNG(snapshotPNG []byte) error {
	if len(snapshotPNG) > maxModelSnapshotSizeBytes || !bytes.HasPrefix(snapshotPNG, pngSignature) {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "model-snapshot", Message: "The snapshot is not a valid image. Try again."}
	}
	return nil
}

// SaveModelSnapshot writes the snapshot of a 3D view where the user chose.
func (service *Service) SaveModelSnapshot(path string, snapshotPNG []byte) (string, error) {
	savedPath, err := service.saveChosenFile(path, modelSnapshotExtension, snapshotPNG, "model-snapshot", "application.model-viewer")
	if err == nil {
		service.logs.Module("application.model-viewer").Info("3D view snapshot saved", "operation", "model-snapshot", "size_bytes", len(snapshotPNG))
	}
	return savedPath, err
}
