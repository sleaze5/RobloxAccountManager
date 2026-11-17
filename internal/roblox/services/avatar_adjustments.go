package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"slices"
)

// adjustmentTolerance absorbs the rounding of the decimal values that Roblox
// stores, such as 1.11 read back as 1.1100000000000001.
const adjustmentTolerance = 1e-6

// AvatarVector is a value on each axis of the accessory's attachment.
type AvatarVector struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

func (vector AvatarVector) axes() [3]float64 {
	return [3]float64{vector.X, vector.Y, vector.Z}
}

func (vector AvatarVector) equals(other AvatarVector) bool {
	for index, axis := range vector.axes() {
		if math.Abs(axis-other.axes()[index]) > adjustmentTolerance {
			return false
		}
	}
	return true
}

// AvatarAccessoryAdjustment moves a rigid accessory from its default
// attachment. Position is in studs, Rotation in degrees, and Scale multiplies
// the size.
type AvatarAccessoryAdjustment struct {
	Position AvatarVector `json:"position"`
	Rotation AvatarVector `json:"rotation"`
	Scale    AvatarVector `json:"scale"`
}

// defaultAccessoryAdjustment is the placement of an accessory that Roblox
// lists without one.
var defaultAccessoryAdjustment = AvatarAccessoryAdjustment{Scale: AvatarVector{X: 1, Y: 1, Z: 1}}

func (adjustment AvatarAccessoryAdjustment) equals(other AvatarAccessoryAdjustment) bool {
	return adjustment.Position.equals(other.Position) && adjustment.Rotation.equals(other.Rotation) && adjustment.Scale.equals(other.Scale)
}

func (adjustment AvatarAccessoryAdjustment) isFinite() bool {
	for _, vector := range []AvatarVector{adjustment.Position, adjustment.Rotation, adjustment.Scale} {
		for _, axis := range vector.axes() {
			if math.IsNaN(axis) || math.IsInf(axis, 0) {
				return false
			}
		}
	}
	return true
}

// AvatarClothingLayer is how Roblox stacks a layered clothing item. Puffiness
// is nil when Roblox lists none; the avatar editor no longer offers it.
type AvatarClothingLayer struct {
	Order     int      `json:"order"`
	Puffiness *float64 `json:"puffiness"`
}

// AvatarAxisLimits is the range of each axis of one adjustment.
type AvatarAxisLimits struct {
	Min AvatarVector `json:"min"`
	Max AvatarVector `json:"max"`
}

func (limits AvatarAxisLimits) contains(vector AvatarVector) bool {
	minimums, maximums := limits.Min.axes(), limits.Max.axes()
	for index, axis := range vector.axes() {
		if axis < minimums[index]-adjustmentTolerance || axis > maximums[index]+adjustmentTolerance {
			return false
		}
	}
	return true
}

func (limits AvatarAxisLimits) isValid() bool {
	minimums, maximums := limits.Min.axes(), limits.Max.axes()
	for index := range minimums {
		if math.IsNaN(minimums[index]) || math.IsNaN(maximums[index]) || minimums[index] > maximums[index] {
			return false
		}
	}
	return true
}

// AvatarAccessoryAdjustmentLimits is how far an account may adjust one
// accessory type.
type AvatarAccessoryAdjustmentLimits struct {
	Position AvatarAxisLimits `json:"position"`
	Rotation AvatarAxisLimits `json:"rotation"`
	Scale    AvatarAxisLimits `json:"scale"`
}

func (limits AvatarAccessoryAdjustmentLimits) contains(adjustment AvatarAccessoryAdjustment) bool {
	return limits.Position.contains(adjustment.Position) && limits.Rotation.contains(adjustment.Rotation) && limits.Scale.contains(adjustment.Scale)
}

func (limits AvatarAccessoryAdjustmentLimits) isValid() bool {
	return limits.Position.isValid() && limits.Rotation.isValid() && limits.Scale.isValid()
}

// metaVector is a vector as asset meta writes it, with uppercase axes.
// Verified against the API on 2026-11-17.
type metaVector struct {
	X float64 `json:"X"`
	Y float64 `json:"Y"`
	Z float64 `json:"Z"`
}

// listedAssetMeta is the part of asset meta that the application shows.
// Roblox lists order and puffiness for layered clothing, and position,
// rotation, and scale for an adjusted rigid accessory, each with version 1.
type listedAssetMeta struct {
	Order     *int        `json:"order"`
	Puffiness *float64    `json:"puffiness"`
	Position  *metaVector `json:"position"`
	Rotation  *metaVector `json:"rotation"`
	Scale     *metaVector `json:"scale"`
}

// hasMeta reports whether Roblox lists meta for the asset at all.
func hasMeta(rawMeta json.RawMessage) bool {
	trimmedMeta := bytes.TrimSpace(rawMeta)
	return len(trimmedMeta) > 0 && !bytes.Equal(trimmedMeta, []byte("null"))
}

