import dracoDecoderUrl from "../../../node_modules/roavatar-renderer/dist/thirdparty/draco/javascript/draco_decoder.js?url"
import {
	API,
	Authentication,
	FLAGS,
	Outfit,
	OutfitRenderer,
	RBXRenderer,
} from "roavatar-renderer"
import { AvatarType } from "../backend/bridge"
import { moveOrbitViewByKey } from "../shared/orbit-view-keys"
import type { AvatarDraft } from "./avatar-draft"
import {
	clearLiveAvatarFiles,
	fetchLiveAvatarFile,
	liveAvatarImageUrl,
	setLiveAvatarAccount,
} from "./live-avatar-files"

// liveAvatar is what the 3D avatar shows. The renderer is one scene on one
// canvas for the whole application, so every view of the 3D avatar shares it.
export const liveAvatar = $state({
	// Whether the renderer downloads or builds a part of the avatar.
	isLoading: false,
	// Whether the avatar has been built once since the 3D avatar opened.
	hasShownAvatar: false,
	// Whether the last avatar could not be built, such as when the rig failed
	// to download.
	isFailed: false,
})

// LiveAvatarHost is a view that shows the canvas. canvasLabel names the canvas
// for a screen reader, and descriptionId is the element that tells how to move
// the view.
interface LiveAvatarHost {
	host: HTMLElement
	canvasLabel: string
	descriptionId: string
}

// The views that show the canvas, newest last. The newest view shows it, and
// the canvas returns to the previous view when the newest one closes, such as
// the avatar page after the expanded view.
const hosts: LiveAvatarHost[] = []
let rendererSetup: Promise<boolean> | undefined,
	outfitRenderer: OutfitRenderer | undefined,
	shownDraft: AvatarDraft | undefined,
	hostResizeObserver: ResizeObserver | undefined,
	isFramePending = false

// startLiveAvatarRenderer sets up the renderer once, and reports whether it
// can draw, which needs WebGL 2. The setup downloads the meshes that compose
// clothing textures, so the account that downloads them is chosen first.
export function startLiveAvatarRenderer(accountId: number): Promise<boolean> {
	setLiveAvatarAccount(accountId)
	rendererSetup ??= setUpRenderer()
	return rendererSetup
}

// attachLiveAvatar shows the canvas in the host until the returned function
// detaches it. When the last host detaches, the avatar is removed from the
// scene, so that its meshes and textures are freed.
export function attachLiveAvatar(attachedHost: LiveAvatarHost): () => void {
	hosts.push(attachedHost)
	showCanvasIn(attachedHost)
	return () => {
		const hostIndex = hosts.lastIndexOf(attachedHost)
		if (hostIndex >= 0) hosts.splice(hostIndex, 1)
		const previousHost = hosts.at(-1)
		if (previousHost) {
			showCanvasIn(previousHost)
			return
		}
		hostResizeObserver?.disconnect()
		RBXRenderer.getRendererElement().remove()
		outfitRenderer?.destroy()
		outfitRenderer = shownDraft = undefined
		clearLiveAvatarFiles()
		Object.assign(liveAvatar, {
			isLoading: false,
			hasShownAvatar: false,
			isFailed: false,
		})
	}
}

// showLiveAvatarDraft builds the draft on the 3D avatar. Only the changed
// parts are built again, so a moved accessory follows its slider.
export function showLiveAvatarDraft(accountId: number, draft: AvatarDraft): void {
	if (draft === shownDraft) return
	shownDraft = draft
	setLiveAvatarAccount(accountId)
	liveAvatar.isFailed = false
	const outfit = outfitOf(draft)
	if (outfitRenderer) {
		outfitRenderer.setOutfit(outfit)
		return
	}
	const createdRenderer = new OutfitRenderer(new Authentication(), outfit)
	createdRenderer.onSuccess.Connect(() => {
		liveAvatar.hasShownAvatar = true
		void poseAvatar(createdRenderer)
		requestLiveAvatarFrame()
	})
	createdRenderer.onError.Connect(() => {
		liveAvatar.isFailed = true
	})
	createdRenderer.onRenderSuccess.Connect(requestLiveAvatarFrame)
	outfitRenderer = createdRenderer
}

// poseAvatar puts the avatar in the first frame of its idle animation, which
// is the pose of the Roblox avatar editor. Without it, the rig stands in its
// bind pose. The animation does not run, so the canvas draws only on changes.
async function poseAvatar(posedRenderer: OutfitRenderer): Promise<void> {
	await posedRenderer.setMainAnimation("idle:0")
	posedRenderer.animateOnce(0)
	requestLiveAvatarFrame()
}

// retryLiveAvatar builds the shown draft again after it failed.
export function retryLiveAvatar(): void {
	if (!outfitRenderer || !shownDraft) return
	liveAvatar.isFailed = false
	outfitRenderer.setOutfit(outfitOf(shownDraft))
}

