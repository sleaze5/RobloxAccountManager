import dracoDecoderUrl from "../../../node_modules/roavatar-renderer/dist/thirdparty/draco/javascript/draco_decoder.js?url"
import {
	API,
	Authentication,
	FLAGS,
	Outfit,
	OutfitRenderer,
	RBXRenderer,
	getThumbnailCamera,
} from "roavatar-renderer"
import type { CFrame, Instance, Vector3 as RobloxVector3 } from "roavatar-renderer"
import { AvatarType } from "../backend/bridge"
import { reducedMotion } from "../shared/motion"
import { placeAdjustmentGuide, removeAdjustmentGuides } from "./adjustment-guides"
import { cameraPoseOf, framingScaleOf, moveOrbitViewByKey } from "./orbit-camera"
import { defaultModelViewerScene, modelViewerScene } from "./model-viewer-scene.svelte"
import type { ModelViewerScene } from "./model-viewer-scene.svelte"
import {
	animationChoiceOfItem,
	animationIdByAssetTypeName,
	animationSetChoices,
	assetAnimationPrefix,
	emoteAnimationPrefix,
	emoteChoiceOf,
	idleAnimationId,
	uploadedAnimationPrefix,
} from "./animation-choices"
import {
	clearLiveModelCache,
	fetchLiveModelFile,
	liveModelImageUrl,
	setLiveModelAccount,
} from "./live-model-files"
import type {
	AdjustmentGuide,
	AnimationChoice,
	AnimationPlayback,
	AnimationStart,
	AvatarLook,
	CameraPose,
	LiveModelSubject,
} from "./model-subject"

// CameraView is where the camera of a view stood when another view took the
// canvas, so that the view returns to it.
interface CameraView {
	pose: CameraPose
	fieldOfViewDegrees: number
}

// AnimationFrame is where an animation starts: at a time, at a keyframe, or
// in its middle, which is where Roblox poses a thumbnail whose emote names no
// frame.
type AnimationFrame =
	| { kind: "time"; timeSeconds: number }
	| { kind: "keyframe"; keyframeIndex: number }
	| { kind: "middle" }

// AnimationAsset is what the viewer reads from an emote or animation asset:
// the animation that it plays, and the frame that a thumbnail shows.
interface AnimationAsset {
	animationId: number
	thumbnailFrame: AnimationFrame
}

// The renderer places the camera of an avatar for this field of view, and
// Roblox places the thumbnail camera of an accessory for it unless the
// accessory has a camera of its own.
const robloxFieldOfViewDegrees = 70,
	keyLightDistance = 15,
	maxCameraDistance = 200,
	shadowResolutionPx = 2048,
	// A frame after a long pause, such as while the window was hidden, moves
	// an animation at most this far.
	maxPlaybackStepSeconds = 0.1,
	// The renderer poses a forced time of -1 in the middle of the animation.
	middleAnimationTimeSeconds = -1,
	thumbnailDetail = "Thumbnail"

// The animation assets that the viewer read for the account, by asset ID. A
// failed read leaves the map, so that the next choice reads it again.
const animationAssetById = new Map<number, Promise<AnimationAsset>>()

// The renderer is one scene on one canvas for the whole application. Every
// view of a 3D model shares it: the newest view shows the canvas, and a view
// under it, such as the avatar page under an item dialog, shows a still image
// of its last frame until the canvas returns to it.
const viewports: LiveModelViewport[] = []
let rendererSetup: Promise<boolean> | undefined,
	outfitRenderer: OutfitRenderer | undefined,
	animationPose: { renderer: OutfitRenderer; completion: Promise<void> } | undefined,
	shownAccessory: { root: Instance; accessory: Instance } | undefined,
	shownSubject: LiveModelSubject | undefined,
	shownAccountId = 0,
	shownSubjectStatus: "loading" | "shown" | "failed" = "loading",
	accessoryLoadNumber = 0,
	hostResizeObserver: ResizeObserver | undefined,
	isFramePending = false,
	playbackFrameRequest = 0,
	lastPlaybackFrameMs = 0,
	// The base strengths of the lights, which the camera and lighting
	// settings scale.
	ambientLightIntensity = 0,
	keyLightIntensity = 0

const authentication = new Authentication()

