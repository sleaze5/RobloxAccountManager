package modelfile

import (
	"encoding/binary"
	"encoding/hex"
	"math"
	"strconv"
)

// maxBinaryClassCount and maxBinaryInstanceCount bound what a file can make
// the decoder allocate before its chunks prove that it holds that much.
const (
	maxBinaryClassCount    = 1 << 16
	maxBinaryInstanceCount = 1 << 20
)

// binaryClass is one INST chunk: a class and the referents of its instances.
type binaryClass struct {
	className string
	referents []int32
}

// decodeBinary reads the binary format. INST chunks list the instances of each
// class by referent, PROP chunks hold one property for every instance of a
// class, PRNT links each instance to its parent, and SSTR holds the shared
// strings that SharedString properties point to.
func decodeBinary(file []byte) ([]*decodedInstance, error) {
	chunks, err := splitBinaryChunks(file)
	if err != nil {
		return nil, err
	}
	decompressor, err := newChunkDecompressor()
	if err != nil {
		return nil, err
	}
	defer decompressor.close()
	decoder := binaryDecoder{classes: map[uint32]binaryClass{}, instanceByReferent: map[int32]*decodedInstance{}}
	for _, chunk := range chunks {
		if chunk.name != "INST" && chunk.name != "PROP" && chunk.name != "PRNT" && chunk.name != "SSTR" {
			continue
		}
		chunkData, err := decompressor.decompress(chunk)
		if err != nil {
			return nil, err
		}
		if err := decoder.readChunk(chunk.name, &byteReader{data: chunkData}); err != nil {
			return nil, err
		}
	}
	return decoder.roots, nil
}

// binaryDecoder collects the instances of a binary file chunk by chunk.
type binaryDecoder struct {
	classes             map[uint32]binaryClass
	instanceByReferent  map[int32]*decodedInstance
	sharedStringLengths []int
	roots               []*decodedInstance
}

func (decoder *binaryDecoder) readChunk(chunkName string, reader *byteReader) error {
	switch chunkName {
	case "INST":
		classID, class, err := readClassChunk(reader)
		if err != nil {
			return err
		}
		if len(decoder.classes) >= maxBinaryClassCount || len(decoder.instanceByReferent)+len(class.referents) > maxBinaryInstanceCount {
			return errInvalidFile
		}
		decoder.classes[classID] = class
		for _, referent := range class.referents {
			decoder.instanceByReferent[referent] = &decodedInstance{className: class.className, referent: strconv.Itoa(int(referent))}
		}
	case "SSTR":
		decoder.sharedStringLengths = readSharedStringLengths(reader)
	case "PROP":
		readPropertyChunk(reader, decoder.classes, decoder.instanceByReferent, decoder.sharedStringLengths)
	case "PRNT":
		roots, err := linkParents(reader, decoder.instanceByReferent)
		if err != nil {
			return err
		}
		decoder.roots = roots
	}
	return nil
}

func readClassChunk(reader *byteReader) (uint32, binaryClass, error) {
	classID := reader.uint32()
	className := string(reader.lengthPrefixed())
	isService := reader.byte() != 0
	instanceCount := int(reader.uint32())
	if reader.failed || instanceCount > maxBinaryInstanceCount {
		return 0, binaryClass{}, errInvalidFile
	}
	referents := reader.referents(instanceCount)
	if isService {
		reader.bytes(instanceCount)
	}
	if reader.failed {
		return 0, binaryClass{}, errInvalidFile
	}
	return classID, binaryClass{className: className, referents: referents}, nil
}

func readSharedStringLengths(reader *byteReader) []int {
	reader.uint32()
	count := int(reader.uint32())
	var lengths []int
	for range count {
		reader.bytes(16)
		value := reader.lengthPrefixed()
		if reader.failed {
			return lengths
		}
		lengths = append(lengths, len(value))
	}
	return lengths
}

// linkParents reads the PRNT chunk, which lists child and parent referents.
// A parent referent of -1 makes the child a root of the file.
func linkParents(reader *byteReader, instanceByReferent map[int32]*decodedInstance) ([]*decodedInstance, error) {
	reader.byte()
	count := int(reader.uint32())
	if reader.failed || count > len(instanceByReferent) {
		return nil, errInvalidFile
	}
	childReferents := reader.referents(count)
	parentReferents := reader.referents(count)
	if reader.failed {
		return nil, errInvalidFile
	}
	var roots []*decodedInstance
	for index, childReferent := range childReferents {
		child := instanceByReferent[childReferent]
		if child == nil {
			continue
		}
		parent := instanceByReferent[parentReferents[index]]
		if parent == nil || parent == child {
			roots = append(roots, child)
			continue
		}
		parent.children = append(parent.children, child)
	}
	return roots, nil
}

