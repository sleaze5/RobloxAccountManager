package modelfile

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// xmlNode is one element of an XML model file with its attributes, text, and
// child elements.
type xmlNode struct {
	XMLName  xml.Name   `xml:""`
	Attrs    []xml.Attr `xml:",any,attr"`
	Text     string     `xml:",chardata"`
	Children []xmlNode  `xml:",any"`
}

func (node xmlNode) attribute(name string) string {
	for _, attribute := range node.Attrs {
		if attribute.Name.Local == name {
			return attribute.Value
		}
	}
	return ""
}

func (node xmlNode) trimmedText() string {
	return strings.TrimSpace(node.Text)
}

func (node xmlNode) child(name string) (xmlNode, bool) {
	for _, child := range node.Children {
		if strings.EqualFold(child.XMLName.Local, name) {
			return child, true
		}
	}
	return xmlNode{}, false
}

func (node xmlNode) childText(name string) string {
	child, _ := node.child(name)
	return child.trimmedText()
}

func (node xmlNode) childNumber(name string) float64 {
	number, _ := strconv.ParseFloat(node.childText(name), 64)
	return number
}

func (node xmlNode) childInteger(name string) int64 {
	integer, _ := strconv.ParseInt(node.childText(name), 10, 64)
	return integer
}

// decodeXML reads the XML format, where each Item element is an instance with
// a Properties element and its child Items.
func decodeXML(file []byte) ([]*decodedInstance, error) {
	var document xmlNode
	if err := xml.NewDecoder(bytes.NewReader(file)).Decode(&document); err != nil {
		return nil, fmt.Errorf("parse XML: %w", err)
	}
	sharedStringLengths := map[string]int{}
	if sharedStrings, hasSharedStrings := document.child("SharedStrings"); hasSharedStrings {
		for _, sharedString := range sharedStrings.Children {
			sharedStringLengths[sharedString.attribute("md5")] = base64.StdEncoding.DecodedLen(len(strings.TrimSpace(sharedString.Text)))
		}
	}
	return decodeXMLItems(document.Children, sharedStringLengths), nil
}

func decodeXMLItems(nodes []xmlNode, sharedStringLengths map[string]int) []*decodedInstance {
	var instances []*decodedInstance
	for _, node := range nodes {
		if node.XMLName.Local != "Item" {
			continue
		}
		instance := &decodedInstance{className: node.attribute("class"), referent: node.attribute("referent")}
		if properties, hasProperties := node.child("Properties"); hasProperties {
			for _, propertyNode := range properties.Children {
				property, rawValue := decodeXMLProperty(propertyNode, sharedStringLengths)
				instance.addProperty(property, rawValue)
			}
		}
		instance.children = decodeXMLItems(node.Children, sharedStringLengths)
		instances = append(instances, instance)
	}
	return instances
}

// decodeXMLProperty reads one property, whose element names its type. It also
// returns the raw bytes of string values, which hold attributes and tags.
func decodeXMLProperty(node xmlNode, sharedStringLengths map[string]int) (Property, []byte) {
	name := node.attribute("name")
	switch strings.ToLower(node.XMLName.Local) {
	case "string", "protectedstring":
		return stringProperty(name, []byte(node.Text)), []byte(node.Text)
	case "binarystring":
		decoded, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(node.Text), ""))
		if err != nil {
			return binaryProperty(name, 0), nil
		}
		return binaryProperty(name, len(decoded)), decoded
	case "sharedstring":
		return binaryProperty(name, sharedStringLengths[strings.TrimSpace(node.Text)]), nil
	}
	if readValue, isKnown := xmlValueReaders[strings.ToLower(node.XMLName.Local)]; isKnown {
		return readValue(node, name), nil
	}
	return structuredProperty(name, strings.TrimSpace(node.Text)), nil
}

