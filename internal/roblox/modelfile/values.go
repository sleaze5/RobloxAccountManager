package modelfile

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// maxRowTextLength is the longest string that shows in a row of the explorer.
// Longer strings open as a document.
const maxRowTextLength = 120

// stringProperty shows a string as text, a document, an asset, or binary
// data, depending on what it holds.
func stringProperty(name string, value []byte) Property {
	if !utf8.Valid(value) || bytes.IndexByte(value, 0) >= 0 {
		return binaryProperty(name, len(value))
	}
	text := string(value)
	if strings.ContainsAny(text, "\r\n") || len(text) > maxRowTextLength {
		return Property{Name: name, Kind: PropertyDocument, Text: text}
	}
	return contentProperty(name, text)
}

// contentProperty shows a content value, which names an asset when it holds
// an asset ID.
func contentProperty(name, content string) Property {
	if assetID := referencedAssetID(content); assetID > 0 {
		return Property{Name: name, Kind: PropertyAsset, Text: content, AssetID: assetID}
	}
	return Property{Name: name, Kind: PropertyText, Text: content}
}

func binaryProperty(name string, lengthBytes int) Property {
	return Property{Name: name, Kind: PropertyBinary, Text: strconv.Itoa(lengthBytes)}
}

func booleanProperty(name string, value bool) Property {
	return Property{Name: name, Kind: PropertyBoolean, Text: strconv.FormatBool(value)}
}

func numberProperty(name string, value float64) Property {
	return Property{Name: name, Kind: PropertyNumber, Text: formatNumber(value)}
}

func integerProperty(name string, value int64) Property {
	return Property{Name: name, Kind: PropertyNumber, Text: strconv.FormatInt(value, 10)}
}

func enumProperty(name string, value uint32) Property {
	return Property{Name: name, Kind: PropertyEnum, Text: strconv.FormatUint(uint64(value), 10), EnumValue: value}
}

func referenceProperty(name, referent string) Property {
	return Property{Name: name, Kind: PropertyReference, referent: referent}
}

func structuredProperty(name, text string) Property {
	return Property{Name: name, Kind: PropertyStructured, Text: text}
}

// colorProperty shows a color whose channels range from 0 to 1, as Color3
// stores them, in the 0 to 255 range that Roblox Studio shows.
func colorProperty(name string, red, green, blue float64) Property {
	return color8Property(name, colorChannelByte(red), colorChannelByte(green), colorChannelByte(blue))
}

func color8Property(name string, red, green, blue uint8) Property {
	return Property{
		Name: name, Kind: PropertyColor,
		Text:  fmt.Sprintf("%d, %d, %d", red, green, blue),
		Color: fmt.Sprintf("#%02x%02x%02x", red, green, blue),
	}
}

// brickColorProperty shows a BrickColor as its name and color. A number
// outside the palette shows as is.
func brickColorProperty(name string, number int64) Property {
	color, isKnown := brickColorByNumber[number]
	if !isKnown {
		return integerProperty(name, number)
	}
	property := color8Property(name, color.red, color.green, color.blue)
	property.Text = color.name
	return property
}

// ShowAsBrickColor shows a number as the BrickColor that it names. Older
// files store BrickColor properties, such as the BrickColor of a part, as
// plain numbers, so only a class index can tell them apart.
func (property *Property) ShowAsBrickColor() {
	if property.Kind != PropertyNumber {
		return
	}
	number, err := strconv.ParseInt(property.Text, 10, 64)
	if err != nil {
		return
	}
	group := property.Group
	*property = brickColorProperty(property.Name, number)
	property.Group = group
}

func colorChannelByte(channel float64) uint8 {
	if math.IsNaN(channel) {
		return 0
	}
	return uint8(math.Round(math.Max(0, math.Min(1, channel)) * 255))
}

// formatNumber rounds a number to 3 decimals, as Roblox Studio shows it, so
// that a 32-bit float such as 0.300000012 shows as 0.3.
func formatNumber(value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return strconv.FormatFloat(value, 'f', -1, 64)
	}
	rounded := math.Round(value*1000) / 1000
	if rounded == 0 {
		rounded = 0
	}
	return strconv.FormatFloat(rounded, 'f', -1, 64)
}

func formatNumbers(values ...float64) string {
	return strings.Join(formatComponents(values), ", ")
}

func formatComponents(values []float64) []string {
	texts := make([]string, len(values))
	for index, value := range values {
		texts[index] = formatNumber(value)
	}
	return texts
}

// vectorProperty shows a Vector2 or a Vector3.
func vectorProperty(name string, components ...float64) Property {
	return vectorPartsProperty(name, VectorPart{Components: formatComponents(components)})
}

// vectorPartsProperty shows a value made of vectors, such as a CFrame. Its
// text writes each vector on its own line.
func vectorPartsProperty(name string, parts ...VectorPart) Property {
	lines := make([]string, len(parts))
	for index, part := range parts {
		lines[index] = strings.Join(part.Components, ", ")
		if part.Label != "" {
			lines[index] = part.Label + ": " + lines[index]
		}
	}
	return Property{Name: name, Kind: PropertyVector, Text: strings.Join(lines, "\n"), VectorParts: parts}
}