// readPropertyChunk reads one property of every instance of a class. A
// property of a type that the explorer does not read, or whose data is
// damaged, is left out, because each chunk stands on its own.
func readPropertyChunk(reader *byteReader, classes map[uint32]binaryClass, instanceByReferent map[int32]*decodedInstance, sharedStringLengths []int) {
	classID := reader.uint32()
	propertyName := string(reader.lengthPrefixed())
	typeID := reader.byte()
	class, hasClass := classes[classID]
	if reader.failed || !hasClass {
		return
	}
	properties, rawValues := readPropertyValues(reader, propertyName, typeID, len(class.referents), sharedStringLengths)
	if reader.failed || len(properties) != len(class.referents) {
		return
	}
	for index, referent := range class.referents {
		var rawValue []byte
		if rawValues != nil {
			rawValue = rawValues[index]
		}
		instanceByReferent[referent].addProperty(properties[index], rawValue)
	}
}

// propertyColumn is the stored values of one property of every instance of a
// class.
type propertyColumn struct {
	reader              *byteReader
	name                string
	count               int
	sharedStringLengths []int
}

// columnReaders read the values of each property type, by type ID. Most types
// store each field of every value together, and numbers byte-interleaved, so
// that they compress well. A reader returns fewer values than the column
// holds, and marks the reader failed, when the data is damaged.
var columnReaders = map[byte]func(column propertyColumn) []Property{
	0x02: func(column propertyColumn) []Property {
		return mapValues(column.reader.bytes(column.count), func(value byte) Property { return booleanProperty(column.name, value != 0) })
	},
	0x03: func(column propertyColumn) []Property {
		return mapValues(column.reader.int32s(column.count), func(value int32) Property { return integerProperty(column.name, int64(value)) })
	},
	0x04: func(column propertyColumn) []Property {
		return mapValues(column.reader.floats(column.count), func(value float64) Property { return numberProperty(column.name, value) })
	},
	0x05: func(column propertyColumn) []Property {
		return repeatValues(column.count, func() Property {
			return numberProperty(column.name, math.Float64frombits(column.reader.uint64()))
		})
	},
	0x06: readUDims,
	0x07: readUDim2s,
	0x08: func(column propertyColumn) []Property {
		return repeatValues(column.count, func() Property {
			origin, direction := column.reader.littleEndianFloats(3), column.reader.littleEndianFloats(3)
			return rayProperty(column.name, origin, direction)
		})
	},
	0x09: func(column propertyColumn) []Property {
		return mapValues(column.reader.bytes(column.count), func(value byte) Property { return structuredProperty(column.name, flagNames(value, faceNames)) })
	},
	0x0a: func(column propertyColumn) []Property {
		return mapValues(column.reader.bytes(column.count), func(value byte) Property { return structuredProperty(column.name, flagNames(value, axisNames)) })
	},
	0x0b: func(column propertyColumn) []Property {
		return mapValues(column.reader.uint32s(column.count), func(value uint32) Property { return brickColorProperty(column.name, int64(value)) })
	},
	0x0c: readColor3s,
	0x0d: readVector2s,
	0x0e: readVector3s,
	0x10: readCFrames,
	0x12: func(column propertyColumn) []Property {
		return mapValues(column.reader.uint32s(column.count), func(value uint32) Property { return enumProperty(column.name, value) })
	},
	0x13: func(column propertyColumn) []Property {
		return mapValues(column.reader.referents(column.count), func(value int32) Property {
			referent := ""
			if value != -1 {
				referent = strconv.Itoa(int(value))
			}
			return referenceProperty(column.name, referent)
		})
	},
	0x14: func(column propertyColumn) []Property {
		return repeatValues(column.count, func() Property {
			x, y, z := int16(column.reader.uint16()), int16(column.reader.uint16()), int16(column.reader.uint16())
			return vectorProperty(column.name, float64(x), float64(y), float64(z))
		})
	},
	0x15: readNumberSequences,
	0x16: readColorSequences,
	0x17: func(column propertyColumn) []Property {
		return repeatValues(column.count, func() Property {
			return structuredProperty(column.name, formatNumbers(column.reader.littleEndianFloats(2)...))
		})
	},
	0x18: readRects,
	0x19: readPhysicalProperties,
	0x1a: readColor3uint8s,
	0x1b: func(column propertyColumn) []Property {
		return mapValues(column.reader.int64s(column.count), func(value int64) Property { return integerProperty(column.name, value) })
	},
	0x1c: func(column propertyColumn) []Property {
		return mapValues(column.reader.uint32s(column.count), func(sharedStringIndex uint32) Property {
			length := 0
			if int(sharedStringIndex) < len(column.sharedStringLengths) {
				length = column.sharedStringLengths[sharedStringIndex]
			}
			return binaryProperty(column.name, length)
		})
	},
	0x1e: readOptionalCFrames,
	0x1f: func(column propertyColumn) []Property {
		return mapValues(column.reader.interleaved(column.count, 16), func(value []byte) Property { return structuredProperty(column.name, hex.EncodeToString(value)) })
	},
	0x20: func(column propertyColumn) []Property {
		return repeatValues(column.count, func() Property {
			family := string(column.reader.lengthPrefixed())
			weight := column.reader.uint16()
			style := column.reader.byte()
			column.reader.lengthPrefixed()
			return structuredProperty(column.name, fontText(family, weight, style))
		})
	},
	0x21: func(column propertyColumn) []Property {
		return mapValues(column.reader.int64s(column.count), func(value int64) Property { return integerProperty(column.name, value) })
	},
	0x22: readContents,
}