async function setUpRenderer(): Promise<boolean> {
	// The rig, the classic body meshes, and the compositing meshes download
	// from Roblox, so the application bundles no Roblox files.
	FLAGS.ONLINE_ASSETS = true
	FLAGS.FETCH_FUNC = fetchLiveAvatarFile
	FLAGS.IMAGE_FUNC = liveAvatarImageUrl
	// The backend already repeats a failed download.
	FLAGS.API_REQUEST_RETRY = false
	FLAGS.AUDIO_ENABLED = false
	RBXRenderer.createLoadingIcon = false
	const canvas = document.createElement("canvas")
	try {
		await loadDracoDecoder()
		RBXRenderer.createContainer()
		await RBXRenderer.boilerplateSetup()
		RBXRenderer.create(canvas)
		RBXRenderer.setupScene()
		RBXRenderer.setupControls()
		RBXRenderer.setBackgroundTransparent(true)
	} catch {
		// WebGL 2 is unavailable, such as on a computer without a usable GPU
		// driver, or the decoder did not load.
		return false
	}
	const controls = RBXRenderer.getRendererControls(),
		camera = RBXRenderer.firstScene.camera
	if (controls) {
		const initialCameraPosition = camera.position.clone(),
			initialTarget = controls.target.clone()
		const resetView = (): void => {
			camera.position.copy(initialCameraPosition)
			controls.target.copy(initialTarget)
			controls.update()
			outfitRenderer?.centerCamera()
		}
		controls.addEventListener("change", requestLiveAvatarFrame)
		canvas.addEventListener("keydown", (event) => {
			if (moveOrbitViewByKey(event, camera, controls, resetView)) {
				event.preventDefault()
			}
		})
	}
	canvas.tabIndex = 0
	canvas.className =
		"cursor-grab rounded-lg -outline-offset-2 focus-visible:outline-2 focus-visible:outline-focus active:cursor-grabbing"
	RBXRenderer.firstScene.compiledRenderDesc.Connect(requestLiveAvatarFrame)
	API.Events.OnLoadingAssets.Connect(() => {
		liveAvatar.isLoading = API.Misc.getCurrentlyLoading()
		requestLiveAvatarFrame()
	})
	return true
}

// loadDracoDecoder loads the decoder of compressed Roblox meshes, which the
// renderer reads from the global scope.
function loadDracoDecoder(): Promise<void> {
	return new Promise((resolve, reject) => {
		const script = document.createElement("script")
		script.src = dracoDecoderUrl
		script.onload = () => resolve()
		script.onerror = () => reject(new Error("The mesh decoder did not load."))
		document.head.append(script)
	})
}

function showCanvasIn({ host, canvasLabel, descriptionId }: LiveAvatarHost): void {
	const canvas = RBXRenderer.getRendererDom()
	canvas?.setAttribute("aria-label", canvasLabel)
	canvas?.setAttribute("aria-describedby", descriptionId)
	host.append(RBXRenderer.getRendererElement())
	hostResizeObserver?.disconnect()
	hostResizeObserver = new ResizeObserver(([entry]) => {
		if (!entry) return
		const { width, height } = entry.contentRect
		if (width === 0 || height === 0) return
		RBXRenderer.setRendererSize(Math.round(width), Math.round(height))
		requestLiveAvatarFrame()
	})
	hostResizeObserver.observe(host)
}

// requestLiveAvatarFrame draws one frame on the next animation frame. The
// canvas draws only when the avatar or the view changes.
function requestLiveAvatarFrame(): void {
	if (isFramePending || hosts.length === 0) return
	isFramePending = true
	requestAnimationFrame(() => {
		isFramePending = false
		RBXRenderer.animateAll(false)
	})
}

// outfitOf writes the draft in the form that the renderer reads, as the
// Roblox avatar API lists an avatar. The render API derives the depth of an
// avatar from its width, as the Roblox avatar editor does.
function outfitOf(draft: AvatarDraft): Outfit {
	const outfit = new Outfit()
	outfit.fromJson({
		playerAvatarType: draft.avatarType === AvatarType.AvatarR6 ? "R6" : "R15",
		scale: { ...draft.scales, depth: draft.scales.width },
		bodyColor3s: {
			headColor3: draft.bodyColors.head,
			torsoColor3: draft.bodyColors.torso,
			leftArmColor3: draft.bodyColors.leftArm,
			rightArmColor3: draft.bodyColors.rightArm,
			leftLegColor3: draft.bodyColors.leftLeg,
			rightLegColor3: draft.bodyColors.rightLeg,
		},
		assets: draft.assets.map((asset) => ({
			id: asset.item.itemId,
			name: asset.item.name,
			assetType: { name: asset.item.typeName },
			meta:
				asset.adjustment || asset.layer
					? {
							...asset.adjustment,
							order: asset.layer?.order,
							puffiness: asset.layer?.puffiness,
						}
					: undefined,
		})),
	})
	return outfit
}
