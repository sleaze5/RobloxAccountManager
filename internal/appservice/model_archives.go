package appservice

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

const (
	modelArchiveMeshName      = "model.obj"
	modelArchiveMaterialsName = "model.mtl"
)

var textureExtensionByContentType = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpg",
}

// ReadInventoryItemModelArchive packs the 3D model that Roblox renders of the
// item into a ZIP archive that 3D programs open: the OBJ mesh, its MTL
// materials, and the textures that the materials apply to the mesh.
func (service *Service) ReadInventoryItemModelArchive(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) (DownloadableFile, error) {
	model, err := service.GetInventoryItemModel(ctx, accountID, category, itemID)
	if err != nil {
		return DownloadableFile{}, err
	}
	archive, err := modelArchiveOf(model)
	if err != nil {
		service.logs.Module("application.inventory").WarnContext(ctx, "could not pack the 3D model", "operation", "model-download", "account_id", accountID, "category", category, "error", err)
		return DownloadableFile{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "model-download", Message: "Could not prepare the 3D model. Try again.", Cause: err}
	}
	return DownloadableFile{Contents: archive, Extension: "zip", FormatName: "ZIP archive of an OBJ model"}, nil
}

// modelArchiveOf names the files of the model for the archive. Roblox names
// the materials file and each texture by its file hash without an extension,
// so the mesh and the materials are changed to name the files in the archive.
// The names hold no spaces, because OBJ and MTL readers split lines at spaces.
func modelArchiveOf(model robloxservices.RenderedModel) ([]byte, error) {
	textureFileHashes := slices.Sorted(maps.Keys(model.TextureDataURLByFileHash))
	textureNameByFileHash := make(map[string]string, len(textureFileHashes))
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for index, fileHash := range textureFileHashes {
		contents, extension, err := decodeTextureDataURL(model.TextureDataURLByFileHash[fileHash])
		if err != nil {
			return nil, err
		}
		name := fmt.Sprintf("texture%d.%s", index+1, extension)
		textureNameByFileHash[fileHash] = name
		if err := writeModelArchiveEntry(writer, name, contents); err != nil {
			return nil, err
		}
	}
	if err := writeModelArchiveEntry(writer, modelArchiveMeshName, []byte(rewriteMeshMaterialLibrary(model.Mesh))); err != nil {
		return nil, err
	}
	if err := writeModelArchiveEntry(writer, modelArchiveMaterialsName, []byte(rewriteMaterialTextureNames(model.Materials, textureNameByFileHash))); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close the archive: %w", err)
	}
	return archive.Bytes(), nil
}

func writeModelArchiveEntry(writer *zip.Writer, name string, contents []byte) error {
	entry, err := writer.Create(name)
	if err != nil {
		return fmt.Errorf("add %s to the archive: %w", name, err)
	}
	if _, err := entry.Write(contents); err != nil {
		return fmt.Errorf("write %s to the archive: %w", name, err)
	}
	return nil
}

func rewriteMeshMaterialLibrary(mesh string) string {
	lines := strings.Split(mesh, "\n")
	hasMaterialLibrary := false
	for index, line := range lines {
		if fields := strings.Fields(line); len(fields) > 0 && fields[0] == "mtllib" {
			lines[index] = "mtllib " + modelArchiveMaterialsName
			hasMaterialLibrary = true
		}
	}
	if !hasMaterialLibrary {
		lines = append([]string{"mtllib " + modelArchiveMaterialsName}, lines...)
	}
	return strings.Join(lines, "\n")
}

// Roblox puts texture hashes at the end of map statements such as map_Kd.
func rewriteMaterialTextureNames(materials string, textureNameByFileHash map[string]string) string {
	lines := strings.Split(materials, "\n")
	for index, line := range lines {
		trimmedLine := strings.TrimRight(line, " \t\r")
		fields := strings.Fields(trimmedLine)
		if len(fields) < 2 {
			continue
		}
		fileHash := fields[len(fields)-1]
		if textureName, isTexture := textureNameByFileHash[fileHash]; isTexture {
			lines[index] = strings.TrimSuffix(trimmedLine, fileHash) + textureName
		}
	}
	return strings.Join(lines, "\n")
}

// decodeTextureDataURL reads a texture from the data URL that the rendered
// model holds it in.
func decodeTextureDataURL(dataURL string) ([]byte, string, error) {
	header, encoded, isDataURL := strings.Cut(strings.TrimPrefix(dataURL, "data:"), ";base64,")
	extension, isKnownType := textureExtensionByContentType[header]
	if !isDataURL || !isKnownType {
		return nil, "", errors.New("decode texture: invalid PNG or JPEG data URL")
	}
	contents, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, "", fmt.Errorf("decode texture: %w", err)
	}
	return contents, extension, nil
}
