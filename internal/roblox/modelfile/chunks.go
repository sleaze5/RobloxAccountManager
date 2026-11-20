// Package modelfile reads Roblox model files in the binary format (rbxm) and
// the XML format (rbxmx).
package modelfile

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/klauspost/compress/zstd"
)

const (
	maxChunkBytes        = 16 << 20
	maxDecompressedBytes = 64 << 20
	binaryHeaderLength   = 32
	chunkHeaderLength    = 16
)

var (
	binarySignature = []byte("<roblox!\x89\xff\x0d\x0a\x1a\x0a")
	xmlSignature    = []byte("<roblox ")
	zstdFrameMagic  = []byte{0x28, 0xb5, 0x2f, 0xfd}
	errInvalidFile  = errors.New("invalid Roblox model file")
)

// IsBinary reports whether the file starts like a model file in the binary
// format.
func IsBinary(file []byte) bool {
	return bytes.HasPrefix(file, binarySignature)
}

// IsXML reports whether the file starts like a model file in the XML format.
func IsXML(file []byte) bool {
	return bytes.HasPrefix(file, xmlSignature)
}

// binaryChunk is one chunk of the binary format, with its data as the file
// stores it.
type binaryChunk struct {
	name               string
	storedData         []byte
	compressedLength   int
	uncompressedLength int
}

// splitBinaryChunks lists the chunks of the binary format without
// decompressing them. After a 32-byte header, each chunk has a 4-byte name,
// its compressed and uncompressed lengths and 4 reserved bytes, all little
// endian, then its data. A compressed length of 0 means the data is stored as
// is. The END chunk closes the file.
func splitBinaryChunks(file []byte) ([]binaryChunk, error) {
	if len(file) < binaryHeaderLength || !IsBinary(file) {
		return nil, errInvalidFile
	}
	var chunks []binaryChunk
	for offset := binaryHeaderLength; offset+chunkHeaderLength <= len(file); {
		chunk := binaryChunk{
			name:               string(file[offset : offset+4]),
			compressedLength:   int(binary.LittleEndian.Uint32(file[offset+4:])),
			uncompressedLength: int(binary.LittleEndian.Uint32(file[offset+8:])),
		}
		offset += chunkHeaderLength
		if chunk.name == "END\x00" {
			break
		}
		storedLength := chunk.compressedLength
		if storedLength == 0 {
			storedLength = chunk.uncompressedLength
		}
		if storedLength < 0 || offset+storedLength > len(file) || chunk.uncompressedLength > maxChunkBytes {
			return nil, errInvalidFile
		}
		chunk.storedData = file[offset : offset+storedLength]
		offset += storedLength
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

// chunkDecompressor decompresses the chunks of one file and bounds their total
// size. Roblox compresses chunks with Zstandard in newer files and LZ4 in
// older ones.
type chunkDecompressor struct {
	zstdDecoder            *zstd.Decoder
	decompressedTotalBytes int
}

func newChunkDecompressor() (*chunkDecompressor, error) {
	zstdDecoder, err := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1), zstd.WithDecoderMaxMemory(maxChunkBytes))
	if err != nil {
		return nil, fmt.Errorf("create zstd decoder: %w", err)
	}
	return &chunkDecompressor{zstdDecoder: zstdDecoder}, nil
}

func (decompressor *chunkDecompressor) close() {
	decompressor.zstdDecoder.Close()
}

func (decompressor *chunkDecompressor) decompress(chunk binaryChunk) ([]byte, error) {
	decompressor.decompressedTotalBytes += chunk.uncompressedLength
	if decompressor.decompressedTotalBytes > maxDecompressedBytes {
		return nil, errInvalidFile
	}
	if chunk.compressedLength == 0 {
		return chunk.storedData, nil
	}
	if bytes.HasPrefix(chunk.storedData, zstdFrameMagic) {
		chunkData, err := decompressor.zstdDecoder.DecodeAll(chunk.storedData, make([]byte, 0, chunk.uncompressedLength))
		if err != nil || len(chunkData) != chunk.uncompressedLength {
			return nil, errInvalidFile
		}
		return chunkData, nil
	}
	return decompressLZ4Block(chunk.storedData, chunk.uncompressedLength)
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
			return nil, errInvalidFile
		}
		output = append(output, compressed[index:index+literalLength]...)
		index += literalLength
		if index >= len(compressed) {
			break
		}
		if index+2 > len(compressed) {
			return nil, errInvalidFile
		}
		matchOffset := int(binary.LittleEndian.Uint16(compressed[index:]))
		index += 2
		matchLength, ok := readLZ4Length(compressed, &index, token&15)
		matchLength += 4
		if !ok || matchOffset == 0 || matchOffset > len(output) || len(output)+matchLength > uncompressedLength {
			return nil, errInvalidFile
		}
		// A match may overlap the bytes that it copies, so it copies one byte
		// at a time.
		matchStart := len(output) - matchOffset
		for copied := range matchLength {
			output = append(output, output[matchStart+copied])
		}
	}
	if len(output) != uncompressedLength {
		return nil, errInvalidFile
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