// readPropertyValues reads count values of one type. It also returns the raw
// bytes of string values, which hold attributes and tags.
func readPropertyValues(reader *byteReader, name string, typeID byte, count int, sharedStringLengths []int) ([]Property, [][]byte) {
	// Strings, and protected strings such as the source of a script.
	if typeID == 0x01 || typeID == 0x1d {
		rawValues := make([][]byte, count)
		properties := make([]Property, count)
		for index := range count {
			rawValues[index] = reader.lengthPrefixed()
			properties[index] = stringProperty(name, rawValues[index])
		}
		return properties, rawValues
	}
	readColumn, isKnown := columnReaders[typeID]
	if !isKnown {
		reader.failed = true
		return nil, nil
	}
	return readColumn(propertyColumn{reader: reader, name: name, count: count, sharedStringLengths: sharedStringLengths}), nil
}

func mapValues[Value any](values []Value, toProperty func(Value) Property) []Property {
	properties := make([]Property, len(values))
	for index, value := range values {
		properties[index] = toProperty(value)
	}
	return properties
}

// repeatValues reads count values that each store their fields together.
func repeatValues(count int, readValue func() Property) []Property {
	properties := make([]Property, count)
	for index := range properties {
		properties[index] = readValue()
	}
	return properties
}

func readUDims(column propertyColumn) []Property {
	scales, offsets := column.reader.floats(column.count), column.reader.int32s(column.count)
	properties := make([]Property, min(len(scales), len(offsets)))
	for index := range properties {
		properties[index] = structuredProperty(column.name, "{"+formatNumber(scales[index])+", "+strconv.Itoa(int(offsets[index]))+"}")
	}
	return properties
}

func readUDim2s(column propertyColumn) []Property {
	reader, count := column.reader, column.count
	scalesX, scalesY, offsetsX, offsetsY := reader.floats(count), reader.floats(count), reader.int32s(count), reader.int32s(count)
	properties := make([]Property, min(len(scalesX), len(scalesY), len(offsetsX), len(offsetsY)))
	for index := range properties {
		properties[index] = structuredProperty(column.name, udim2Text(scalesX[index], int64(offsetsX[index]), scalesY[index], int64(offsetsY[index])))
	}
	return properties
}

func readColor3s(column propertyColumn) []Property {
	reds, greens, blues := column.reader.floats(column.count), column.reader.floats(column.count), column.reader.floats(column.count)
	properties := make([]Property, min(len(reds), len(greens), len(blues)))
	for index := range properties {
		properties[index] = colorProperty(column.name, reds[index], greens[index], blues[index])
	}
	return properties
}

func readColor3uint8s(column propertyColumn) []Property {
	reds, greens, blues := column.reader.bytes(column.count), column.reader.bytes(column.count), column.reader.bytes(column.count)
	properties := make([]Property, min(len(reds), len(greens), len(blues)))
	for index := range properties {
		properties[index] = color8Property(column.name, reds[index], greens[index], blues[index])
	}
	return properties
}

func readVector2s(column propertyColumn) []Property {
	xs, ys := column.reader.floats(column.count), column.reader.floats(column.count)
	properties := make([]Property, min(len(xs), len(ys)))
	for index := range properties {
		properties[index] = vectorProperty(column.name, xs[index], ys[index])
	}
	return properties
}

func readVector3s(column propertyColumn) []Property {
	xs, ys, zs := column.reader.floats(column.count), column.reader.floats(column.count), column.reader.floats(column.count)
	properties := make([]Property, min(len(xs), len(ys), len(zs)))
	for index := range properties {
		properties[index] = vectorProperty(column.name, xs[index], ys[index], zs[index])
	}
	return properties
}

