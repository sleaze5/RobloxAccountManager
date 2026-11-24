package modelfile

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

type Format string

const (
	FormatBinary Format = "rbxm"
	FormatXML    Format = "rbxmx"
)

// PropertyKind tells the explorer how to show the value of a property.
type PropertyKind string

const (
	// PropertyText is a short string or content value.
	PropertyText PropertyKind = "text"
	// PropertyDocument is a string that spans lines or is too long to show in
	// a row, such as the source of a script.
	PropertyDocument PropertyKind = "document"
	PropertyBoolean  PropertyKind = "boolean"
	PropertyNumber   PropertyKind = "number"
	// PropertyColor is a color, whose Color holds its CSS hex value.
	PropertyColor PropertyKind = "color"
	// PropertyEnum is an enum item, whose Text is its number until a class
	// index names it.
	PropertyEnum PropertyKind = "enum"
	// PropertyReference points to another instance of the file.
	PropertyReference PropertyKind = "reference"
	// PropertyAsset is a content value that names a Roblox asset.
	PropertyAsset PropertyKind = "asset"
	// PropertyBinary is data that only Roblox reads, such as physics data.
	PropertyBinary PropertyKind = "binary"
	// PropertyVector is a vector, or a value made of vectors such as a CFrame,
	// whose VectorParts hold its components.
	PropertyVector PropertyKind = "vector"
	// PropertyStructured is any other value, such as a UDim2. A value of
	// several parts writes each part on its own line.
	PropertyStructured PropertyKind = "structured"
)

// Property is one property of an instance, with its value written out for
// the explorer.
type Property struct {
	Name string       `json:"name"`
	Kind PropertyKind `json:"kind"`
	Text string       `json:"text"`
	// Group is the category of the property in Roblox Studio, such as
	// Appearance, set by a class index. Attributes are in Attributes.
	Group string `json:"group"`
	// Color is the CSS hex value of a color, such as #a3a2a5.
	Color string `json:"color"`
	// AssetID is the asset that a content value names.
	AssetID int64 `json:"assetId"`
	// ReferencedInstanceID is the instance that a reference points to, or 0
	// when it points to none.
	ReferencedInstanceID int `json:"referencedInstanceId"`
	// VectorParts holds the components of a vector, or of each vector of a
	// value made of several, such as the position and the orientation of a
	// CFrame.
	VectorParts []VectorPart `json:"vectorParts"`
	EnumValue   uint32       `json:"-"`
	referent    string
	// isRenamed marks a property that the file stores under an older name.
	isRenamed bool
}

// VectorPart is one vector of a property value: its X, Y, and optional Z
// components, written out. Label names the vector within a value made of
// several, such as Position, and is empty for a plain vector.
type VectorPart struct {
	Label      string   `json:"label"`
	Components []string `json:"components"`
}

const AttributesGroup = "Attributes"

// Instance is one instance of a model file. IDs are unique within one
// decoded model and count from the first instance ID that Decode takes.
type Instance struct {
	ID         int        `json:"id"`
	ClassName  string     `json:"className"`
	Name       string     `json:"name"`
	Tags       []string   `json:"tags"`
	Properties []Property `json:"properties"`
	Children   []Instance `json:"children"`
}

type Model struct {
	Format    Format     `json:"format"`
	Instances []Instance `json:"instances"`
	// InstanceCount counts every instance of the file, including those left
	// out beyond the limit that Decode takes.
	InstanceCount int  `json:"instanceCount"`
	IsTruncated   bool `json:"isTruncated"`
}

// decodedInstance is an instance while a file is decoded, before its
// references point to instance IDs.
type decodedInstance struct {
	className  string
	referent   string
	properties []Property
	tags       []string
	children   []*decodedInstance
}

// Decode reads the instance tree of a model file. It keeps at most
// maxInstanceCount instances, in tree order, and numbers them from
// firstInstanceID.
func Decode(file []byte, firstInstanceID, maxInstanceCount int) (Model, error) {
	var (
		roots  []*decodedInstance
		format Format
		err    error
	)
	switch {
	case IsBinary(file):
		format = FormatBinary
		roots, err = decodeBinary(file)
	case IsXML(file):
		format = FormatXML
		roots, err = decodeXML(file)
	default:
		return Model{}, errInvalidFile
	}
	if err != nil {
		return Model{}, fmt.Errorf("decode %s file: %w", format, err)
	}
	builder := treeBuilder{
		nextID: firstInstanceID, maxInstanceCount: maxInstanceCount,
		idByReferent: map[string]int{}, nameByReferent: map[string]string{},
	}
	instances := builder.build(roots)
	builder.resolveReferences(instances)
	return Model{Format: format, Instances: instances, InstanceCount: builder.instanceCount, IsTruncated: builder.instanceCount > builder.keptCount}, nil
}

