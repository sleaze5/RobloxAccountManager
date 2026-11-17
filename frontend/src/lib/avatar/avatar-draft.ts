import type {
	Avatar,
	AvatarAccessoryAdjustment,
	AvatarAccessoryAdjustmentLimits,
	AvatarAsset,
	AvatarBodyColors,
	AvatarChange,
	AvatarEmote,
	AvatarScales,
	AvatarType,
	AvatarVector,
	InventoryItem,
} from "../backend/bridge"

// AvatarDraft is the avatar that the editor changes until the user saves it.
export interface AvatarDraft {
	avatarType: AvatarType
	scales: AvatarScales
	bodyColors: AvatarBodyColors
	assets: AvatarAsset[]
	emotes: AvatarEmote[]
}

export type BodyPart = keyof AvatarBodyColors

export const bodyParts: { part: BodyPart; label: string }[] = [
	{ part: "head", label: "Head" },
	{ part: "torso", label: "Torso" },
	{ part: "leftArm", label: "Left arm" },
	{ part: "rightArm", label: "Right arm" },
	{ part: "leftLeg", label: "Left leg" },
	{ part: "rightLeg", label: "Right leg" },
]

// bodyScales lists the scales of an R15 avatar. Roblox lets an account change
// the body shape scales only when AvatarRules.isBodyShapeEnabled is true.
export const bodyScales: {
	scale: keyof AvatarScales
	label: string
	isBodyShape: boolean
}[] = [
	{ scale: "height", label: "Height", isBodyShape: false },
	{ scale: "width", label: "Width", isBodyShape: false },
	{ scale: "head", label: "Head", isBodyShape: false },
	{ scale: "proportion", label: "Proportions", isBodyShape: true },
	{ scale: "bodyType", label: "Body type", isBodyShape: true },
]

export const emoteSlotCount = 8

export type AdjustmentKind = keyof AvatarAccessoryAdjustment
export type AdjustmentAxis = keyof AvatarVector

export const adjustmentAxes: AdjustmentAxis[] = ["x", "y", "z"]

// adjustmentKinds lists what an accessory adjustment changes, with each axis
// named by the way it moves the accessory on the avatar: Roblox points X to
// the avatar's side, Y up, and Z to its back. A value field shows a scale as a
// percentage, so its factor is 100. The steps match the precision of the
// values that the Roblox avatar editor stores.
export const adjustmentKinds: {
	kind: AdjustmentKind
	label: string
	unitLabel: string
	axisLabels: Record<AdjustmentAxis, string>
	step: number
	fieldFactor: number
}[] = [
	{
		kind: "position",
		label: "Position",
		unitLabel: "studs",
		axisLabels: { x: "Side", y: "Height", z: "Depth" },
		step: 0.01,
		fieldFactor: 1,
	},
	{
		kind: "rotation",
		label: "Rotation",
		unitLabel: "degrees",
		axisLabels: { x: "Tilt", y: "Turn", z: "Roll" },
		step: 0.1,
		fieldFactor: 1,
	},
	{
		kind: "scale",
		label: "Scale",
		unitLabel: "percent",
		axisLabels: { x: "Width", y: "Height", z: "Depth" },
		step: 0.01,
		fieldFactor: 100,
	},
]

// defaultAdjustment is the placement of an accessory that Roblox lists
// without an adjustment.
export const defaultAdjustment: AvatarAccessoryAdjustment = {
	position: { x: 0, y: 0, z: 0 },
	rotation: { x: 0, y: 0, z: 0 },
	scale: { x: 1, y: 1, z: 1 },
}

export function adjustmentOf(asset: AvatarAsset): AvatarAccessoryAdjustment {
	return asset.adjustment ?? defaultAdjustment
}

export function isDefaultAdjustmentKind(
	adjustment: AvatarAccessoryAdjustment | null,
	kind: AdjustmentKind,
): boolean {
	return (
		!adjustment ||
		adjustmentAxes.every(
			(axis) => adjustment[kind][axis] === defaultAdjustment[kind][axis],
		)
	)
}

export function isDefaultAdjustment(
	adjustment: AvatarAccessoryAdjustment | null,
): boolean {
	return adjustmentKinds.every(({ kind }) =>
		isDefaultAdjustmentKind(adjustment, kind),
	)
}

export function isSameAdjustment(
	first: AvatarAccessoryAdjustment | null,
	second: AvatarAccessoryAdjustment | null,
): boolean {
	const firstAdjustment = first ?? defaultAdjustment,
		secondAdjustment = second ?? defaultAdjustment
	return adjustmentKinds.every(({ kind }) =>
		adjustmentAxes.every(
			(axis) => firstAdjustment[kind][axis] === secondAdjustment[kind][axis],
		),
	)
}

// mirroredAdjustment reflects the placement across the middle of the avatar,
// so that the placement of a left accessory fits its right counterpart: the
// side moves to the other side, and the turn and roll reverse.
export function mirroredAdjustment(
	adjustment: AvatarAccessoryAdjustment,
): AvatarAccessoryAdjustment {
	return {
		position: { ...adjustment.position, x: -adjustment.position.x },
		rotation: {
			x: adjustment.rotation.x,
			y: -adjustment.rotation.y,
			z: -adjustment.rotation.z,
		},
		scale: { ...adjustment.scale },
	}
}

