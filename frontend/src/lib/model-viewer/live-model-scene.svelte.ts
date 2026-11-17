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
import { moveOrbitViewByKey } from "./orbit-view-keys"
import { defaultModelViewerScene, modelViewerScene } from "./model-viewer-scene.svelte"
import type { ModelViewerScene } from "./model-viewer-scene.svelte"
import {
	clearLiveModelFiles,
	fetchLiveModelFile,
	liveModelImageUrl,
	setLiveModelAccount,
} from "./live-model-files"
import type { AvatarLook, LiveModelSubject } from "./model-subject"

// CameraView is where the camera of a view stood when another view took the
// canvas, so that the view returns to it.
interface CameraView {
	position: [number, number, number]
	target: [number, number, number]
	fieldOfViewDegrees: number
}

// The renderer places the camera of an avatar for this field of view, and
// Roblox places the thumbnail camera of an accessory for it too.
const robloxFieldOfViewDegrees = 70,
	keyLightDistance = 15,
	maxCameraDistance = 200

// The renderer is one scene on one canvas for the whole application. Every
// view of a 3D model shares it: the newest view shows the canvas, and a view
// under it, such as the avatar page under an item dialog, shows a still image
// of its last frame until the canvas returns to it.
const viewports: LiveModelViewport[] = []
let rendererSetup: Promise<boolean> | undefined,
	outfitRenderer: OutfitRenderer | undefined,
	shownAccessory: { root: Instance; accessory: Instance } | undefined,
	shownSubject: LiveModelSubject | undefined,
	accessoryLoadNumber = 0,
	hostResizeObserver: ResizeObserver | undefined,
	isFramePending = false,
	// The base strengths of the lights, which the camera and lighting
	// settings scale.
	ambientLightIntensity = 0,
	keyLightIntensity = 0

const authentication = new Authentication()

// LiveModelViewport is one view of the scene. It holds what the view shows,
// and the state that the view draws around the canvas.
export class LiveModelViewport {
	// Whether the renderer downloads or builds a part of the subject.
	isLoading = $state(false)
	// Whether the subject has been built since the view opened.
	hasShownSubject = $state(false)
	// Whether the subject could not be built, such as when a file failed to
	// download.
	isFailed = $state(false)
	// The last frame of the view, while another view shows the canvas.
	coveredFrameUrl = $state("")
	readonly host: HTMLElement
	readonly canvasLabel: string
	readonly descriptionId: string
	accountId = 0
	subject: LiveModelSubject | undefined
	cameraView: CameraView | undefined

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
		if (activeViewport() === this) showSubject(this)
	}

	// retry builds the subject again after it failed.
	retry(): void {
		if (activeViewport() !== this) return
		shownSubject = undefined
		showSubject(this)
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
		clearLiveModelFiles()
	}
}

// startLiveModelScene sets up the renderer once, and reports whether it can
// draw, which needs WebGL 2. The setup downloads the meshes that compose
// clothing textures, so the account that downloads them is chosen first.
export function startLiveModelScene(accountId: number): Promise<boolean> {
	setLiveModelAccount(accountId)
	rendererSetup ??= setUpRenderer()
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
		coveredViewport.coveredFrameUrl = captureFrame()
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
	if (viewport.subject) showSubject(viewport)
	if (viewport.cameraView) restoreCameraView(viewport.cameraView)
	requestFrame()
}

// showSubject builds the subject of the active view, unless the scene shows
// it already.
function showSubject(viewport: LiveModelViewport): void {
	const subject = viewport.subject
	if (!subject || subject === shownSubject) return
	const previousSubject = shownSubject
	shownSubject = subject
	setLiveModelAccount(viewport.accountId)
	viewport.isFailed = false
	if (subject.kind === "avatar") {
		removeAccessory()
		const isFirstAvatar = !outfitRenderer || previousSubject?.kind !== "avatar"
		showOutfit(outfitOf(subject.look))
		if (isFirstAvatar && !viewport.cameraView) frameSubject(subject)
		return
	}
	outfitRenderer?.destroy()
	outfitRenderer = undefined
	void showAccessory(subject.assetId, viewport)
}

function showOutfit(outfit: Outfit): void {
	if (outfitRenderer) {
		outfitRenderer.setOutfit(outfit)
		return
	}
	const createdRenderer = new OutfitRenderer(authentication, outfit)
	createdRenderer.onSuccess.Connect(() => {
		const viewport = activeViewport()
		if (viewport) viewport.hasShownSubject = true
		void poseAvatar(createdRenderer)
		requestFrame()
	})
	createdRenderer.onError.Connect(() => {
		const viewport = activeViewport()
		if (viewport) viewport.isFailed = true
	})
	createdRenderer.onRenderSuccess.Connect(requestFrame)
	outfitRenderer = createdRenderer
}

