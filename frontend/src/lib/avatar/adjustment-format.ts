import type { AvatarClothingLayer } from "../backend/bridge"
import type { AdjustmentKind } from "./avatar-draft"

const studs = new Intl.NumberFormat("en-US", {
		minimumFractionDigits: 2,
		maximumFractionDigits: 2,
		signDisplay: "exceptZero",
	}),
	degrees = new Intl.NumberFormat("en-US", {
		maximumFractionDigits: 1,
		signDisplay: "exceptZero",
	}),
	percent = new Intl.NumberFormat("en-US", {
		style: "percent",
		maximumFractionDigits: 0,
	})

const formatterByKind: Record<AdjustmentKind, (value: number) => string> = {
		position: (value) => studs.format(value),
		rotation: (value) => `${degrees.format(value)}°`,
		scale: (value) => percent.format(value),
	},
	unitLabelByKind: Record<AdjustmentKind, string> = {
		position: "studs",
		rotation: "degrees",
		scale: "of the default size",
	}

// formatAdjustmentValue writes a position in studs, a rotation in degrees,
// and a scale as a percentage of the default size.
export function formatAdjustmentValue(kind: AdjustmentKind, value: number): string {
	return formatterByKind[kind](value)
}

export function adjustmentUnitLabel(kind: AdjustmentKind): string {
	return unitLabelByKind[kind]
}

export function formatPuffiness(layer: AvatarClothingLayer): string {
	return layer.puffiness === null ? "" : percent.format(layer.puffiness)
}