func rayProperty(name string, origin, direction []float64) Property {
	return vectorPartsProperty(name,
		VectorPart{Label: "Origin", Components: formatComponents(origin)},
		VectorPart{Label: "Direction", Components: formatComponents(direction)},
	)
}

// cframeProperty shows the position of a CFrame and its orientation in
// degrees, which Roblox Studio derives from the rotation matrix in Y, X, Z
// order. The matrix is in row-major order.
func cframeProperty(name string, position [3]float64, rotation [9]float64) Property {
	x := math.Asin(math.Max(-1, math.Min(1, -rotation[5])))
	var y, z float64
	if math.Abs(rotation[5]) < 0.99999 {
		y = math.Atan2(rotation[2], rotation[8])
		z = math.Atan2(rotation[3], rotation[4])
	} else {
		y = math.Atan2(-rotation[6], rotation[0])
	}
	degrees := 180 / math.Pi
	return vectorPartsProperty(name,
		VectorPart{Label: "Position", Components: formatComponents(position[:])},
		VectorPart{Label: "Orientation", Components: formatComponents([]float64{x * degrees, y * degrees, z * degrees})},
	)
}

// identityRotation is the rotation of a CFrame that a file stores without
// one.
var identityRotation = [9]float64{1, 0, 0, 0, 1, 0, 0, 0, 1}

// specialRotation builds one of the 24 axis-aligned rotations that the binary
// format stores as an ID. The ID picks the right and up vectors among the six
// axis directions, and the back vector is their cross product.
func specialRotation(rotationID byte) ([9]float64, bool) {
	axisDirections := [6][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {-1, 0, 0}, {0, -1, 0}, {0, 0, -1}}
	index := int(rotationID) - 1
	if index < 0 || index >= 36 {
		return [9]float64{}, false
	}
	right, up := axisDirections[index/6], axisDirections[index%6]
	back := [3]float64{
		right[1]*up[2] - up[1]*right[2],
		right[2]*up[0] - up[2]*right[0],
		right[0]*up[1] - up[0]*right[1],
	}
	var rotation [9]float64
	for row := range 3 {
		rotation[row*3], rotation[row*3+1], rotation[row*3+2] = right[row], up[row], back[row]
	}
	return rotation, true
}

func udim2Text(scaleX float64, offsetX int64, scaleY float64, offsetY int64) string {
	return fmt.Sprintf("{%s, %d}, {%s, %d}", formatNumber(scaleX), offsetX, formatNumber(scaleY), offsetY)
}

// numberSequenceText writes the keypoints of a NumberSequence as time and
// value pairs.
func numberSequenceText(keypoints [][3]float64) string {
	texts := make([]string, len(keypoints))
	for index, keypoint := range keypoints {
		texts[index] = "(" + formatNumbers(keypoint[0], keypoint[1]) + ")"
	}
	return strings.Join(texts, " ")
}

// colorSequenceText writes the keypoints of a ColorSequence as times and 0 to
// 255 colors.
func colorSequenceText(keypoints [][4]float64) string {
	texts := make([]string, len(keypoints))
	for index, keypoint := range keypoints {
		texts[index] = fmt.Sprintf("(%s: %d, %d, %d)", formatNumber(keypoint[0]), colorChannelByte(keypoint[1]), colorChannelByte(keypoint[2]), colorChannelByte(keypoint[3]))
	}
	return strings.Join(texts, " ")
}

func physicalPropertiesText(isCustom bool, values []float64) string {
	if !isCustom || len(values) < 5 {
		return "Default"
	}
	return fmt.Sprintf("Density: %s\nFriction: %s\nElasticity: %s", formatNumber(values[0]), formatNumber(values[1]), formatNumber(values[2]))
}

// flagNames writes the names of the bits that a Faces or Axes value sets.
func flagNames(flags byte, names []string) string {
	var setNames []string
	for bit, name := range names {
		if flags&(1<<bit) != 0 {
			setNames = append(setNames, name)
		}
	}
	if len(setNames) == 0 {
		return "None"
	}
	return strings.Join(setNames, ", ")
}

var (
	faceNames = []string{"Right", "Top", "Back", "Left", "Bottom", "Front"}
	axisNames = []string{"X", "Y", "Z"}
)

// fontText writes a Font as its family, weight, and style. The family is a
// content value such as rbxasset://fonts/families/GothamSSm.json.
func fontText(family string, weight uint16, style byte) string {
	familyName := family
	if slash := strings.LastIndexByte(familyName, '/'); slash >= 0 {
		familyName = familyName[slash+1:]
	}
	familyName = strings.TrimSuffix(familyName, ".json")
	styleName := "Normal"
	if style == 1 {
		styleName = "Italic"
	}
	return fmt.Sprintf("%s · %d · %s", familyName, weight, styleName)
}
