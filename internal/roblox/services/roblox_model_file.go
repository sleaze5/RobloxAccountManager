package services

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/klauspost/compress/zstd"
)

const (
	maxModelFileChunkBytes        = 16 << 20
	maxModelFileDecompressedBytes = 64 << 20
	maxReferencedAssetCount       = 50
)

var (
	binaryModelFileSignature = []byte("<roblox!\x89\xff\x0d\x0a\x1a\x0a")
	xmlModelFileSignature    = []byte("<roblox ")
	zstdFrameMagic           = []byte{0x28, 0xb5, 0x2f, 0xfd}
	// assetReferencePattern finds the asset IDs in content values, such as
	// rbxassetid://123 and http://www.roblox.com/asset/?id=123.
	assetReferencePattern = regexp.MustCompile(`(?i)(?:rbxassetid://|[?&]id=)(\d{1,19})`)
	errInvalidModelFile   = errors.New("invalid Roblox model file")
)

// assetIDsReferencedByModelFile lists the assets that the properties of a
// Roblox model file name, such as the mesh and the texture of an accessory, in
// the order in which they first appear. A file that is not a model file, such
// as an image, references nothing.
func assetIDsReferencedByModelFile(modelFile []byte) ([]int64, error) {
	switch {
	case bytes.HasPrefix(modelFile, binaryModelFileSignature):
		return assetIDsReferencedByBinaryModelFile(modelFile)
	case bytes.HasPrefix(modelFile, xmlModelFileSignature):
		return assetIDsReferencedByText(modelFile, nil), nil
	default:
		return nil, nil
	}
}

// assetIDsReferencedByBinaryModelFile reads the property chunks of the binary
// format. After a 32-byte header, each chunk has a 4-byte name, its compressed
// and uncompressed lengths and 4 reserved bytes, all little endian, then its
// data. A compressed length of 0 means the data is stored as is. Roblox
// compresses chunks with Zstandard in newer files and LZ4 in older ones.
func assetIDsReferencedByBinaryModelFile(modelFile []byte) ([]int64, error) {
	const (
		headerLength      = 32
		chunkHeaderLength = 16
	)
	if len(modelFile) < headerLength {
		return nil, errInvalidModelFile
	}
	decoder, err := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1), zstd.WithDecoderMaxMemory(maxModelFileChunkBytes))
	if err != nil {
		return nil, fmt.Errorf("create zstd decoder: %w", err)
	}
	defer decoder.Close()
	var assetIDs []int64
	decompressedTotal := 0
	for offset := headerLength; offset+chunkHeaderLength <= len(modelFile); {
		chunkName := string(modelFile[offset : offset+4])
		compressedLength := int(binary.LittleEndian.Uint32(modelFile[offset+4:]))
		uncompressedLength := int(binary.LittleEndian.Uint32(modelFile[offset+8:]))
		offset += chunkHeaderLength
		if chunkName == "END\x00" {
			break
		}
		storedLength := compressedLength
		if compressedLength == 0 {
			storedLength = uncompressedLength
		}
		if storedLength < 0 || offset+storedLength > len(modelFile) || uncompressedLength > maxModelFileChunkBytes {
			return nil, errInvalidModelFile
		}
		storedData := modelFile[offset : offset+storedLength]
		offset += storedLength
		if chunkName != "PROP" {
			continue
		}
		decompressedTotal += uncompressedLength
		if decompressedTotal > maxModelFileDecompressedBytes {
			return nil, errInvalidModelFile
		}
		chunkData, err := decompressModelFileChunk(decoder, storedData, compressedLength, uncompressedLength)
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

func decompressModelFileChunk(decoder *zstd.Decoder, storedData []byte, compressedLength, uncompressedLength int) ([]byte, error) {
	if compressedLength == 0 {
		return storedData, nil
	}
	if bytes.HasPrefix(storedData, zstdFrameMagic) {
		chunkData, err := decoder.DecodeAll(storedData, make([]byte, 0, uncompressedLength))
		if err != nil || len(chunkData) != uncompressedLength {
			return nil, errInvalidModelFile
		}
		return chunkData, nil
	}
	return decompressLZ4Block(storedData, uncompressedLength)
}

// decompressLZ4Block decodes one LZ4 block. Each sequence starts with a token
// whose high 4 bits count literal bytes and whose low 4 bits count match bytes
// minus 4, where 15 continues the count in the following bytes. The literals
// follow, then a 2-byte little-endian offset back into the output for the
// match. The last sequence has literals only.
func decompressLZ4Block(compressed []byte, uncompressedLength int) ([]byte, error) {
	output := make([]byte, 0, uncompressedLength)
	for index := 0; index < len(compressed); {
		token := int(compressed[index])
		index++
		literalLength, ok := readLZ4Length(compressed, &index, token>>4)
		if !ok || index+literalLength > len(compressed) || len(output)+literalLength > uncompressedLength {
			return nil, errInvalidModelFile
		}
		output = append(output, compressed[index:index+literalLength]...)
		index += literalLength
		if index >= len(compressed) {
			break
		}
		if index+2 > len(compressed) {
			return nil, errInvalidModelFile
		}
		matchOffset := int(binary.LittleEndian.Uint16(compressed[index:]))
		index += 2
		matchLength, ok := readLZ4Length(compressed, &index, token&15)
		matchLength += 4
		if !ok || matchOffset == 0 || matchOffset > len(output) || len(output)+matchLength > uncompressedLength {
			return nil, errInvalidModelFile
		}
		// A match may overlap the bytes that it copies, so it copies one byte
		// at a time.
		matchStart := len(output) - matchOffset
		for copied := range matchLength {
			output = append(output, output[matchStart+copied])
		}
	}
	if len(output) != uncompressedLength {
		return nil, errInvalidModelFile
	}
	return output, nil
}

// readLZ4Length continues a length of 15 from the token with the bytes at
// index, which add up until one is below 255.
func readLZ4Length(compressed []byte, index *int, tokenLength int) (int, bool) {
	if tokenLength != 15 {
		return tokenLength, true
	}
	length := tokenLength
	for *index < len(compressed) {
		lengthByte := int(compressed[*index])
		*index++
		length += lengthByte
		if lengthByte != 255 {
			return length, true
		}
	}
	return 0, false
}

// assetIDsReferencedByText adds the asset IDs that the text names to
// assetIDs, without repeating one.
func assetIDsReferencedByText(text []byte, assetIDs []int64) []int64 {
	for _, match := range assetReferencePattern.FindAllSubmatch(text, -1) {
		assetID, err := strconv.ParseInt(string(match[1]), 10, 64)
		if err != nil || assetID <= 0 {
			continue
		}
		isListed := false
		for _, listedAssetID := range assetIDs {
			isListed = isListed || listedAssetID == assetID
		}
		if !isListed {
			assetIDs = append(assetIDs, assetID)
		}
		if len(assetIDs) >= maxReferencedAssetCount {
			break
		}
	}
	return assetIDs
}