type treeBuilder struct {
	nextID           int
	maxInstanceCount int
	instanceCount    int
	keptCount        int
	idByReferent     map[string]int
	nameByReferent   map[string]string
}

// build numbers the instances in tree order and sorts siblings by name, as
// Roblox Studio lists them.
func (builder *treeBuilder) build(decodedInstances []*decodedInstance) []Instance {
	slices.SortStableFunc(decodedInstances, func(first, second *decodedInstance) int {
		return cmp.Or(
			cmp.Compare(strings.ToLower(nameOf(first)), strings.ToLower(nameOf(second))),
			cmp.Compare(first.className, second.className),
		)
	})
	instances := make([]Instance, 0, len(decodedInstances))
	for _, decoded := range decodedInstances {
		builder.instanceCount++
		if builder.keptCount >= builder.maxInstanceCount {
			builder.countDescendants(decoded.children)
			continue
		}
		builder.keptCount++
		instance := Instance{ID: builder.nextID, ClassName: decoded.className, Name: nameOf(decoded), Tags: decoded.tags, Properties: decoded.properties}
		builder.nextID++
		if decoded.referent != "" {
			builder.idByReferent[decoded.referent] = instance.ID
			builder.nameByReferent[decoded.referent] = instance.Name
		}
		instance.Children = builder.build(decoded.children)
		instances = append(instances, instance)
	}
	return instances
}

func (builder *treeBuilder) countDescendants(decodedInstances []*decodedInstance) {
	for _, decoded := range decodedInstances {
		builder.instanceCount++
		builder.countDescendants(decoded.children)
	}
}

// resolveReferences points each reference to the ID of its instance and
// names it. A reference to an instance outside the file, or one left out,
// names nothing.
func (builder *treeBuilder) resolveReferences(instances []Instance) {
	for instanceIndex := range instances {
		instance := &instances[instanceIndex]
		for propertyIndex := range instance.Properties {
			property := &instance.Properties[propertyIndex]
			if property.Kind != PropertyReference {
				continue
			}
			property.ReferencedInstanceID = builder.idByReferent[property.referent]
			property.Text = builder.nameByReferent[property.referent]
		}
		builder.resolveReferences(instance.Children)
	}
}

func nameOf(decoded *decodedInstance) string {
	for _, property := range decoded.properties {
		if property.Name == "Name" {
			return property.Text
		}
	}
	return decoded.className
}

// serializedPropertyNames maps the names under which files store some
// properties to the names that Roblox Studio shows.
var serializedPropertyNames = map[string]string{
	"Color3uint8": "Color", "formFactorRaw": "FormFactor", "Health_XML": "Health",
	"xmlRead_MaxDistance_3": "MaxDistance", "shape": "Shape", "size": "Size",
	"formFactor": "FormFactor", "archivable": "Archivable", "style": "Style",
	"MeshID": "MeshId", "MaterialVariantSerialized": "MaterialVariant",
	"DefinesCapabilities": "Sandboxed",
}

// addProperty adds a decoded property under the name that Roblox Studio
// shows. Attributes and tags become their own lists, and a stored name that
// another stored property already shows under its Studio name is left out.
func (decoded *decodedInstance) addProperty(property Property, rawValue []byte) {
	switch property.Name {
	case "AttributesSerialize":
		decoded.properties = append(decoded.properties, decodeAttributes(rawValue)...)
		return
	case "Tags":
		for tag := range strings.SplitSeq(string(rawValue), "\x00") {
			if tag != "" {
				decoded.tags = append(decoded.tags, tag)
			}
		}
		return
	}
	if shownName, isRenamed := serializedPropertyNames[property.Name]; isRenamed {
		if slices.ContainsFunc(decoded.properties, func(existing Property) bool { return existing.Name == shownName }) {
			return
		}
		property.Name = shownName
		property.isRenamed = true
	} else {
		decoded.properties = slices.DeleteFunc(decoded.properties, func(existing Property) bool {
			return existing.isRenamed && existing.Name == property.Name
		})
	}
	decoded.properties = append(decoded.properties, property)
}