// assetLayoutOf reads the adjustment and the clothing layer from the meta of
// an asset. Each is nil when the meta does not hold it, and both are nil for
// meta that is not an object.
func assetLayoutOf(rawMeta json.RawMessage) (*AvatarAccessoryAdjustment, *AvatarClothingLayer) {
	if !hasMeta(rawMeta) {
		return nil, nil
	}
	var meta listedAssetMeta
	if err := json.Unmarshal(rawMeta, &meta); err != nil {
		return nil, nil
	}
	var adjustment *AvatarAccessoryAdjustment
	if meta.Position != nil || meta.Rotation != nil || meta.Scale != nil {
		listedAdjustment := defaultAccessoryAdjustment
		if meta.Position != nil {
			listedAdjustment.Position = AvatarVector(*meta.Position)
		}
		if meta.Rotation != nil {
			listedAdjustment.Rotation = AvatarVector(*meta.Rotation)
		}
		if meta.Scale != nil {
			listedAdjustment.Scale = AvatarVector(*meta.Scale)
		}
		if listedAdjustment.isFinite() {
			adjustment = &listedAdjustment
		}
	}
	var layer *AvatarClothingLayer
	if meta.Order != nil {
		layer = &AvatarClothingLayer{Order: *meta.Order, Puffiness: meta.Puffiness}
		if meta.Puffiness != nil && (math.IsNaN(*meta.Puffiness) || math.IsInf(*meta.Puffiness, 0)) {
			layer.Puffiness = nil
		}
	}
	return adjustment, layer
}

// metaWithAdjustment writes the adjustment into the meta of an asset. It keeps
// every other field, such as the order and puffiness of layered clothing, and
// adds version 1, the format that Roblox lists, when the meta has no version.
func metaWithAdjustment(rawMeta json.RawMessage, adjustment AvatarAccessoryAdjustment) (json.RawMessage, error) {
	fields := map[string]json.RawMessage{}
	if hasMeta(rawMeta) {
		if err := json.Unmarshal(rawMeta, &fields); err != nil {
			return nil, fmt.Errorf("decode asset meta: %w", err)
		}
	}
	for name, vector := range map[string]AvatarVector{"position": adjustment.Position, "rotation": adjustment.Rotation, "scale": adjustment.Scale} {
		encodedVector, err := json.Marshal(metaVector(vector))
		if err != nil {
			return nil, fmt.Errorf("encode asset meta %s: %w", name, err)
		}
		fields[name] = encodedVector
	}
	if _, hasVersion := fields["version"]; !hasVersion {
		fields["version"] = json.RawMessage("1")
	}
	encodedMeta, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("encode asset meta: %w", err)
	}
	return encodedMeta, nil
}

// listedRefinementBounds is the range of one accessory type as the avatar
// rules list it. Verified against the API on 2026-11-17.
type listedRefinementBounds struct {
	Position *struct {
		X float64 `json:"xPosition"`
		Y float64 `json:"yPosition"`
		Z float64 `json:"zPosition"`
	} `json:"position"`
	Rotation *struct {
		X float64 `json:"xRotation"`
		Y float64 `json:"yRotation"`
		Z float64 `json:"zRotation"`
	} `json:"rotation"`
	Scale *struct {
		X float64 `json:"xScale"`
		Y float64 `json:"yScale"`
		Z float64 `json:"zScale"`
	} `json:"scale"`
}

func (listed listedRefinementBounds) isComplete() bool {
	return listed.Position != nil && listed.Rotation != nil && listed.Scale != nil
}

// accessoryAdjustmentLimitsOf lists the adjustment ranges by type name. The
// rules list the adjustable types by ID in accessoryRefinementTypes, and the
// bounds by the type name of wearableAssetTypes without spaces, such as
// BackAccessory for type 46. A type without complete, valid bounds is left
// out, because the account cannot adjust it.
func accessoryAdjustmentLimitsOf(payload avatarRulesPayload) map[string]AvatarAccessoryAdjustmentLimits {
	limitsByTypeName := make(map[string]AvatarAccessoryAdjustmentLimits, len(payload.AccessoryRefinementTypes))
	for _, wearableType := range payload.WearableAssetTypes {
		if !slices.Contains(payload.AccessoryRefinementTypes, wearableType.ID) {
			continue
		}
		typeName := ruleTypeNameReplacer.Replace(wearableType.Name)
		lower, upper := payload.AccessoryRefinementLowerBounds[typeName], payload.AccessoryRefinementUpperBounds[typeName]
		if !lower.isComplete() || !upper.isComplete() {
			continue
		}
		limits := AvatarAccessoryAdjustmentLimits{
			Position: AvatarAxisLimits{Min: AvatarVector(*lower.Position), Max: AvatarVector(*upper.Position)},
			Rotation: AvatarAxisLimits{Min: AvatarVector(*lower.Rotation), Max: AvatarVector(*upper.Rotation)},
			Scale:    AvatarAxisLimits{Min: AvatarVector(*lower.Scale), Max: AvatarVector(*upper.Scale)},
		}
		if limits.isValid() {
			limitsByTypeName[typeName] = limits
		}
	}
	return limitsByTypeName
}

// IsSameAdjustment reports whether two placements match, where a nil other
// placement stands for the default placement, as Roblox lists it.
func IsSameAdjustment(adjustment AvatarAccessoryAdjustment, other *AvatarAccessoryAdjustment) bool {
	if other == nil {
		return adjustment.equals(defaultAccessoryAdjustment)
	}
	return adjustment.equals(*other)
}