// LiveModelViewport is one view of the scene. It holds what the view shows,
// the playback of its animations, and the state that the view draws around
// the canvas.
export class LiveModelViewport implements AnimationPlayback {
	// Whether the renderer downloads or builds a part of the subject.
	isLoading = $state(false)
	// Whether the subject has been built since the view opened.
	hasShownSubject = $state(false)
	// Whether the subject could not be built, such as when a file failed to
	// download.
	isFailed = $state(false)
	// The last frame of the view, while another view shows the canvas.
	coveredFrameUrl = $state("")
	cameraPose = $state.raw<CameraPose | undefined>(undefined)
	animationChoices = $state.raw<AnimationChoice[]>([])
	animationId = $state(idleAnimationId)
	isPlaying = $state(false)
	isAnimationLoading = $state(false)
	isAnimationFailed = $state(false)
	timeSeconds = $state(0)
	lengthSeconds = $state(0)
	speed = $state(1)
	readonly host: HTMLElement
	readonly canvasLabel: string
	readonly descriptionId: string
	accountId = 0
	subject: LiveModelSubject | undefined
	adjustmentGuide: AdjustmentGuide | undefined
	cameraView: CameraView | undefined
	// The first animation of the subject that chose the animation, so that a
	// changed draft keeps the animation that the user chose.
	#appliedFirstAnimationId = ""
	// Whether the user chose an animation, which a thumbnail that loads later
	// keeps.
	#isAnimationChosen = false
	// The choices that the user played from outside the subject, such as an
	// owned emote, newest last.
	#playedChoices: AnimationChoice[] = []
	// The frame that the next pose starts the animation at, once.
	#startFrame: { animationId: string; frame: Promise<AnimationFrame> } | undefined

	constructor(host: HTMLElement, canvasLabel: string, descriptionId: string) {
		this.host = host
		this.canvasLabel = canvasLabel
		this.descriptionId = descriptionId
	}

	// show builds the subject with the files that the account downloads.
	// Only the changed parts of an avatar are built again, so a moved
	// accessory follows its slider.
	show(accountId: number, subject: LiveModelSubject): void {
		this.accountId = accountId
		this.subject = subject
		if (subject.kind === "avatar") {
			const firstAnimationId = firstAnimationIdOf(subject)
			this.animationChoices = animationChoicesOf(subject, this.#playedChoices)
			if (
				firstAnimationId !== this.#appliedFirstAnimationId &&
				(!this.#isAnimationChosen || !!subject.startAnimation)
			) {
				this.#appliedFirstAnimationId = firstAnimationId
				this.#isAnimationChosen = false
				this.#selectAnimation(firstAnimationId)
				if (!subject.startAnimation && subject.thumbnailAnimation) {
					this.#startFrame = {
						animationId: firstAnimationId,
						frame: thumbnailFrameOf(firstAnimationId),
					}
				}
				// A preview plays what it shows, and an avatar its thumbnail
				// animation, unless motion is reduced.
				this.isPlaying =
					(!!subject.startAnimation || !!subject.thumbnailAnimation) &&
					!reducedMotion()
			}
		} else {
			this.animationChoices = []
			this.isPlaying = false
		}
		if (activeViewport() === this) showSubject(this)
	}

	// showAdjustmentGuide shows the guides of the accessory that the user
	// adjusts, or hides them without a guide.
	showAdjustmentGuide(guide: AdjustmentGuide | undefined): void {
		this.adjustmentGuide = guide
		if (activeViewport() === this) requestFrame()
	}

	// chooseAnimation plays the choice, and keeps a choice from outside the
	// subject, such as a pasted emote, among the choices.
	chooseAnimation(choice: AnimationChoice): void {
		if (!this.animationChoices.some((shown) => shown.id === choice.id)) {
			this.#playedChoices = [
				...this.#playedChoices.filter((played) => played.id !== choice.id),
				choice,
			]
			this.animationChoices = [...this.animationChoices, choice]
		}
		this.#isAnimationChosen = true
		this.#selectAnimation(choice.id)
		if (activeViewport() === this) poseAvatar(this)
	}

	// takeStartFrame returns the frame that the animation starts at once,
	// such as the frame of a thumbnail, or undefined to start at the time of
	// the view.
	takeStartFrame(animationId: string): Promise<AnimationFrame> | undefined {
		const startFrame = this.#startFrame
		this.#startFrame = undefined
		return startFrame?.animationId === animationId ? startFrame.frame : undefined
	}

	#selectAnimation(animationId: string): void {
		this.animationId = animationId
		this.timeSeconds = 0
		this.isAnimationFailed = false
		this.#startFrame = undefined
	}

	play(): void {
		this.isPlaying = true
		if (activeViewport() === this) startPlayback()
	}

	pause(): void {
		this.isPlaying = false
	}

	// seek shows the animation at the time. A playing animation goes on from
	// there.
	seek(timeSeconds: number): void {
		this.timeSeconds = timeSeconds
		if (activeViewport() !== this || !outfitRenderer || this.isAnimationLoading) {
			return
		}
		outfitRenderer.animateOnce(0, timeSeconds)
		readAnimationTime(this)
		requestFrame()
	}

	setSpeed(speed: number): void {
		this.speed = speed
	}

	// retry builds the subject again after it failed.
	retry(): void {
		if (activeViewport() !== this) return
		clearLiveModelCache()
		discardOutfitRenderer()
		shownSubject = undefined
		showSubject(this)
	}

	// captureShownFrame returns the frame that the view shows now as a PNG data
	// URL, with a transparent background.
	captureShownFrame(): string {
		return activeViewport() === this ? captureFrame(true) : this.coveredFrameUrl
	}

	// resetView returns the camera to where it starts for the subject.
	resetView(): void {
		if (activeViewport() !== this) return
		frameSubject(this.subject)
	}

	// detach removes the view. The view under it shows the canvas again, and
	// when no view is left, the subject is removed from the scene, so that its
	// meshes and textures are freed.
	detach(): void {
		const isActive = activeViewport() === this,
			viewportIndex = viewports.indexOf(this)
		if (viewportIndex < 0) return
		viewports.splice(viewportIndex, 1)
		if (!isActive) return
		const previousViewport = activeViewport()
		if (previousViewport) {
			activateViewport(previousViewport)
			return
		}
		hostResizeObserver?.disconnect()
		RBXRenderer.getRendererElement().remove()
		clearScene()
		setLiveModelAccount(0)
	}
}

