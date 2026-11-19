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
// playing it, such as the emote that a preview shows, and offers only it. An
// avatar with thumbnailAnimation, the emote or animation that poses its
// full-body image, starts playing it from the frame that the image shows, and
// offers every animation.
export type LiveModelSubject =
	| {
			kind: "avatar"
			look: AvatarLook
			startAnimation?: AnimationStart
			thumbnailAnimation?: InventoryItem
	  }
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

// ThumbnailFraming shows a live avatar as Roblox frames one of its profile
// images: the headshot or the full-body image, posed with the animation of
// poseAnimationId at the frame that the image shows. camera is the customized
// camera, or undefined for the standard framing of Roblox.
export interface ThumbnailFraming {
	image: "headshot" | "fullBody"
	poseAnimationId?: string
	camera?: {
		fieldOfViewDegrees: number
		yRotationDegrees: number
		distanceScale: number
	}
}

// AnimationChoiceGroup is the section of the animation picker that lists a
// choice: the animation set that the avatar wears, the emotes of its emote
// wheel and thumbnail, the emotes and animations that the user owns, and the
// ones that the user added by ID.
export type AnimationChoiceGroup =
	| "animationSet"
	| "emote"
	| "ownedEmote"
	| "ownedAnimation"
	| "added"

// AnimationChoice is an animation that an avatar can play: one of its
// animation set, such as Run, an emote, or an animation item, whose asset ID
// assetId holds. detail tells what the choice replaces or where it shows,
// such as Run or Thumbnail.
export interface AnimationChoice {
	id: string
	label: string
	group: AnimationChoiceGroup
	assetId?: number
	detail?: string
	imageUrl?: string
}

// AnimationPlayback plays the animations of the avatar in a 3D viewer. Time
// is in seconds of the animation, and speed is a factor of its normal speed.
// animationChoices lists what the avatar offers, and every other choice that
// the user played.
export interface AnimationPlayback {
	readonly animationChoices: AnimationChoice[]
	readonly animationId: string
	readonly isPlaying: boolean
	// Whether the chosen animation downloads.
	readonly isAnimationLoading: boolean
	// Whether the chosen animation could not download or play.
	readonly isAnimationFailed: boolean
	readonly timeSeconds: number
	readonly lengthSeconds: number
	readonly speed: number
	chooseAnimation(choice: AnimationChoice): void
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

// OwnedAnimations are the emotes or animations of one kind that the user
// owns. choices is undefined until they load.
export interface OwnedAnimations {
	readonly choices: AnimationChoice[] | undefined
	readonly isLoading: boolean
	readonly error: string
}

// AnimationLibrary offers animations beyond those of the avatar: what the
// user owns, and an emote or animation that the user names by ID or link.
// findAnimation resolves to undefined when it fails, and findError tells why.
export interface AnimationLibrary {
	readonly ownedEmotes: OwnedAnimations
	readonly ownedAnimations: OwnedAnimations
	readonly isFinding: boolean
	readonly findError: string
	loadOwned(): void
	findAnimation(reference: string): Promise<AnimationChoice | undefined>
	clearFindError(): void
}