func readRects(column propertyColumn) []Property {
	reader, count := column.reader, column.count
	minXs, minYs, maxXs, maxYs := reader.floats(count), reader.floats(count), reader.floats(count), reader.floats(count)
	properties := make([]Property, min(len(minXs), len(minYs), len(maxXs), len(maxYs)))
	for index := range properties {
		properties[index] = structuredProperty(column.name, "{"+formatNumbers(minXs[index], minYs[index])+"}, {"+formatNumbers(maxXs[index], maxYs[index])+"}")
	}
	return properties
}

// readNumberSequences reads, for each value, a keypoint count and then the
// time, value, and envelope of each keypoint.
func readNumberSequences(column propertyColumn) []Property {
	return repeatValues(column.count, func() Property {
		keypointCount := int(column.reader.uint32())
		if keypointCount > column.reader.remaining()/12 {
			column.reader.failed = true
			return Property{}
		}
		keypoints := make([][3]float64, keypointCount)
		for keypointIndex := range keypoints {
			values := column.reader.littleEndianFloats(3)
			keypoints[keypointIndex] = [3]float64{values[0], values[1], values[2]}
		}
		return structuredProperty(column.name, numberSequenceText(keypoints))
	})
}

// readColorSequences reads, for each value, a keypoint count and then the
// time, red, green, blue, and envelope of each keypoint.
func readColorSequences(column propertyColumn) []Property {
	return repeatValues(column.count, func() Property {
		keypointCount := int(column.reader.uint32())
		if keypointCount > column.reader.remaining()/20 {
			column.reader.failed = true
			return Property{}
		}
		keypoints := make([][4]float64, keypointCount)
		for keypointIndex := range keypoints {
			values := column.reader.littleEndianFloats(5)
			keypoints[keypointIndex] = [4]float64{values[0], values[1], values[2], values[3]}
		}
		return structuredProperty(column.name, colorSequenceText(keypoints))
	})
}

// readPhysicalProperties reads a flag byte per value. Bit 0 marks custom
// values, and bit 1 an acoustic absorption value after them.
func readPhysicalProperties(column propertyColumn) []Property {
	return repeatValues(column.count, func() Property {
		flags := column.reader.byte()
		var values []float64
		if flags&1 != 0 {
			values = column.reader.littleEndianFloats(5)
			if flags&2 != 0 {
				column.reader.littleEndianFloats(1)
			}
		}
		return structuredProperty(column.name, physicalPropertiesText(flags&1 != 0, values))
	})
}

// readCFrames reads CFrames. Each rotation is an ID byte, followed by 9
// little-endian floats when the ID is 0. The positions follow as three
// interleaved float arrays.
func readCFrames(column propertyColumn) []Property {
	reader := column.reader
	rotations := make([][9]float64, column.count)
	for index := range rotations {
		rotationID := reader.byte()
		if rotationID != 0 {
			rotation, isKnown := specialRotation(rotationID)
			if !isKnown {
				reader.failed = true
				return nil
			}
			rotations[index] = rotation
			continue
		}
		copy(rotations[index][:], reader.littleEndianFloats(9))
	}
	xs, ys, zs := reader.floats(column.count), reader.floats(column.count), reader.floats(column.count)
	properties := make([]Property, min(len(xs), len(ys), len(zs)))
	for index := range properties {
		properties[index] = cframeProperty(column.name, [3]float64{xs[index], ys[index], zs[index]}, rotations[index])
	}
	return properties
}

// readOptionalCFrames reads a CFrame array and then a bool array that tells
// which values are set, each after its type ID.
func readOptionalCFrames(column propertyColumn) []Property {
	if column.reader.byte() != 0x10 {
		column.reader.failed = true
		return nil
	}
	properties := readCFrames(column)
	if column.reader.byte() != 0x02 {
		column.reader.failed = true
		return nil
	}
	for index, isSet := range column.reader.bytes(column.count) {
		if isSet == 0 && index < len(properties) {
			properties[index] = structuredProperty(column.name, "None")
		}
	}
	return properties
}