// startLiveModelScene sets up the renderer once, and reports whether it can
// draw, which needs WebGL 2. The setup downloads the meshes that compose
// clothing textures, so the account that downloads them is chosen first.
export function startLiveModelScene(accountId: number): Promise<boolean> {
	if (!rendererSetup) {
		setLiveModelAccount(accountId)
		rendererSetup = setUpRenderer()
	}
	return rendererSetup
}

// attachLiveModelView adds a view on top of the others and gives it the
// canvas. The view under it keeps a still image of its last frame.
export function attachLiveModelView(
	host: HTMLElement,
	canvasLabel: string,
	descriptionId: string,
): LiveModelViewport {
	const coveredViewport = activeViewport()
	if (coveredViewport) {
		coveredViewport.coveredFrameUrl = captureFrame(false)
		coveredViewport.cameraView = currentCameraView()
	}
	const viewport = new LiveModelViewport(host, canvasLabel, descriptionId)
	viewports.push(viewport)
	activateViewport(viewport)
	return viewport
}

// applyLiveModelSceneSettings follows the camera and lighting settings. A
// change of the field of view moves the camera along its line of sight, so
// that the subject stays framed.
export function applyLiveModelSceneSettings(sceneSettings: ModelViewerScene): void {
	const renderScene = RBXRenderer.firstScene,
		camera = renderScene.camera,
		controls = RBXRenderer.getRendererControls()
	if (!controls) return
	if (camera.fov !== sceneSettings.fieldOfViewDegrees) {
		moveCameraForFieldOfView(camera.fov, sceneSettings.fieldOfViewDegrees)
		camera.fov = sceneSettings.fieldOfViewDegrees
		camera.updateProjectionMatrix()
		controls.update()
	}
	if (renderScene.ambientLight) {
		renderScene.ambientLight.intensity =
			(ambientLightIntensity * sceneSettings.brightness) /
			defaultModelViewerScene.brightness
	}
	if (renderScene.directionalLight) {
		renderScene.directionalLight.intensity =
			(keyLightIntensity * sceneSettings.lightIntensity) /
			defaultModelViewerScene.lightIntensity
	}
	requestFrame()
}

function activeViewport(): LiveModelViewport | undefined {
	return viewports.at(-1)
}

function activateViewport(viewport: LiveModelViewport): void {
	const canvas = RBXRenderer.getRendererDom()
	canvas?.setAttribute("aria-label", viewport.canvasLabel)
	canvas?.setAttribute("aria-describedby", viewport.descriptionId)
	viewport.host.append(RBXRenderer.getRendererElement())
	viewport.coveredFrameUrl = ""
	hostResizeObserver?.disconnect()
	hostResizeObserver = new ResizeObserver(([entry]) => {
		if (!entry) return
		const { width, height } = entry.contentRect
		if (width === 0 || height === 0) return
		RBXRenderer.setRendererSize(Math.round(width), Math.round(height))
		requestFrame()
	})
	hostResizeObserver.observe(viewport.host)
	// The loading state of a covered view stopped following the renderer.
	viewport.isLoading = API.Misc.getCurrentlyLoading()
	if (viewport.subject) showSubject(viewport)
	if (viewport.cameraView) restoreCameraView(viewport.cameraView)
	requestFrame()
}

