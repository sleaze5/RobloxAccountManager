import type { InventoryItem } from "../backend/bridge"
import type { AnimationChoice, AnimationChoiceGroup } from "./model-subject"

export const idleAnimationId = "idle"

// The prefixes of the animation IDs that the renderer plays by an asset:
// an emote, and an animation that a creator uploaded. assetAnimationPrefix
// names an asset of an avatar animation set, such as a Run animation, which
// the viewer plays by the first animation inside it.
export const emoteAnimationPrefix = "emote.",
	uploadedAnimationPrefix = "id.",
	assetAnimationPrefix = "asset."

// The animations of an avatar animation set that the viewer offers, by the
// name that the renderer gives them.
export const animationSetChoices: AnimationChoice[] = [
	{ id: idleAnimationId, label: "Idle", group: "animationSet" },
	{ id: "walk", label: "Walk", group: "animationSet" },
	{ id: "run", label: "Run", group: "animationSet" },
	{ id: "jump", label: "Jump", group: "animationSet" },
	{ id: "fall", label: "Fall", group: "animationSet" },
	{ id: "climb", label: "Climb", group: "animationSet" },
	{ id: "swim", label: "Swim", group: "animationSet" },
]

// The animation of the set that shows an animation asset, by its asset type.
// A mood plays on the face, which shows best while the avatar idles.
export const animationIdByAssetTypeName: Record<string, string> = {
	IdleAnimation: idleAnimationId,
	MoodAnimation: idleAnimationId,
	WalkAnimation: "walk",
	RunAnimation: "run",
	JumpAnimation: "jump",
	FallAnimation: "fall",
	ClimbAnimation: "climb",
	SwimAnimation: "swim",
}

const animationSetLabelById = new Map(
		animationSetChoices.map((choice) => [choice.id, choice.label]),
	),
	moodLabel = "Mood"

export function emoteChoiceOf(
	emote: Pick<InventoryItem, "itemId" | "name">,
	group: AnimationChoiceGroup,
): AnimationChoice {
	return {
		id: `${emoteAnimationPrefix}${emote.itemId}`,
		label: emote.name || `Emote ${emote.itemId}`,
		group,
		assetId: emote.itemId,
	}
}

// animationChoiceOfItem offers an emote or animation item, such as one that
// the user owns or pasted, or returns undefined for an item that the viewer
// cannot play. An item of an animation set names the animation that it
// replaces, such as Run.
export function animationChoiceOfItem(
	item: InventoryItem,
	group: AnimationChoiceGroup,
): AnimationChoice | undefined {
	const imageUrl = item.imageUrl || undefined
	if (item.typeName === "EmoteAnimation") {
		return { ...emoteChoiceOf(item, group), imageUrl }
	}
	if (item.typeName === "Animation") {
		return {
			id: `${uploadedAnimationPrefix}${item.itemId}`,
			label: item.name || `Animation ${item.itemId}`,
			detail: "Animation",
			group,
			assetId: item.itemId,
			imageUrl,
		}
	}
	const setAnimationId = animationIdByAssetTypeName[item.typeName]
	if (!setAnimationId) return undefined
	return {
		id: `${assetAnimationPrefix}${item.itemId}`,
		label: item.name || `Animation ${item.itemId}`,
		detail:
			item.typeName === "MoodAnimation"
				? moodLabel
				: animationSetLabelById.get(setAnimationId),
		group,
		assetId: item.itemId,
		imageUrl,
	}
}
