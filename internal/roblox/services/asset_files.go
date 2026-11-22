package services

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox/modelfile"
)

const (
	maxAssetFileBytes         = 16 << 20
	maxUnpackedAssetFileBytes = 64 << 20
	meshAssetTypeID           = 4
	assetFileEndpoint         = "asset-file"
	avatarRenderFileEndpoint  = "avatar-render-file"
	// maxRepresentationCount bounds the content representations that one
	// request lists. The avatar renderer lists at most two for a head.
	maxRepresentationCount = 8
	maxPriorityListLength  = 2048
)

// AssetRepresentation picks the form in which the asset delivery API returns
// an asset, such as a classic head as a mesh part. The zero value asks for
// the form that Roblox stores.
type AssetRepresentation struct {
	// Format is the value of the Roblox-AssetFormat header, such as
	// avatar_meshpart_accessory.
	Format string `json:"format"`
	// PriorityList is the contentRepresentationPriorityList query value: a
	// base64 JSON list of the formats to try, in order, each with a major
	// version.
	PriorityList string `json:"priorityList"`
}

// representationNamePattern accepts the names of asset formats, such as
// avatar_meshpart_head and Roundy_SpecialMesh, and majorVersionPattern their
// major versions, such as 1. Neither accepts anything that could change the
// meaning of a header or a query.
var (
	representationNamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{1,64}$`)
	majorVersionPattern       = regexp.MustCompile(`^\d{1,4}$`)
)

// normalize forwards only the format names and major versions that were checked.
func (representation AssetRepresentation) normalize() (AssetRepresentation, bool) {
	if representation.Format != "" && !representationNamePattern.MatchString(representation.Format) {
		return AssetRepresentation{}, false
	}
	if representation.PriorityList == "" {
		return representation, true
	}
	if len(representation.PriorityList) > maxPriorityListLength {
		return AssetRepresentation{}, false
	}
	priorityListJSON, err := base64.StdEncoding.DecodeString(representation.PriorityList)
	if err != nil {
		return AssetRepresentation{}, false
	}
	var listedRepresentations []struct {
		Format       string `json:"format"`
		MajorVersion string `json:"majorVersion"`
	}
	if err := json.Unmarshal(priorityListJSON, &listedRepresentations); err != nil || len(listedRepresentations) == 0 || len(listedRepresentations) > maxRepresentationCount {
		return AssetRepresentation{}, false
	}
	for _, listedRepresentation := range listedRepresentations {
		if !representationNamePattern.MatchString(listedRepresentation.Format) || !majorVersionPattern.MatchString(listedRepresentation.MajorVersion) {
			return AssetRepresentation{}, false
		}
	}
	normalizedPriorityList, err := json.Marshal(listedRepresentations)
	if err != nil {
		return AssetRepresentation{}, false
	}
	representation.PriorityList = base64.StdEncoding.EncodeToString(normalizedPriorityList)
	return representation, true
}

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

var (
	gzipMagic                 = []byte{0x1f, 0x8b}
	errUnknownAssetFileFormat = errors.New("unknown asset file format")
	errOversizedAssetFile     = errors.New("unpacked asset file exceeds the size limit")
)

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
	contents, assetTypeID, err := service.downloadAssetFile(ctx, accountID, version, assetFileEndpoint, assetID, AssetRepresentation{}, maxAssetFileBytes)
	if remote, isRefused := assetFileRefusal(err); isRefused {
		return AssetFile{}, &roblox.Error{Kind: remote.Kind, Endpoint: assetFileEndpoint, Status: remote.Status, Message: "Roblox does not let this account download this item.", Cause: err}
	}
	if err != nil {
		return AssetFile{}, err
	}
	if assetTypeID == meshAssetTypeID {
		return AssetFile{Contents: contents, Format: AssetFileRobloxMesh}, nil
	}
	format, err := assetFileFormatOf(contents)
	if err != nil {
		return AssetFile{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: assetFileEndpoint, Message: "This kind of item cannot be downloaded yet.", Cause: err}
	}
	return AssetFile{Contents: contents, Format: format}, nil
}

// AvatarRenderFile downloads the file of an asset that the avatar renderer
// reads, such as a mesh, a texture, or an accessory model, in the
// representation that the renderer asks for.
func (service *Users) AvatarRenderFile(ctx context.Context, accountID, version, assetID int64, representation AssetRepresentation) ([]byte, error) {
	normalizedRepresentation, isValid := representation.normalize()
	if !isValid {
		return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: avatarRenderFileEndpoint, Message: "The 3D avatar asked for an asset in an unknown form."}
	}
	contents, _, err := service.downloadAssetFile(ctx, accountID, version, avatarRenderFileEndpoint, assetID, normalizedRepresentation, maxAssetFileBytes)
	return contents, err
}

// assetFileRefusal returns the error with which Roblox refuses to give the
// account the file of an asset, which it answers with 403 or 404.
func assetFileRefusal(err error) (*roblox.Error, bool) {
	remote, ok := errors.AsType[*roblox.Error](err)
	if ok && (remote.Status == http.StatusForbidden || remote.Status == http.StatusNotFound) {
		return remote, true
	}
	return nil, false
}

// downloadAssetFile downloads the file of the asset in the representation and
// reports its asset type ID.
func (service *Users) downloadAssetFile(ctx context.Context, accountID, version int64, endpoint string, assetID int64, representation AssetRepresentation, maxResponseSizeBytes int64) ([]byte, int64, error) {
	var payload struct {
		AssetTypeID int64 `json:"assetTypeId"`
		Locations   []struct {
			Location string `json:"location"`
		} `json:"locations"`
	}
	query := url.Values{"id": {strconv.FormatInt(assetID, 10)}}
	if representation.PriorityList != "" {
		query.Set("contentRepresentationPriorityList", representation.PriorityList)
	}
	headers := http.Header{}
	if representation.Format != "" {
		headers.Set("Roblox-Assetformat", representation.Format)
	}
	if err := service.readAccountDetailsWithHeaders(ctx, accountID, version, endpoint, "https://assetdelivery.roblox.com/v2/asset/?"+query.Encode(), headers, &payload); err != nil {
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
		return nil, 0, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Roblox returned a file that the app cannot read.", Cause: err}
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
		return nil, fmt.Errorf("open gzipped asset file: %w", err)
	}
	// Reading one byte past the limit tells an oversized file apart from one
	// that is exactly the limit, so that a cut-off file is never returned.
	unpacked, err := io.ReadAll(io.LimitReader(reader, maxUnpackedAssetFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("unpack gzipped asset file: %w", err)
	}
	if len(unpacked) > maxUnpackedAssetFileBytes {
		return nil, errOversizedAssetFile
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
	case modelfile.IsBinary(contents):
		return AssetFileRBXM, nil
	case modelfile.IsXML(contents):
		return AssetFileRBXMX, nil
	case bytes.HasPrefix(contents, []byte("OggS")):
		return AssetFileOGG, nil
	case bytes.HasPrefix(contents, []byte("ID3")), len(contents) > 1 && contents[0] == 0xff && contents[1]&0xe0 == 0xe0:
		return AssetFileMP3, nil
	default:
		return "", errUnknownAssetFileFormat
	}
}
