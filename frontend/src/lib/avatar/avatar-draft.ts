import type {
	Avatar,
	AvatarAsset,
	AvatarBodyColors,
	AvatarChange,
	AvatarEmote,
	AvatarScales,
	AvatarType,
	InventoryCategory,
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

export function draftOf(avatar: Avatar): AvatarDraft {
	return {
		avatarType: avatar.avatarType,
		scales: { ...avatar.scales },
		bodyColors: { ...avatar.bodyColors },
		assets: [...(avatar.assets ?? [])],
		emotes: [...(avatar.emotes ?? [])],
	}
}

export function avatarChangeOf(draft: AvatarDraft): AvatarChange {
	return {
		avatarType: draft.avatarType,
		scales: draft.scales,
		bodyColors: draft.bodyColors,
		assetIds: draft.assets.map((asset) => asset.item.itemId),
		emotes: draft.emotes.map((emote) => ({
			assetId: emote.item.itemId,
			slot: emote.slot,
		})),
	}
}

// isDraftChanged compares what a save would write, so that wearing an item
// and taking it off again leaves the draft unchanged.
export function isDraftChanged(avatar: Avatar, draft: AvatarDraft): boolean {
	return (
		JSON.stringify(normalizedChange(avatarChangeOf(draftOf(avatar)))) !==
		JSON.stringify(normalizedChange(avatarChangeOf(draft)))
	)
}

function normalizedChange(change: AvatarChange): AvatarChange {
	return {
		...change,
		assetIds: [...(change.assetIds ?? [])].sort((first, second) => first - second),
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
	item: InventoryItem,
	category: InventoryCategory,
	maxWornCount: number,
): AvatarDraft {
	if (isWornInDraft(draft, item.itemId) || maxWornCount < 1) return draft
	const sameTypeAssets = draft.assets.filter(
			(asset) => asset.item.typeName === item.typeName,
		),
		replacedAssets = sameTypeAssets.slice(
			0,
			Math.max(sameTypeAssets.length - maxWornCount + 1, 0),
		)
	return {
		...draft,
		assets: [
			...draft.assets.filter((asset) => !replacedAssets.includes(asset)),
			{ category, item },
		],
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