// showSubject builds the subject of the active view, unless the scene shows
// it already.
function showSubject(viewport: LiveModelViewport): void {
	const subject = viewport.subject
	if (!subject) return
	if (subject === shownSubject && viewport.accountId === shownAccountId) {
		viewport.isFailed = shownSubjectStatus === "failed"
		if (shownSubjectStatus === "shown") {
			viewport.hasShownSubject = true
			if (subject.kind === "avatar") poseAvatar(viewport)
		}
		return
	}
	if (viewport.accountId !== shownAccountId) clearScene()
	const previousSubject = shownSubject
	shownAccountId = viewport.accountId
	shownSubject = subject
	shownSubjectStatus = "loading"
	setLiveModelAccount(viewport.accountId)
	viewport.isFailed = false
	if (subject.kind === "avatar") {
		removeAccessory()
		const isFirstAvatar = !outfitRenderer || previousSubject?.kind !== "avatar"
		showOutfit(outfitOf(subject.look))
		// The renderer centers the camera on the avatar each time it finishes
		// building it, which would undo a pan after every change, such as a
		// moved accessory. It centers only when a view first shows the avatar.
		if (outfitRenderer)
			outfitRenderer.doCameraUpdateOnLoad = !viewport.hasShownSubject
		if (isFirstAvatar && !viewport.cameraView) frameSubject(subject)
		return
	}
	discardOutfitRenderer()
	void showAccessory(subject)
}

function showOutfit(outfit: Outfit): void {
	if (outfitRenderer) {
		outfitRenderer.setOutfit(outfit)
		return
	}
	const createdRenderer = new OutfitRenderer(authentication, outfit)
	createdRenderer.onSuccess.Connect(() => {
		if (outfitRenderer !== createdRenderer || createdRenderer.hasNewUpdate) return
		const viewport = finishSubjectBuild("shown")
		if (viewport) poseAvatar(viewport)
		requestFrame()
	})
	createdRenderer.onError.Connect(() => {
		if (outfitRenderer === createdRenderer) finishSubjectBuild("failed")
	})
	createdRenderer.onRenderSuccess.Connect(requestFrame)
	outfitRenderer = createdRenderer
}

function finishSubjectBuild(status: "shown" | "failed"): LiveModelViewport | undefined {
	shownSubjectStatus = status
	const viewport = activeViewport()
	if (
		!viewport ||
		viewport.subject !== shownSubject ||
		viewport.accountId !== shownAccountId
	)
		return undefined
	viewport.isFailed = status === "failed"
	if (status === "shown") viewport.hasShownSubject = true
	return viewport
}

// poseAvatar loads the chosen animation of the view and shows it at the time
// of the view, then plays it when the view plays. Without an animation, the
// rig stands in its bind pose. The first variant of the idle animation is the
// pose of the Roblox avatar editor.
function poseAvatar(viewport: LiveModelViewport): void {
	const posedRenderer = outfitRenderer
	if (!posedRenderer) return
	const animationId = viewport.animationId,
		posedSubject = shownSubject,
		previousPose =
			animationPose?.renderer === posedRenderer
				? animationPose.completion
				: undefined
	viewport.isAnimationLoading = true
	const isCurrent = (): boolean =>
		animationPose === requestedPose &&
		posedRenderer === outfitRenderer &&
		posedSubject === shownSubject &&
		activeViewport() === viewport
	// The renderer changes the animation as part of loading it. Finish an
	// older load before asking the same renderer to play a different track.
	const applyPose = async (): Promise<void> => {
		await previousPose
		if (!isCurrent()) return
		const startFrame = viewport.takeStartFrame(animationId)
		try {
			const isPlayed = await posedRenderer.setMainAnimation(
				await rendererAnimationNameOf(animationId),
			)
			if (!isCurrent()) return
			// The renderer resolves false for an emote or animation that did
			// not load. An animation of the set that the rig lacks stands in
			// its bind pose instead.
			if (!isPlayed && !isAnimationSetChoice(animationId)) {
				throw new Error("The animation did not play.")
			}
			const frame = await startFrame?.catch(() => undefined)
			if (!isCurrent()) return
			showAnimationFrame(
				posedRenderer,
				frame ?? { kind: "time", timeSeconds: viewport.timeSeconds },
			)
			readAnimationTime(viewport)
			if (viewport.isPlaying) startPlayback()
			requestFrame()
		} catch {
			if (isCurrent()) {
				viewport.pause()
				viewport.isAnimationFailed = true
			}
		} finally {
			if (isCurrent()) viewport.isAnimationLoading = false
		}
	}
	const requestedPose = { renderer: posedRenderer, completion: applyPose() }
	animationPose = requestedPose
}

function showAnimationFrame(renderer: OutfitRenderer, frame: AnimationFrame): void {
	if (frame.kind === "keyframe")
		renderer.animateOnce(0, undefined, frame.keyframeIndex)
	else if (frame.kind === "middle")
		renderer.animateOnce(0, middleAnimationTimeSeconds)
	else renderer.animateOnce(0, frame.timeSeconds)
}

// rendererAnimationNameOf names the animation as the renderer plays it. The
// first variant of the idle animation is the pose of the Roblox avatar
// editor, and an asset of an animation set plays its first animation.
async function rendererAnimationNameOf(animationId: string): Promise<string> {
	if (animationId === idleAnimationId) return `${idleAnimationId}:0`
	if (!animationId.startsWith(assetAnimationPrefix)) return animationId
	const asset = await readAnimationAsset(assetIdOf(animationId))
	return `${uploadedAnimationPrefix}${asset.animationId}`
}

