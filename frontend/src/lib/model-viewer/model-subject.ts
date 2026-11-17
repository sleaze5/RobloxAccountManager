import type { Avatar, AvatarAsset, RenderedModel } from "../backend/bridge"

// AvatarLook is what the application builds an avatar from: a saved avatar,
// an outfit, or a draft of the avatar editor.
export type AvatarLook = Pick<Avatar, "avatarType" | "scales" | "bodyColors"> & {
	assets: AvatarAsset[] | null
}

// LiveModelSubject is what the application builds itself: an avatar wearing
// a look, or one accessory on its own.
export type LiveModelSubject =
	| { kind: "avatar"; look: AvatarLook }
	| { kind: "accessory"; assetId: number }

// RenderedModelSubject is a model that Roblox renders, for items that the
// application cannot build, such as models and gear. model is null while it
// loads, and error tells why it failed.
export interface RenderedModelSubject {
	kind: "rendered"
	model: RenderedModel | null
	error: string
	onRetry: () => void
}

// ModelSubject is what a 3D viewer shows.
export type ModelSubject = LiveModelSubject | RenderedModelSubject
