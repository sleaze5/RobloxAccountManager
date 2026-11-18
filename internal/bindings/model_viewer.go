package bindings

import (
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

// SaveModelSnapshot asks where to save a PNG frame of a 3D view, then saves it.
// subjectName names the file, such as the name of an item. An empty path
// without an error means the user cancelled the dialog.
func (service *Service) SaveModelSnapshot(subjectName string, snapshotPNG []byte) (string, error) {
	if err := service.core.ValidateModelSnapshotPNG(snapshotPNG); err != nil {
		return "", err
	}
	path, err := chooseSaveFile(service.core.DefaultDownloadDirectory(), service.core.ModelSnapshotFileName(subjectName), "PNG image", "*.png")
	if err != nil {
		return "", &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "model-snapshot", Message: "Could not open the save dialog.", Cause: err}
	}
	if path == "" {
		return "", nil
	}
	return service.core.SaveModelSnapshot(path, snapshotPNG)
}