function isAnimationSetChoice(animationId: string): boolean {
	return animationSetChoices.some((choice) => choice.id === animationId)
}

function assetIdOf(animationId: string): number {
	return Number(animationId.slice(animationId.indexOf(".") + 1))
}

// thumbnailFrameOf reads the frame of the animation that a thumbnail shows.
// An emote or animation item names it with a ThumbnailKeyframe or a
// ThumbnailTime value, as RoAvatar reads it. An uploaded animation names
// none.
async function thumbnailFrameOf(animationId: string): Promise<AnimationFrame> {
	if (
		!animationId.startsWith(emoteAnimationPrefix) &&
		!animationId.startsWith(assetAnimationPrefix)
	) {
		return { kind: "middle" }
	}
	return (await readAnimationAsset(assetIdOf(animationId))).thumbnailFrame
}

function readAnimationAsset(assetId: number): Promise<AnimationAsset> {
	const knownAsset = animationAssetById.get(assetId)
	if (knownAsset) return knownAsset
	const reading = downloadAnimationAsset(assetId)
	animationAssetById.set(assetId, reading)
	reading.catch(() => {
		if (animationAssetById.get(assetId) === reading)
			animationAssetById.delete(assetId)
	})
	return reading
}

// downloadAnimationAsset reads the first animation of the asset. An emote
// holds one Animation, and an asset of an animation set holds a folder of
// variants for each animation that it replaces.
async function downloadAnimationAsset(assetId: number): Promise<AnimationAsset> {
	const model = await API.Asset.GetRBX(`rbxassetid://${assetId}`)
	if (model instanceof Response) throw new Error("The animation did not download.")
	const root = model.generateTree()
	try {
		const animation = root
			.GetDescendants()
			.find(
				(descendant) =>
					descendant.className === "Animation" &&
					String(descendant.PropOrDefault("AnimationId", "")).length > 0,
			)
		if (!animation) throw new Error("The asset holds no animation.")
		const thumbnailKeyframe = animation
				.FindFirstChild("ThumbnailKeyframe")
				?.PropOrDefault("Value", undefined),
			thumbnailTime = animation
				.FindFirstChild("ThumbnailTime")
				?.PropOrDefault("Value", undefined)
		return {
			animationId: API.Misc.idFromStr(String(animation.Prop("AnimationId"))),
			thumbnailFrame:
				typeof thumbnailKeyframe === "number"
					? { kind: "keyframe", keyframeIndex: thumbnailKeyframe }
					: typeof thumbnailTime === "number"
						? { kind: "time", timeSeconds: thumbnailTime }
						: { kind: "middle" },
		}
	} finally {
		root.Destroy()
	}
}

function discardOutfitRenderer(): void {
	const discardedRenderer = outfitRenderer
	outfitRenderer = undefined
	animationPose = undefined
	if (!discardedRenderer) return
	// RoAvatar can finish creating a rig after destroy(). Do not let it add
	// the late rig to the shared scene or move the camera.
	discardedRenderer.doAddInstance = false
	discardedRenderer.doCameraUpdateOnLoad = false
	const wasBuilding = discardedRenderer.currentlyUpdating
	discardedRenderer.destroy()
	if (wasBuilding) {
		const destroyLateRig = (): void => {
			discardedRenderer.currentRig?.Destroy()
			discardedRenderer.onSuccess.Clear()
			discardedRenderer.onError.Clear()
		}
		discardedRenderer.onSuccess.Connect(destroyLateRig)
		discardedRenderer.onError.Connect(destroyLateRig)
	}
}

// startPlayback draws a frame on every animation frame while the active view
// plays, and stops when it pauses. Only playback draws continuously.
function startPlayback(): void {
	if (playbackFrameRequest) return
	lastPlaybackFrameMs = performance.now()
	playbackFrameRequest = requestAnimationFrame(playNextFrame)
}

function playNextFrame(frameMs: number): void {
	playbackFrameRequest = 0
	const viewport = activeViewport()
	if (!viewport) return
	if (!viewport.isPlaying || !outfitRenderer || viewport.isAnimationLoading) {
		drawFrame()
		return
	}
	const elapsedSeconds = Math.min(
		(frameMs - lastPlaybackFrameMs) / 1000,
		maxPlaybackStepSeconds,
	)
	lastPlaybackFrameMs = frameMs
	outfitRenderer.animateOnce(elapsedSeconds * viewport.speed)
	readAnimationTime(viewport)
	drawFrame()
	playbackFrameRequest = requestAnimationFrame(playNextFrame)
}

function readAnimationTime(viewport: LiveModelViewport): void {
	const track = outfitRenderer?.animatorW?.data.currentAnimationTrack
	viewport.timeSeconds = track?.timePosition ?? 0
	viewport.lengthSeconds = track?.length ?? 0
}