// readContents reads Content values: the source type of each value, then the
// URIs of the values that hold one, in order, then the referents of the
// values that hold an object.
func readContents(column propertyColumn) []Property {
	const (
		contentURI    = 1
		contentObject = 2
	)
	reader, name := column.reader, column.name
	sourceTypes := reader.int32s(column.count)
	uriCount := int(reader.uint32())
	if reader.failed || uriCount > column.count {
		reader.failed = true
		return nil
	}
	uris := make([]string, uriCount)
	for index := range uris {
		uris[index] = string(reader.lengthPrefixed())
	}
	objectCount := int(reader.uint32())
	if reader.failed || objectCount > column.count {
		reader.failed = true
		return nil
	}
	objectReferents := reader.referents(objectCount)
	properties := make([]Property, len(sourceTypes))
	uriIndex, objectIndex := 0, 0
	for index, sourceType := range sourceTypes {
		switch {
		case sourceType == contentURI && uriIndex < len(uris):
			properties[index] = contentProperty(name, uris[uriIndex])
			uriIndex++
		case sourceType == contentObject && objectIndex < len(objectReferents):
			properties[index] = referenceProperty(name, strconv.Itoa(int(objectReferents[objectIndex])))
			objectIndex++
		default:
			properties[index] = contentProperty(name, "")
		}
	}
	return properties
}

// byteReader reads little-endian values from a chunk. A read past the end
// marks it failed and returns zero values, so that a caller checks failed
// once after a series of reads.
type byteReader struct {
	data   []byte
	offset int
	failed bool
}

func (reader *byteReader) remaining() int {
	return len(reader.data) - reader.offset
}

func (reader *byteReader) bytes(count int) []byte {
	if reader.failed || count < 0 || count > reader.remaining() {
		reader.failed = true
		return nil
	}
	value := reader.data[reader.offset : reader.offset+count]
	reader.offset += count
	return value
}

func (reader *byteReader) byte() byte {
	value := reader.bytes(1)
	if value == nil {
		return 0
	}
	return value[0]
}

func (reader *byteReader) uint16() uint16 {
	value := reader.bytes(2)
	if value == nil {
		return 0
	}
	return binary.LittleEndian.Uint16(value)
}

func (reader *byteReader) uint32() uint32 {
	value := reader.bytes(4)
	if value == nil {
		return 0
	}
	return binary.LittleEndian.Uint32(value)
}

func (reader *byteReader) uint64() uint64 {
	value := reader.bytes(8)
	if value == nil {
		return 0
	}
	return binary.LittleEndian.Uint64(value)
}

func (reader *byteReader) lengthPrefixed() []byte {
	return reader.bytes(int(reader.uint32()))
}

func (reader *byteReader) littleEndianFloats(count int) []float64 {
	values := make([]float64, count)
	for index := range values {
		values[index] = float64(math.Float32frombits(reader.uint32()))
	}
	return values
}

// interleaved reads count values of width bytes that are stored with the
// first byte of every value, then the second byte of every value, and so on.
// It returns each value with its bytes in order, big endian.
func (reader *byteReader) interleaved(count, width int) [][]byte {
	if count > reader.remaining()/width {
		reader.failed = true
		return nil
	}
	stored := reader.bytes(count * width)
	if stored == nil {
		return nil
	}
	values := make([][]byte, count)
	for index := range values {
		value := make([]byte, width)
		for byteIndex := range width {
			value[byteIndex] = stored[byteIndex*count+index]
		}
		values[index] = value
	}
	return values
}

func (reader *byteReader) uint32s(count int) []uint32 {
	values := make([]uint32, 0, count)
	for _, value := range reader.interleaved(count, 4) {
		values = append(values, binary.BigEndian.Uint32(value))
	}
	return values
}

// int32s reads interleaved integers stored in zigzag form, which moves the
// sign to the lowest bit so that small negative numbers compress well.
func (reader *byteReader) int32s(count int) []int32 {
	stored := reader.uint32s(count)
	values := make([]int32, len(stored))
	for index, value := range stored {
		values[index] = int32(value>>1) ^ -int32(value&1)
	}
	return values
}

func (reader *byteReader) int64s(count int) []int64 {
	values := make([]int64, 0, count)
	for _, value := range reader.interleaved(count, 8) {
		stored := binary.BigEndian.Uint64(value)
		values = append(values, int64(stored>>1)^-int64(stored&1))
	}
	return values
}

// floats reads interleaved 32-bit floats whose sign bit is rotated to the
// lowest bit.
func (reader *byteReader) floats(count int) []float64 {
	stored := reader.uint32s(count)
	values := make([]float64, len(stored))
	for index, value := range stored {
		values[index] = float64(math.Float32frombits(value>>1 | value<<31))
	}
	return values
}

// referents reads interleaved zigzag integers that each add to the previous
// referent.
func (reader *byteReader) referents(count int) []int32 {
	values := reader.int32s(count)
	var previous int32
	for index, value := range values {
		previous += value
		values[index] = previous
	}
	return values
}
