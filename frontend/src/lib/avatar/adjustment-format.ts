import type { AvatarAccessoryAdjustment, AvatarClothingLayer } from "../backend/bridge"
import { isDefaultAdjustmentKind, isEvenScale } from "./avatar-draft"
import type { AdjustmentKind } from "./avatar-draft"

// The values read as the editor's fields write them, with a sign only when
// negative.
const studs = new Intl.NumberFormat("en-US", {
		minimumFractionDigits: 2,
		maximumFractionDigits: 2,
	}),
	degrees = new Intl.NumberFormat("en-US", { maximumFractionDigits: 1 }),
	percent = new Intl.NumberFormat("en-US", {
		style: "percent",
		maximumFractionDigits: 0,
	}),
	list = new Intl.ListFormat("en-US", { style: "long", type: "conjunction" })

const formatterByKind: Record<AdjustmentKind, (value: number) => string> = {
	position: (value) => studs.format(value),
	rotation: (value) => `${degrees.format(value)}°`,
	scale: (value) => percent.format(value),
}

// formatAdjustmentValue writes a position in studs, a rotation in degrees,
// and a scale as a percentage of the default size.
export function formatAdjustmentValue(kind: AdjustmentKind, value: number): string {
	return formatterByKind[kind](value)
}

// describeAdjustment says what an adjustment changes, such as "Moved and
// turned", or "Resized to 120%" when only an even size changes.
export function describeAdjustment(
	adjustment: AvatarAccessoryAdjustment | null,
): string {
	if (!adjustment) return "Default placement"
	const isMoved = !isDefaultAdjustmentKind(adjustment, "position"),
		isTurned = !isDefaultAdjustmentKind(adjustment, "rotation"),
		isResized = !isDefaultAdjustmentKind(adjustment, "scale")
	if (!isMoved && !isTurned && !isResized) return "Default placement"
	if (!isMoved && !isTurned && isEvenScale(adjustment.scale)) {
		return `Resized to ${percent.format(adjustment.scale.x)}`
	}
	const changes = [
		isMoved && "moved",
		isTurned && "turned",
		isResized && "resized",
	].filter((change) => change !== false)
	const description = list.format(changes)
	return description.charAt(0).toUpperCase() + description.slice(1)
}

export function formatPuffiness(layer: AvatarClothingLayer): string {
	return layer.puffiness === null ? "" : percent.format(layer.puffiness)
}