// animationChoicesOf lists the animation set, the emotes of the look, the
// thumbnail animation, and the choices that the user played. A preview of an
// animation or an emote offers only that one.
function animationChoicesOf(
	subject: Extract<LiveModelSubject, { kind: "avatar" }>,
	playedChoices: AnimationChoice[],
): AnimationChoice[] {
	const { startAnimation } = subject
	if (startAnimation?.kind === "emote") {
		return [emoteChoiceOf(startAnimation.emote, "emote")]
	}
	if (startAnimation?.kind === "animation") {
		const animationId = startAnimationIdOf(startAnimation)
		return animationSetChoices.filter((choice) => choice.id === animationId)
	}
	const thumbnailChoice = subject.thumbnailAnimation
			? animationChoiceOfItem(subject.thumbnailAnimation, "emote")
			: undefined,
		emoteChoices = [...(subject.look.emotes ?? [])]
			.sort((first, second) => first.slot - second.slot)
			.map((emote) => {
				const choice = emoteChoiceOf(emote.item, "emote")
				choice.detail = `Slot ${emote.slot}`
				choice.imageUrl = emote.item.imageUrl || undefined
				return choice
			}),
		subjectChoices = [...animationSetChoices, ...emoteChoices]
	if (thumbnailChoice) {
		const slotIndex = subjectChoices.findIndex(
			(choice) => choice.id === thumbnailChoice.id,
		)
		if (slotIndex < 0) {
			subjectChoices.push({ ...thumbnailChoice, detail: thumbnailDetail })
		} else {
			const slotChoice = subjectChoices[slotIndex]
			subjectChoices[slotIndex] = {
				...slotChoice,
				detail: `${slotChoice.detail} · ${thumbnailDetail}`,
			}
		}
	}
	return [
		...subjectChoices,
		...playedChoices.filter(
			(played) => !subjectChoices.some((choice) => choice.id === played.id),
		),
	]
}

// firstAnimationIdOf names the animation that the avatar starts with: the
// animation that a preview shows, the thumbnail animation, or the idle pose.
function firstAnimationIdOf(
	subject: Extract<LiveModelSubject, { kind: "avatar" }>,
): string {
	if (subject.startAnimation) return startAnimationIdOf(subject.startAnimation)
	if (subject.thumbnailAnimation) {
		return (
			animationChoiceOfItem(subject.thumbnailAnimation, "emote")?.id ??
			idleAnimationId
		)
	}
	return idleAnimationId
}

function startAnimationIdOf(startAnimation: AnimationStart): string {
	if (startAnimation.kind === "emote") {
		return `${emoteAnimationPrefix}${startAnimation.emote.itemId}`
	}
	return animationIdByAssetTypeName[startAnimation.assetTypeName] ?? idleAnimationId
}

// showAccessory downloads the accessory as a mesh part, the form in which an
// avatar wears it, and frames it with its thumbnail camera.
async function showAccessory(
	subject: Extract<LiveModelSubject, { kind: "accessory" }>,
): Promise<void> {
	removeAccessory()
	const loadNumber = ++accessoryLoadNumber,
		model = await API.Asset.GetRBX(`rbxassetid://${subject.assetId}`, {
			"Roblox-AssetFormat": "avatar_meshpart_accessory",
		}).catch(() => null)
	if (loadNumber !== accessoryLoadNumber) return
	if (!model || model instanceof Response) {
		finishSubjectBuild("failed")
		return
	}
	const root = model.generateTree(),
		accessory = root.FindFirstChildOfClass("Accessory")
	if (!accessory) {
		root.Destroy()
		finishSubjectBuild("failed")
		return
	}
	shownAccessory = { root, accessory }
	RBXRenderer.addInstance(accessory, authentication)
	const viewport = finishSubjectBuild("shown")
	if (!viewport?.cameraView) frameSubject(subject)
	requestFrame()
}

function removeAccessory(): void {
	accessoryLoadNumber++
	if (!shownAccessory) return
	RBXRenderer.removeInstance(shownAccessory.accessory)
	shownAccessory.root.Destroy()
	shownAccessory = undefined
}

function clearScene(): void {
	animationAssetById.clear()
	removeAccessory()
	discardOutfitRenderer()
	shownSubject = undefined
	shownAccountId = 0
	shownSubjectStatus = "loading"
	removeAdjustmentGuides()
}

