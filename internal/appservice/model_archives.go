package appservice

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
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
func (service *Service) ReadInventoryItemModelArchive(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) ([]byte, error) {
	model, err := service.GetInventoryItemModel(ctx, accountID, category, itemID)
	if err != nil {
		return nil, err
	}
	archive, err := modelArchiveOf(model)
	if err != nil {
		service.logs.Module("application.inventory").WarnContext(ctx, "could not pack the 3D model", "operation", "model-download", "account_id", accountID, "category", category, "error", err)
		return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "model-download", Message: "Could not prepare the 3D model. Try again.", Cause: err}
	}
	return archive, nil
}

// modelArchiveOf names the files of the model for the archive. Roblox names
// the materials file and each texture by its file hash without an extension,
// so the mesh and the materials are changed to name the files in the archive.
// The names hold no spaces, because OBJ and MTL readers split lines at spaces.
func modelArchiveOf(model robloxservices.RenderedModel) ([]byte, error) {
	type archiveFile struct {
		name     string
		contents []byte
	}
	textureFileHashes := slices.Sorted(maps.Keys(model.TextureDataURLByFileHash))
	textureNameByFileHash := make(map[string]string, len(textureFileHashes))
	textureFiles := make([]archiveFile, 0, len(textureFileHashes))
	for index, fileHash := range textureFileHashes {
		contents, extension, err := decodeTextureDataURL(model.TextureDataURLByFileHash[fileHash])
		if err != nil {
			return nil, err
		}
		name := fmt.Sprintf("texture%d.%s", index+1, extension)
		textureNameByFileHash[fileHash] = name
		textureFiles = append(textureFiles, archiveFile{name, contents})
	}
	files := append([]archiveFile{
		{modelArchiveMeshName, []byte(meshNamingMaterials(model.Mesh))},
		{modelArchiveMaterialsName, []byte(materialsNamingTextures(model.Materials, textureNameByFileHash))},
	}, textureFiles...)

	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for _, file := range files {
		fileWriter, err := writer.Create(file.name)
		if err != nil {
			return nil, fmt.Errorf("add %s to the archive: %w", file.name, err)
		}
		if _, err := fileWriter.Write(file.contents); err != nil {
			return nil, fmt.Errorf("write %s to the archive: %w", file.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close the archive: %w", err)
	}
	return archive.Bytes(), nil
}

// meshNamingMaterials points the mtllib statement of the mesh at the
// materials file of the archive, and adds one when the mesh has none.
func meshNamingMaterials(mesh string) string {
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

// materialsNamingTextures replaces each texture file hash at the end of a
// statement, such as map_Kd, with the name of the texture in the archive.
func materialsNamingTextures(materials string, textureNameByFileHash map[string]string) string {
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
		return nil, "", fmt.Errorf("decode texture: unexpected data URL type %q", header)
	}
	contents, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, "", fmt.Errorf("decode texture: %w", err)
	}
	return contents, extension, nil
}