// poseAvatar puts the avatar in the first frame of its idle animation, which
// is the pose of the Roblox avatar editor. Without it, the rig stands in its
// bind pose. The animation does not run, so the canvas draws only on changes.
async function poseAvatar(posedRenderer: OutfitRenderer): Promise<void> {
	await posedRenderer.setMainAnimation("idle:0")
	posedRenderer.animateOnce(0)
	requestFrame()
}

// showAccessory downloads the accessory as a mesh part, the form in which an
// avatar wears it, and frames it with its thumbnail camera.
async function showAccessory(
	assetId: number,
	viewport: LiveModelViewport,
): Promise<void> {
	removeAccessory()
	const loadNumber = ++accessoryLoadNumber,
		model = await API.Asset.GetRBX(`rbxassetid://${assetId}`, {
			"Roblox-AssetFormat": "avatar_meshpart_accessory",
		})
	if (loadNumber !== accessoryLoadNumber) return
	if (model instanceof Response) {
		viewport.isFailed = true
		return
	}
	const root = model.generateTree(),
		accessory = root.FindFirstChildOfClass("Accessory")
	if (!accessory) {
		root.Destroy()
		viewport.isFailed = true
		return
	}
	shownAccessory = { root, accessory }
	RBXRenderer.addInstance(accessory, authentication)
	viewport.hasShownSubject = true
	if (!viewport.cameraView) frameSubject(viewport.subject)
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
	removeAccessory()
	outfitRenderer?.destroy()
	outfitRenderer = shownSubject = undefined
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
		)
		outfitRenderer?.centerCamera()
		return
	}
	if (!shownAccessory) return
	const thumbnailCamera = getThumbnailCamera(shownAccessory.accessory),
		cameraFrame = thumbnailCamera.Prop("CFrame") as CFrame | undefined,
		handle = shownAccessory.accessory.FindFirstChild("Handle"),
		handlePosition = handle?.Prop("Position") as RobloxVector3 | undefined
	if (
		thumbnailCamera !== shownAccessory.accessory.FindFirstChild("ThumbnailCamera")
	) {
		thumbnailCamera.Destroy()
	}
	if (!cameraFrame || !handlePosition) return
	placeCamera(cameraFrame.Position, [
		handlePosition.X,
		handlePosition.Y,
		handlePosition.Z,
	])
}

// placeCamera points the camera at the target from a position that Roblox
// chose for its own field of view, moved to keep the same framing at the field
// of view of the viewer.
function placeCamera(
	robloxPosition: readonly [number, number, number],
	target: readonly [number, number, number],
): void {
	const camera = RBXRenderer.firstScene.camera,
		controls = RBXRenderer.getRendererControls()
	if (!controls) return
	controls.target.set(...target)
	camera.position.set(...robloxPosition)
	moveCameraForFieldOfView(robloxFieldOfViewDegrees, camera.fov)
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
	const framingScale =
			Math.tan((framedFieldOfViewDegrees * Math.PI) / 360) /
			Math.tan((fieldOfViewDegrees * Math.PI) / 360),
		offset = camera.position
			.clone()
			.sub(controls.target)
			.multiplyScalar(framingScale)
	camera.position.copy(controls.target).add(offset)
}

function currentCameraView(): CameraView | undefined {
	const camera = RBXRenderer.firstScene.camera,
		controls = RBXRenderer.getRendererControls()
	if (!controls) return undefined
	return {
		position: [camera.position.x, camera.position.y, camera.position.z],
		target: [controls.target.x, controls.target.y, controls.target.z],
		fieldOfViewDegrees: camera.fov,
	}
}

function restoreCameraView(cameraView: CameraView): void {
	const camera = RBXRenderer.firstScene.camera,
		controls = RBXRenderer.getRendererControls()
	if (!controls) return
	controls.target.set(...cameraView.target)
	camera.position.set(...cameraView.position)
	moveCameraForFieldOfView(cameraView.fieldOfViewDegrees, camera.fov)
	controls.update()
}

// captureFrame draws a frame and reads it at once, because the canvas keeps
// its pixels only until the browser shows them.
function captureFrame(): string {
	const canvas = RBXRenderer.getRendererDom()
	if (!canvas) return ""
	RBXRenderer.animateAll(false)
	return canvas.toDataURL("image/png")
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
// only when the subject or the view changes.
function requestFrame(): void {
	if (isFramePending || viewports.length === 0) return
	isFramePending = true
	requestAnimationFrame(() => {
		isFramePending = false
		placeKeyLight(modelViewerScene)
		RBXRenderer.animateAll(false)
	})
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