// xmlValueReaders read the value of each property type, by the lowercase
// name of its element.
var xmlValueReaders = map[string]func(node xmlNode, name string) Property{
	"content": func(node xmlNode, name string) Property {
		for _, uriName := range []string{"url", "uri"} {
			if uri, hasURI := node.child(uriName); hasURI {
				return contentProperty(name, strings.TrimSpace(uri.Text))
			}
		}
		return contentProperty(name, "")
	},
	"bool": func(node xmlNode, name string) Property {
		return booleanProperty(name, node.trimmedText() == "true")
	},
	"int":   readXMLInteger,
	"int64": readXMLInteger,
	"brickcolor": func(node xmlNode, name string) Property {
		number, _ := strconv.ParseInt(node.trimmedText(), 10, 64)
		return brickColorProperty(name, number)
	},
	"securitycapabilities": readXMLInteger,
	"float":                readXMLNumber,
	"double":               readXMLNumber,
	"token": func(node xmlNode, name string) Property {
		value, _ := strconv.ParseUint(node.trimmedText(), 10, 32)
		return enumProperty(name, uint32(value))
	},
	"vector3":      readXMLVector3,
	"vector3int16": readXMLVector3,
	"vector2": func(node xmlNode, name string) Property {
		return vectorProperty(name, node.childNumber("X"), node.childNumber("Y"))
	},
	"coordinateframe": xmlCFrameProperty,
	"optionalcoordinateframe": func(node xmlNode, name string) Property {
		if cframe, hasCFrame := node.child("CFrame"); hasCFrame {
			return xmlCFrameProperty(cframe, name)
		}
		return structuredProperty(name, "None")
	},
	"color3": func(node xmlNode, name string) Property {
		if _, hasChannels := node.child("R"); hasChannels {
			return colorProperty(name, node.childNumber("R"), node.childNumber("G"), node.childNumber("B"))
		}
		return packedColorProperty(name, node.trimmedText())
	},
	"color3uint8": func(node xmlNode, name string) Property {
		return packedColorProperty(name, node.trimmedText())
	},
	"udim": func(node xmlNode, name string) Property {
		return structuredProperty(name, "{"+formatNumber(node.childNumber("S"))+", "+strconv.FormatInt(node.childInteger("O"), 10)+"}")
	},
	"udim2": func(node xmlNode, name string) Property {
		return structuredProperty(name, udim2Text(node.childNumber("XS"), node.childInteger("XO"), node.childNumber("YS"), node.childInteger("YO")))
	},
	"ref": func(node xmlNode, name string) Property {
		referent := node.trimmedText()
		if referent == "null" {
			referent = ""
		}
		return referenceProperty(name, referent)
	},
	"numbersequence": func(node xmlNode, name string) Property {
		values := parseNumbers(node.trimmedText())
		keypoints := make([][3]float64, 0, len(values)/3)
		for index := 0; index+2 < len(values); index += 3 {
			keypoints = append(keypoints, [3]float64{values[index], values[index+1], values[index+2]})
		}
		return structuredProperty(name, numberSequenceText(keypoints))
	},
	"colorsequence": func(node xmlNode, name string) Property {
		values := parseNumbers(node.trimmedText())
		keypoints := make([][4]float64, 0, len(values)/5)
		for index := 0; index+4 < len(values); index += 5 {
			keypoints = append(keypoints, [4]float64{values[index], values[index+1], values[index+2], values[index+3]})
		}
		return structuredProperty(name, colorSequenceText(keypoints))
	},
	"numberrange": func(node xmlNode, name string) Property {
		return structuredProperty(name, formatNumbers(parseNumbers(node.trimmedText())...))
	},
	"rect2d": func(node xmlNode, name string) Property {
		minimum, _ := node.child("min")
		maximum, _ := node.child("max")
		return structuredProperty(name, "{"+formatNumbers(minimum.childNumber("X"), minimum.childNumber("Y"))+"}, {"+formatNumbers(maximum.childNumber("X"), maximum.childNumber("Y"))+"}")
	},
	"physicalproperties": func(node xmlNode, name string) Property {
		values := []float64{node.childNumber("Density"), node.childNumber("Friction"), node.childNumber("Elasticity"), node.childNumber("FrictionWeight"), node.childNumber("ElasticityWeight")}
		return structuredProperty(name, physicalPropertiesText(node.childText("CustomPhysics") == "true", values))
	},
	"faces": func(node xmlNode, name string) Property {
		flags, _ := strconv.ParseUint(node.childText("faces"), 10, 8)
		return structuredProperty(name, flagNames(byte(flags), faceNames))
	},
	"axes": func(node xmlNode, name string) Property {
		flags, _ := strconv.ParseUint(node.childText("axes"), 10, 8)
		return structuredProperty(name, flagNames(byte(flags), axisNames))
	},
	"font": func(node xmlNode, name string) Property {
		family, _ := node.child("Family")
		weight, _ := strconv.ParseUint(node.childText("Weight"), 10, 16)
		style := byte(0)
		if node.childText("Style") == "Italic" {
			style = 1
		}
		return structuredProperty(name, fontText(family.childText("url"), uint16(weight), style))
	},
	"ray": func(node xmlNode, name string) Property {
		origin, _ := node.child("origin")
		direction, _ := node.child("direction")
		return rayProperty(name, []float64{origin.childNumber("X"), origin.childNumber("Y"), origin.childNumber("Z")}, []float64{direction.childNumber("X"), direction.childNumber("Y"), direction.childNumber("Z")})
	},
}

func readXMLInteger(node xmlNode, name string) Property {
	integer, _ := strconv.ParseInt(node.trimmedText(), 10, 64)
	return integerProperty(name, integer)
}

func readXMLNumber(node xmlNode, name string) Property {
	number, _ := strconv.ParseFloat(node.trimmedText(), 64)
	return numberProperty(name, number)
}

func readXMLVector3(node xmlNode, name string) Property {
	return vectorProperty(name, node.childNumber("X"), node.childNumber("Y"), node.childNumber("Z"))
}

func xmlCFrameProperty(node xmlNode, name string) Property {
	var rotation [9]float64
	for index, componentName := range []string{"R00", "R01", "R02", "R10", "R11", "R12", "R20", "R21", "R22"} {
		rotation[index] = node.childNumber(componentName)
	}
	return cframeProperty(name, [3]float64{node.childNumber("X"), node.childNumber("Y"), node.childNumber("Z")}, rotation)
}

// packedColorProperty reads a color packed into one number as 0xAARRGGBB.
func packedColorProperty(name, text string) Property {
	packed, _ := strconv.ParseUint(text, 10, 32)
	return color8Property(name, uint8(packed>>16), uint8(packed>>8), uint8(packed))
}

func parseNumbers(text string) []float64 {
	fields := strings.Fields(text)
	numbers := make([]float64, 0, len(fields))
	for _, field := range fields {
		if number, err := strconv.ParseFloat(field, 64); err == nil {
			numbers = append(numbers, number)
		}
	}
	return numbers
}