// frameSubject moves the camera to where it starts for the subject: in front
// of the avatar, as the renderer places it, or where the thumbnail camera of
// the accessory stands.
function frameSubject(subject: LiveModelSubject | undefined): void {
	if (!subject) return
	const renderScene = RBXRenderer.firstScene
	if (subject.kind === "avatar") {
		const target = RBXRenderer.orbitControlsTarget,
			[x, y, z] = renderScene.lookAwayVector,
			distance = renderScene.lookAwayDistance
		placeCamera(
			[
				target[0] + x * distance,
				target[1] + y * distance,
				target[2] + z * distance,
			],
			target,
			robloxFieldOfViewDegrees,
		)
		outfitRenderer?.centerCamera()
		return
	}
	if (!shownAccessory) return
	const thumbnailCamera = getThumbnailCamera(shownAccessory.accessory),
		cameraFrame = thumbnailCamera.Prop("CFrame") as CFrame | undefined,
		cameraFieldOfViewDegrees = thumbnailCamera.PropOrDefault(
			"FieldOfView",
			robloxFieldOfViewDegrees,
		) as number,
		handle = shownAccessory.accessory.FindFirstChild("Handle"),
		handlePosition = handle?.Prop("Position") as RobloxVector3 | undefined
	if (
		thumbnailCamera !== shownAccessory.accessory.FindFirstChild("ThumbnailCamera")
	) {
		thumbnailCamera.Destroy()
	}
	if (!cameraFrame || !handlePosition) return
	placeCamera(
		cameraFrame.Position,
		[handlePosition.X, handlePosition.Y, handlePosition.Z],
		cameraFieldOfViewDegrees,
	)
}

// placeCamera points the camera at the target from a position that Roblox
// chose for the framed field of view, moved to keep the same framing at the
// field of view of the viewer.
function placeCamera(
	robloxPosition: readonly [number, number, number],
	target: readonly [number, number, number],
	framedFieldOfViewDegrees: number,
): void {
	const camera = RBXRenderer.firstScene.camera,
		controls = RBXRenderer.getRendererControls()
	if (!controls) return
	controls.target.set(...target)
	camera.position.set(...robloxPosition)
	moveCameraForFieldOfView(framedFieldOfViewDegrees, camera.fov)
	controls.update()
	requestFrame()
}

// moveCameraForFieldOfView scales the distance from the camera to its target,
// so that a subject framed at one field of view stays framed at the other.
function moveCameraForFieldOfView(
	framedFieldOfViewDegrees: number,
	fieldOfViewDegrees: number,
): void {
	const camera = RBXRenderer.firstScene.camera,
		controls = RBXRenderer.getRendererControls()
	if (!controls) return
	const offset = camera.position
		.clone()
		.sub(controls.target)
		.multiplyScalar(framingScaleOf(framedFieldOfViewDegrees, fieldOfViewDegrees))
	camera.position.copy(controls.target).add(offset)
}

function currentCameraView(): CameraView | undefined {
	const camera = RBXRenderer.firstScene.camera,
		controls = RBXRenderer.getRendererControls()
	if (!controls) return undefined
	return { pose: cameraPoseOf(camera, controls), fieldOfViewDegrees: camera.fov }
}

function restoreCameraView(cameraView: CameraView): void {
	const camera = RBXRenderer.firstScene.camera,
		controls = RBXRenderer.getRendererControls()
	if (!controls) return
	controls.target.set(...cameraView.pose.target)
	camera.position.set(...cameraView.pose.position)
	moveCameraForFieldOfView(cameraView.fieldOfViewDegrees, camera.fov)
	controls.update()
}

// captureFrame draws a frame and reads it at once, because the canvas keeps
// its pixels only until the browser shows them. The frame leaves out the
// adjustment guides, and with isBackgroundTransparent the background and the
// ground. The next frame shows them again.
function captureFrame(isBackgroundTransparent: boolean): string {
	const canvas = RBXRenderer.getRendererDom()
	if (!canvas) return ""
	try {
		RBXRenderer.setBackgroundTransparent(isBackgroundTransparent)
		drawFrame(false)
		return canvas.toDataURL("image/png")
	} catch {
		return ""
	} finally {
		RBXRenderer.setBackgroundTransparent(false)
		requestFrame()
	}
}

