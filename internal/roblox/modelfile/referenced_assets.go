package modelfile

import (
	"regexp"
	"slices"
	"strconv"
)

const maxReferencedAssetCount = 50

// assetReferencePattern finds the asset IDs in content values, such as
// rbxassetid://123 and http://www.roblox.com/asset/?id=123.
var assetReferencePattern = regexp.MustCompile(`(?i)(?:rbxassetid://|[?&]id=)(\d{1,19})`)

// ReferencedAssetIDs lists the assets that the properties of a Roblox model
// file name, such as the mesh and the texture of an accessory, in the order in
// which they first appear. A file that is not a model file, such as an image,
// references nothing.
func ReferencedAssetIDs(file []byte) ([]int64, error) {
	switch {
	case IsBinary(file):
		return assetIDsReferencedByBinaryFile(file)
	case IsXML(file):
		return assetIDsReferencedByText(file, nil), nil
	default:
		return nil, nil
	}
}

// assetIDsReferencedByBinaryFile searches the property chunks of the binary
// format, which hold every content value.
func assetIDsReferencedByBinaryFile(file []byte) ([]int64, error) {
	chunks, err := splitBinaryChunks(file)
	if err != nil {
		return nil, err
	}
	decompressor, err := newChunkDecompressor()
	if err != nil {
		return nil, err
	}
	defer decompressor.close()
	var assetIDs []int64
	for _, chunk := range chunks {
		if chunk.name != "PROP" {
			continue
		}
		chunkData, err := decompressor.decompress(chunk)
		if err != nil {
			return nil, err
		}
		assetIDs = assetIDsReferencedByText(chunkData, assetIDs)
		if len(assetIDs) >= maxReferencedAssetCount {
			break
		}
	}
	return assetIDs, nil
}

// assetIDsReferencedByText adds the asset IDs that the text names to
// assetIDs, without repeating one.
func assetIDsReferencedByText(text []byte, assetIDs []int64) []int64 {
	for _, match := range assetReferencePattern.FindAllSubmatch(text, -1) {
		assetID, err := strconv.ParseInt(string(match[1]), 10, 64)
		if err != nil || assetID <= 0 {
			continue
		}
		if !slices.Contains(assetIDs, assetID) {
			assetIDs = append(assetIDs, assetID)
		}
		if len(assetIDs) >= maxReferencedAssetCount {
			break
		}
	}
	return assetIDs
}

// referencedAssetID returns the asset that one content value names, such as
// rbxassetid://123, or 0 when it names none.
func referencedAssetID(content string) int64 {
	match := assetReferencePattern.FindStringSubmatch(content)
	if match == nil {
		return 0
	}
	assetID, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil || assetID <= 0 {
		return 0
	}
	return assetID
}
