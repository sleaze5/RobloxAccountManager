package services

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	maxAssetFileBytes = 16 << 20
	meshAssetTypeID   = 4
	assetFileEndpoint = "asset-file"
)

// AssetFileFormat is the format of the file of an asset.
type AssetFileFormat string

const (
	AssetFilePNG  AssetFileFormat = "png"
	AssetFileJPEG AssetFileFormat = "jpg"
	AssetFileRBXM AssetFileFormat = "rbxm"
	// AssetFileRBXMX is a Roblox model file in the XML format.
	AssetFileRBXMX AssetFileFormat = "rbxmx"
	AssetFileOGG   AssetFileFormat = "ogg"
	AssetFileMP3   AssetFileFormat = "mp3"
	// AssetFileRobloxMesh is a mesh in the format of Roblox, which other
	// programs cannot open. Newer versions compress it with Draco.
	AssetFileRobloxMesh AssetFileFormat = "mesh"
)

// AssetFile is the file of an asset as Roblox stores it.
type AssetFile struct {
	Contents []byte
	Format   AssetFileFormat
}

var errUnknownAssetFileFormat = errors.New("unknown asset file format")

// canDownload reports whether the items of the category are assets whose
// files the account can save. Places can be too large, and badges, game
// passes, and bundles have no file.
func canDownload(category InventoryCategory) bool {
	itemType, ok := inventoryItemTypeOf(category)
	return ok && itemType == inventoryItemAsset && category != InventoryPlaces
}

// AssetFile downloads the file of the asset. The asset delivery API answers
// with the address of the file on a content delivery host, which serves it
// without credentials.
func (service *Users) AssetFile(ctx context.Context, accountID, version int64, category InventoryCategory, assetID int64) (AssetFile, error) {
	if !canDownload(category) {
		return AssetFile{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: assetFileEndpoint, Message: "This item has no file to download."}
	}
	contents, assetTypeID, err := service.downloadAssetFile(ctx, accountID, version, assetFileEndpoint, assetID, maxAssetFileBytes)
	if remote, ok := errors.AsType[*roblox.Error](err); ok && (remote.Status == http.StatusForbidden || remote.Status == http.StatusNotFound) {
		return AssetFile{}, &roblox.Error{Kind: remote.Kind, Endpoint: assetFileEndpoint, Status: remote.Status, Message: "Roblox doesn't let this account download this item.", Cause: err}
	}
	if err != nil {
		return AssetFile{}, err
	}
	if assetTypeID == meshAssetTypeID {
		return AssetFile{Contents: contents, Format: AssetFileRobloxMesh}, nil
	}
	format, err := assetFileFormatOf(contents)
	if err != nil {
		return AssetFile{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: assetFileEndpoint, Message: "This kind of item can't be downloaded yet.", Cause: err}
	}
	return AssetFile{Contents: contents, Format: format}, nil
}

// downloadAssetFile downloads the file of the asset and reports its asset type
// ID.
func (service *Users) downloadAssetFile(ctx context.Context, accountID, version int64, endpoint string, assetID, maxResponseSizeBytes int64) ([]byte, int64, error) {
	var payload struct {
		AssetTypeID int64 `json:"assetTypeId"`
		Locations   []struct {
			Location string `json:"location"`
		} `json:"locations"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, endpoint, "https://assetdelivery.roblox.com/v2/asset/?id="+strconv.FormatInt(assetID, 10), &payload); err != nil {
		return nil, 0, err
	}
	if len(payload.Locations) == 0 {
		return nil, 0, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Status: http.StatusNotFound, Message: "Roblox has no file for this item."}
	}
	contents, err := service.downloadCDNFile(ctx, payload.Locations[0].Location, maxResponseSizeBytes)
	if err != nil {
		return nil, 0, err
	}
	contents, err = unpackGzip(contents)
	if err != nil {
		return nil, 0, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Roblox returned a file that couldn't be read.", Cause: err}
	}
	return contents, payload.AssetTypeID, nil
}

// unpackGzip unpacks a gzipped file. The content delivery host sends files
// gzipped, and the HTTP client unpacks them unless the file itself is gzipped.
func unpackGzip(contents []byte) ([]byte, error) {
	if !bytes.HasPrefix(contents, gzipMagic) {
		return contents, nil
	}
	reader, err := gzip.NewReader(bytes.NewReader(contents))
	if err != nil {
		return nil, errInvalidModelFile
	}
	unpacked, err := io.ReadAll(io.LimitReader(reader, maxModelFileDecompressedBytes))
	if err != nil {
		return nil, errInvalidModelFile
	}
	return unpacked, nil
}

// assetFileFormatOf names the format of a file by its first bytes.
func assetFileFormatOf(contents []byte) (AssetFileFormat, error) {
	switch {
	case bytes.HasPrefix(contents, []byte("\x89PNG\r\n\x1a\n")):
		return AssetFilePNG, nil
	case bytes.HasPrefix(contents, []byte{0xff, 0xd8, 0xff}):
		return AssetFileJPEG, nil
	case bytes.HasPrefix(contents, binaryModelFileSignature):
		return AssetFileRBXM, nil
	case bytes.HasPrefix(contents, xmlModelFileSignature):
		return AssetFileRBXMX, nil
	case bytes.HasPrefix(contents, []byte("OggS")):
		return AssetFileOGG, nil
	case bytes.HasPrefix(contents, []byte("ID3")), len(contents) > 1 && contents[0] == 0xff && contents[1]&0xe0 == 0xe0:
		return AssetFileMP3, nil
	default:
		return "", errUnknownAssetFileFormat
	}
}
