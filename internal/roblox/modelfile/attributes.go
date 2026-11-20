package modelfile

import (
	"math"
	"strconv"
)

// maxAttributeCount bounds the attributes that one instance shows.
const maxAttributeCount = 1024

// decodeAttributes reads the attributes that a file stores in the
// AttributesSerialize property: a count, then each attribute as its name, a
// type byte, and its value, all little endian. Reading stops at a type that it
// does not know, because the length of its value is unknown.
func decodeAttributes(serialized []byte) []Property {
	reader := &byteReader{data: serialized}
	count := int(reader.uint32())
	var attributes []Property
	for range min(count, maxAttributeCount) {
		name := string(reader.lengthPrefixed())
		attribute, isKnown := readAttributeValue(reader, name)
		if !isKnown || reader.failed {
			break
		}
		attribute.Group = AttributesGroup
		attributes = append(attributes, attribute)
	}
	return attributes
}

// attributeReaders read the value of each attribute type, by type byte. A
// reader reports false when the value is damaged.
var attributeReaders = map[byte]func(reader *byteReader, name string) (Property, bool){
	0x02: func(reader *byteReader, name string) (Property, bool) {
		return stringProperty(name, reader.lengthPrefixed()), true
	},
	0x03: func(reader *byteReader, name string) (Property, bool) {
		return booleanProperty(name, reader.byte() != 0), true
	},
	0x04: func(reader *byteReader, name string) (Property, bool) {
		return integerProperty(name, int64(int32(reader.uint32()))), true
	},
	0x05: func(reader *byteReader, name string) (Property, bool) {
		return numberProperty(name, float64(math.Float32frombits(reader.uint32()))), true
	},
	0x06: func(reader *byteReader, name string) (Property, bool) {
		return numberProperty(name, math.Float64frombits(reader.uint64())), true
	},
	0x09: func(reader *byteReader, name string) (Property, bool) {
		scale := float64(math.Float32frombits(reader.uint32()))
		offset := int32(reader.uint32())
		return structuredProperty(name, "{"+formatNumber(scale)+", "+strconv.Itoa(int(offset))+"}"), true
	},
	0x0a: func(reader *byteReader, name string) (Property, bool) {
		scaleX, offsetX := float64(math.Float32frombits(reader.uint32())), int32(reader.uint32())
		scaleY, offsetY := float64(math.Float32frombits(reader.uint32())), int32(reader.uint32())
		return structuredProperty(name, udim2Text(scaleX, int64(offsetX), scaleY, int64(offsetY))), true
	},
	0x0e: func(reader *byteReader, name string) (Property, bool) {
		return integerProperty(name, int64(reader.uint32())), true
	},
	0x0f: func(reader *byteReader, name string) (Property, bool) {
		channels := reader.littleEndianFloats(3)
		return colorProperty(name, channels[0], channels[1], channels[2]), true
	},
	0x10: func(reader *byteReader, name string) (Property, bool) {
		return structuredProperty(name, formatNumbers(reader.littleEndianFloats(2)...)), true
	},
	0x11: func(reader *byteReader, name string) (Property, bool) {
		return structuredProperty(name, formatNumbers(reader.littleEndianFloats(3)...)), true
	},
	0x14: func(reader *byteReader, name string) (Property, bool) {
		return readAttributeCFrame(reader, name), true
	},
	0x15: func(reader *byteReader, name string) (Property, bool) {
		enumName := string(reader.lengthPrefixed())
		value := reader.uint32()
		return structuredProperty(name, "Enum."+enumName+" "+strconv.FormatUint(uint64(value), 10)), true
	},
	0x17: func(reader *byteReader, name string) (Property, bool) {
		count := int(reader.uint32())
		if count > reader.remaining()/12 {
			return Property{}, false
		}
		keypoints := make([][3]float64, count)
		for index := range keypoints {
			values := reader.littleEndianFloats(3)
			// An attribute stores the envelope first, then the time and value.
			keypoints[index] = [3]float64{values[1], values[2], values[0]}
		}
		return structuredProperty(name, numberSequenceText(keypoints)), true
	},
	0x19: func(reader *byteReader, name string) (Property, bool) {
		count := int(reader.uint32())
		if count > reader.remaining()/20 {
			return Property{}, false
		}
		keypoints := make([][4]float64, count)
		for index := range keypoints {
			values := reader.littleEndianFloats(5)
			keypoints[index] = [4]float64{values[1], values[2], values[3], values[4]}
		}
		return structuredProperty(name, colorSequenceText(keypoints)), true
	},
	0x1b: func(reader *byteReader, name string) (Property, bool) {
		return structuredProperty(name, formatNumbers(reader.littleEndianFloats(2)...)), true
	},
	0x1c: func(reader *byteReader, name string) (Property, bool) {
		corners := reader.littleEndianFloats(4)
		return structuredProperty(name, "{"+formatNumbers(corners[0], corners[1])+"}, {"+formatNumbers(corners[2], corners[3])+"}"), true
	},
	0x21: func(reader *byteReader, name string) (Property, bool) {
		weight := reader.uint16()
		style := reader.byte()
		family := string(reader.lengthPrefixed())
		reader.lengthPrefixed()
		return structuredProperty(name, fontText(family, weight, style)), true
	},
}

func readAttributeValue(reader *byteReader, name string) (Property, bool) {
	readValue, isKnown := attributeReaders[reader.byte()]
	if !isKnown {
		return Property{}, false
	}
	return readValue(reader, name)
}

// readAttributeCFrame reads a CFrame attribute: its position, then a rotation
// ID byte, followed by 9 floats when the ID is 0.
func readAttributeCFrame(reader *byteReader, name string) Property {
	positionValues := reader.littleEndianFloats(3)
	position := [3]float64{positionValues[0], positionValues[1], positionValues[2]}
	rotationID := reader.byte()
	rotation := identityRotation
	if rotationID == 0 {
		copy(rotation[:], reader.littleEndianFloats(9))
	} else if special, isKnown := specialRotation(rotationID); isKnown {
		rotation = special
	}
	return structuredProperty(name, cframeText(position, rotation))
}