async function setUpRenderer(): Promise<boolean> {
	// The rig, the classic body meshes, and the compositing meshes download
	// from Roblox, so the application bundles no Roblox files.
	FLAGS.ONLINE_ASSETS = true
	FLAGS.FETCH_FUNC = fetchLiveModelFile
	FLAGS.IMAGE_FUNC = liveModelImageUrl
	// The backend already repeats a failed download.
	FLAGS.API_REQUEST_RETRY = false
	FLAGS.AUDIO_ENABLED = false
	// Ambient occlusion shades the creases of the avatar, SMAA smooths its
	// edges, and bloom lets neon parts glow. The renderer draws only on
	// changes, so the passes cost nothing while the view stands still.
	FLAGS.USE_POST_PROCESSING = true
	RBXRenderer.createLoadingIcon = false
	RBXRenderer.firstScene.shadowResolution = [shadowResolutionPx, shadowResolutionPx]
	const canvas = document.createElement("canvas"),
		// The scene is opaque in the color of the content plane, because the
		// ambient occlusion pass needs a background behind the avatar.
		backgroundColor = getComputedStyle(document.documentElement)
			.getPropertyValue("--color-canvas")
			.trim()
	try {
		await loadDracoDecoder()
		RBXRenderer.createContainer()
		await RBXRenderer.boilerplateSetup()
		RBXRenderer.create(canvas)
		RBXRenderer.setupScene("WellLit", backgroundColor)
		RBXRenderer.setupControls()
	} catch {
		// WebGL 2 is unavailable, such as on a computer without a usable GPU
		// driver, or the decoder did not load.
		return false
	}
	const renderScene = RBXRenderer.firstScene,
		camera = renderScene.camera,
		controls = RBXRenderer.getRendererControls()
	ambientLightIntensity = renderScene.ambientLight?.intensity ?? 0
	keyLightIntensity = renderScene.directionalLight?.intensity ?? 0
	camera.fov = robloxFieldOfViewDegrees
	if (controls) {
		controls.maxDistance = maxCameraDistance
		controls.addEventListener("change", requestFrame)
		canvas.addEventListener("keydown", (event) => {
			const viewport = activeViewport()
			if (event.key === " " && viewport && viewport.animationChoices.length > 0) {
				event.preventDefault()
				if (viewport.isPlaying) viewport.pause()
				else viewport.play()
				return
			}
			const isViewMoved = moveOrbitViewByKey(event, camera, controls, () =>
				activeViewport()?.resetView(),
			)
			if (isViewMoved) event.preventDefault()
		})
	}
	canvas.tabIndex = 0
	canvas.className =
		"cursor-grab rounded-lg -outline-offset-2 focus-visible:outline-2 focus-visible:outline-focus active:cursor-grabbing"
	renderScene.compiledRenderDesc.Connect(requestFrame)
	API.Events.OnLoadingAssets.Connect(() => {
		const viewport = activeViewport()
		if (viewport) viewport.isLoading = API.Misc.getCurrentlyLoading()
		requestFrame()
	})
	applyLiveModelSceneSettings({ ...modelViewerScene })
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

// requestFrame draws one frame on the next animation frame. The canvas draws
// only when the subject or the view changes, or while an animation plays.
function requestFrame(): void {
	if (isFramePending || playbackFrameRequest || viewports.length === 0) return
	isFramePending = true
	requestAnimationFrame(() => {
		isFramePending = false
		if (viewports.length > 0) drawFrame()
	})
}

function drawFrame(isAdjustmentGuideShown = true): void {
	placeKeyLight(modelViewerScene)
	placeAdjustmentGuide(
		isAdjustmentGuideShown && shownSubject?.kind === "avatar"
			? activeViewport()?.adjustmentGuide
			: undefined,
		outfitRenderer?.currentRig,
	)
	RBXRenderer.animateAll(false)
	const viewport = activeViewport(),
		camera = RBXRenderer.firstScene.camera,
		controls = RBXRenderer.getRendererControls()
	if (!viewport || !controls) return
	viewport.cameraPose = cameraPoseOf(camera, controls)
}

// placeKeyLight keeps the key light at its angle to the view, as in the other
// 3D viewers: 0 degrees shines from the camera, and 90 from its right.
function placeKeyLight(sceneSettings: ModelViewerScene): void {
	const renderScene = RBXRenderer.firstScene,
		keyLight = renderScene.directionalLight,
		controls = RBXRenderer.getRendererControls()
	if (!keyLight || !controls) return
	const horizontalRadians = (sceneSettings.lightAngleXDegrees * Math.PI) / 180,
		verticalRadians = (sceneSettings.lightAngleYDegrees * Math.PI) / 180,
		direction = renderScene.camera.position
			.clone()
			.set(
				Math.sin(horizontalRadians) * Math.cos(verticalRadians),
				Math.sin(verticalRadians),
				Math.cos(horizontalRadians) * Math.cos(verticalRadians),
			)
			.applyQuaternion(renderScene.camera.quaternion)
	keyLight.position.copy(controls.target).addScaledVector(direction, keyLightDistance)
	keyLight.target.position.copy(controls.target)
	keyLight.target.updateMatrixWorld()
}

// outfitOf writes the look in the form that the renderer reads, as the Roblox
// avatar API lists an avatar. The render API derives the depth of an avatar
// from its width, as the Roblox avatar editor does.
function outfitOf(look: AvatarLook): Outfit {
	const outfit = new Outfit()
	outfit.fromJson({
		playerAvatarType: look.avatarType === AvatarType.AvatarR6 ? "R6" : "R15",
		scale: { ...look.scales, depth: look.scales.width },
		bodyColor3s: {
			headColor3: look.bodyColors.head,
			torsoColor3: look.bodyColors.torso,
			leftArmColor3: look.bodyColors.leftArm,
			rightArmColor3: look.bodyColors.rightArm,
			leftLegColor3: look.bodyColors.leftLeg,
			rightLegColor3: look.bodyColors.rightLeg,
		},
		assets: (look.assets ?? []).map((asset) => ({
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