// clampedAdjustment keeps each axis within the limits, such as when a pasted
// placement comes from an accessory type with a wider range.
export function clampedAdjustment(
	adjustment: AvatarAccessoryAdjustment,
	limits: AvatarAccessoryAdjustmentLimits,
): AvatarAccessoryAdjustment {
	const clamped = structuredClone(adjustment)
	for (const { kind } of adjustmentKinds) {
		for (const axis of adjustmentAxes) {
			clamped[kind][axis] = Math.min(
				Math.max(clamped[kind][axis], limits[kind].min[axis]),
				limits[kind].max[axis],
			)
		}
	}
	return clamped
}

export function draftOf(avatar: Avatar): AvatarDraft {
	return {
		avatarType: avatar.avatarType,
		scales: { ...avatar.scales },
		bodyColors: { ...avatar.bodyColors },
		assets: [...(avatar.assets ?? [])],
		emotes: [...(avatar.emotes ?? [])],
	}
}

// avatarChangeOf writes the draft as a change of the saved avatar. A change
// carries the placement of an accessory only when the draft edited it, so
// that a save keeps a placement that Roblox changed since the avatar loaded.
export function avatarChangeOf(draft: AvatarDraft, savedAvatar: Avatar): AvatarChange {
	const savedAssetById = new Map(
		(savedAvatar.assets ?? []).map((asset) => [asset.item.itemId, asset]),
	)
	return {
		avatarType: draft.avatarType,
		scales: draft.scales,
		bodyColors: draft.bodyColors,
		assets: draft.assets.map((asset) => {
			const savedAsset = savedAssetById.get(asset.item.itemId),
				isEdited =
					!!savedAsset &&
					!isSameAdjustment(savedAsset.adjustment, asset.adjustment)
			return {
				assetId: asset.item.itemId,
				adjustment: isEdited ? asset.adjustment : null,
			}
		}),
		emotes: draft.emotes.map((emote) => ({
			assetId: emote.item.itemId,
			slot: emote.slot,
		})),
	}
}

// isDraftChanged compares what a save would write, so that wearing an item
// and taking it off again, or moving an accessory and moving it back, leaves
// the draft unchanged.
export function isDraftChanged(avatar: Avatar, draft: AvatarDraft): boolean {
	return (
		JSON.stringify(normalizedChange(avatarChangeOf(draftOf(avatar), avatar))) !==
		JSON.stringify(normalizedChange(avatarChangeOf(draft, avatar)))
	)
}

// normalizedChange orders the assets and emotes. It keeps a placement reset to
// the default, because avatarChangeOf writes a placement only when the draft
// edited it.
function normalizedChange(change: AvatarChange): AvatarChange {
	return {
		...change,
		assets: [...(change.assets ?? [])].sort(
			(first, second) => first.assetId - second.assetId,
		),
		emotes: [...(change.emotes ?? [])].sort(
			(first, second) => first.slot - second.slot,
		),
	}
}

export function isWornInDraft(draft: AvatarDraft, assetId: number): boolean {
	return draft.assets.some((asset) => asset.item.itemId === assetId)
}

// wearItem adds the item to the draft. When the avatar already wears as many
// items of the type as Roblox allows, the item replaces the one of that type
// that the avatar wore first, as the Roblox avatar editor does.
export function wearItem(
	draft: AvatarDraft,
	wornAsset: AvatarAsset,
	maxWornCount: number,
): AvatarDraft {
	if (isWornInDraft(draft, wornAsset.item.itemId) || maxWornCount < 1) return draft
	const sameTypeAssets = draft.assets.filter(
			(asset) => asset.item.typeName === wornAsset.item.typeName,
		),
		replacedAssets = sameTypeAssets.slice(
			0,
			Math.max(sameTypeAssets.length - maxWornCount + 1, 0),
		)
	return {
		...draft,
		assets: [
			...draft.assets.filter((asset) => !replacedAssets.includes(asset)),
			wornAsset,
		],
	}
}

// adjustAsset returns the same draft when the placement does not change, so
// that applying a value again adds no step to the history.
export function adjustAsset(
	draft: AvatarDraft,
	assetId: number,
	adjustment: AvatarAccessoryAdjustment,
): AvatarDraft {
	const adjustedAsset = draft.assets.find((asset) => asset.item.itemId === assetId)
	if (!adjustedAsset || isSameAdjustment(adjustedAsset.adjustment, adjustment)) {
		return draft
	}
	return {
		...draft,
		assets: draft.assets.map((asset) =>
			asset.item.itemId === assetId ? { ...asset, adjustment } : asset,
		),
	}
}

export function takeOffItem(draft: AvatarDraft, assetId: number): AvatarDraft {
	return {
		...draft,
		assets: draft.assets.filter((asset) => asset.item.itemId !== assetId),
	}
}

// putEmoteInSlot moves an emote that another slot holds, because Roblox keeps
// each emote in one slot.
export function putEmoteInSlot(
	draft: AvatarDraft,
	slot: number,
	item: InventoryItem,
): AvatarDraft {
	return {
		...draft,
		emotes: [
			...draft.emotes.filter(
				(emote) => emote.slot !== slot && emote.item.itemId !== item.itemId,
			),
			{ slot, item },
		],
	}
}

export function clearEmoteSlot(draft: AvatarDraft, slot: number): AvatarDraft {
	return { ...draft, emotes: draft.emotes.filter((emote) => emote.slot !== slot) }
}
