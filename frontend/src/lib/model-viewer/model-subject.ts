import type {
	Avatar,
	AvatarAsset,
	AvatarEmote,
	InventoryItem,
	RenderedModel,
} from "../backend/bridge"
import type { AdjustmentAxis, AdjustmentKind } from "../avatar/avatar-draft"

// AvatarLook is what the application builds an avatar from: a saved avatar,
// an outfit, or a draft of the avatar editor. Its emotes are the emotes that
// the avatar can play, and an outfit has none.
export type AvatarLook = Pick<Avatar, "avatarType" | "scales" | "bodyColors"> & {
	assets: AvatarAsset[] | null
	emotes?: AvatarEmote[] | null
}

// AnimationStart is what a preview plays: an emote, or the animation of the
// set that an animation asset replaces, by its asset type, such as
// RunAnimation.
export type AnimationStart =
	| { kind: "emote"; emote: Pick<InventoryItem, "itemId" | "name"> }
	| { kind: "animation"; assetTypeName: string }

// LiveModelSubject is what the application builds itself: an avatar wearing
// a look, or one accessory on its own. An avatar with startAnimation starts
// playing it, such as the emote that a preview shows.
export type LiveModelSubject =
	| { kind: "avatar"; look: AvatarLook; startAnimation?: AnimationStart }
	| { kind: "accessory"; assetId: number }

// AdjustmentGuide is the accessory of an avatar whose adjustment the user
// edits, with the kind of adjustment that its guides show. axis is the axis
// that the user points at or edits, which the guides highlight, or undefined
// for every axis.
export interface AdjustmentGuide {
	assetId: number
	kind: AdjustmentKind
	axis?: AdjustmentAxis
}

// AnimationChoice is an animation that an avatar can play: one of its
// animation set, such as Run, or one of its emotes.
export interface AnimationChoice {
	id: string
	label: string
	isEmote: boolean
}

// AnimationPlayback plays the animations of the avatar in a 3D viewer. Time
// is in seconds of the animation, and speed is a factor of its normal speed.
export interface AnimationPlayback {
	readonly animationChoices: AnimationChoice[]
	readonly animationId: string
	readonly isPlaying: boolean
	// Whether the chosen animation downloads.
	readonly isAnimationLoading: boolean
	readonly timeSeconds: number
	readonly lengthSeconds: number
	readonly speed: number
	chooseAnimation(animationId: string): void
	play(): void
	pause(): void
	seek(timeSeconds: number): void
	setSpeed(speed: number): void
}

// CameraPose is where the camera of a 3D viewer stands and looks, which the
// axes and the coordinates show. orientation is the rotation of the camera as
// a quaternion.
export interface CameraPose {
	position: [number, number, number]
	target: [number, number, number]
	orientation: [number, number, number, number]
}

// RenderedModelSubject is a model that Roblox renders, for items that the
// application cannot build, such as models and gear. model is null while it
// loads, and error tells why it failed.
interface RenderedModelSubject {
	kind: "rendered"
	model: RenderedModel | null
	error: string
	onRetry: () => void
}

// ModelSubject is what a 3D viewer shows.
export type ModelSubject = LiveModelSubject | RenderedModelSubject
