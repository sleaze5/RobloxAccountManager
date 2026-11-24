<script lang="ts">
	import {
		AmbientLight,
		DirectionalLight,
		LoadingManager,
		MathUtils,
		Mesh,
		PerspectiveCamera,
		Scene,
		Texture,
		Vector3,
		WebGLRenderer,
	} from "three"
	import type { Material, Object3D } from "three"
	import { OrbitControls } from "three/addons/controls/OrbitControls.js"
	import { MTLLoader } from "three/addons/loaders/MTLLoader.js"
	import { OBJLoader } from "three/addons/loaders/OBJLoader.js"
	import { untrack } from "svelte"
	import type { RenderedModel } from "../backend/bridge"
	import type { CameraPose } from "./model-subject"
	import {
		cameraPoseOf,
		framingScaleOf,
		moveOrbitViewByKey,
		orbitViewKeysDescription,
	} from "./orbit-camera"
	import { modelViewerScene } from "./model-viewer-scene.svelte"
	import type { ModelViewerScene } from "./model-viewer-scene.svelte"

	// label names what the model shows, such as "@builderman's avatar".
	// onposechange receives where the camera stands after each frame, for the
	// axes and the coordinates around the viewer.
	let {
		model,
		label,
		onerror,
		onposechange,
	}: {
		model: RenderedModel
		label: string
		onerror: () => void
		onposechange: (pose: CameraPose) => void
	} = $props()

	let canvas = $state<HTMLCanvasElement>()
	let applyScene: ((sceneSettings: ModelViewerScene) => void) | undefined
	let resetCameraView: (() => void) | undefined
	// captureSceneFrame draws the built scene and reads it at once, because the
	// canvas keeps its pixels only until the browser shows them.
	let captureSceneFrame: (() => string) | undefined

	// resetView returns the camera to where it starts, as Home does.
	export function resetView(): void {
		resetCameraView?.()
	}

	// captureFrame returns the frame shown now as a PNG data URL.
	export function captureFrame(): string {
		return captureSceneFrame?.() ?? ""
	}

	const controlsDescriptionId = $props.id(),
		keyLightIntensity = 1.6,
		ambientLightIntensity = 1.4

	$effect(() => {
		if (!canvas) return
		let renderer: WebGLRenderer
		try {
			renderer = new WebGLRenderer({ canvas, alpha: true, antialias: true })
		} catch {
			// WebGL is unavailable, such as on a computer without a usable GPU
			// driver.
			onerror()
			return
		}
		renderer.setPixelRatio(window.devicePixelRatio)
		let isDisposed = false
		const initialScene = untrack(() => ({ ...modelViewerScene })),
			loadingManager = new LoadingManager(),
			scene = new Scene(),
			modelObject = parseModelObject(model, loadingManager),
			center = new Vector3(
				(model.bounds.minimum.x + model.bounds.maximum.x) / 2,
				(model.bounds.minimum.y + model.bounds.maximum.y) / 2,
				(model.bounds.minimum.z + model.bounds.maximum.z) / 2,
			),
			boundsSize = new Vector3(
				model.bounds.maximum.x - model.bounds.minimum.x,
				model.bounds.maximum.y - model.bounds.minimum.y,
				model.bounds.maximum.z - model.bounds.minimum.z,
			).length(),
			camera = new PerspectiveCamera(
				initialScene.fieldOfViewDegrees,
				1,
				0.1,
				1000,
			),
			// Roblox places the camera for its own field of view, so the camera
			// moves to keep the model framed at the chosen one.
			framingScale = framingScaleOf(
				model.camera.fieldOfViewDegrees,
				initialScene.fieldOfViewDegrees,
			),
			initialCameraPosition = new Vector3(
				model.camera.position.x,
				model.camera.position.y,
				model.camera.position.z,
			)
				.sub(center)
				.multiplyScalar(framingScale)
				.add(center),
			initialDistance = initialCameraPosition.distanceTo(center),
			keyLight = new DirectionalLight(0xffffff, keyLightIntensity),
			ambientLight = new AmbientLight(0xffffff, ambientLightIntensity)
		scene.add(modelObject)
		scene.add(ambientLight)
		// The light and its target move with the camera, so that the light angle
		// stays relative to the view.
		keyLight.target.position.set(0, 0, -initialDistance)
		camera.add(keyLight, keyLight.target)
		scene.add(camera)
		camera.position.copy(initialCameraPosition)

		const controls = new OrbitControls(camera, canvas)
		controls.target.copy(center)
		// OrbitControls moves the view while Control, Command, or Shift is held
		// during a drag, and on a right drag.
		controls.enablePan = true
		controls.minDistance = Math.min(boundsSize * 0.3, initialDistance)
		controls.maxDistance = Math.max(initialDistance * 3, controls.minDistance)

		const render = (): void => {
			if (isDisposed) return
			renderer.render(scene, camera)
			onposechange(cameraPoseOf(camera, controls))
		}
		controls.addEventListener("change", render)
		// The textures decode after the first render, and the canvas renders
		// only when the view changes.
		loadingManager.onLoad = render
		controls.update()

		applyScene = (sceneSettings) => {
			camera.fov = sceneSettings.fieldOfViewDegrees
			camera.updateProjectionMatrix()
			keyLight.intensity = keyLightIntensity * sceneSettings.lightIntensity
			ambientLight.intensity = ambientLightIntensity * sceneSettings.brightness
			const lightAngleXRadians = MathUtils.degToRad(
					sceneSettings.lightAngleXDegrees,
				),
				lightAngleYRadians = MathUtils.degToRad(
					sceneSettings.lightAngleYDegrees,
				)
			// The light sits on a sphere around its target, which lies at the
			// model in front of the camera.
			keyLight.position.set(
				Math.sin(lightAngleXRadians) *
					Math.cos(lightAngleYRadians) *
					initialDistance,
				Math.sin(lightAngleYRadians) * initialDistance,
				(Math.cos(lightAngleXRadians) * Math.cos(lightAngleYRadians) - 1) *
					initialDistance,
			)
			render()
		}

		const resizeObserver = new ResizeObserver(([entry]) => {
			if (!entry) return
			const { width, height } = entry.contentRect
			if (width === 0 || height === 0) return
			renderer.setSize(width, height, false)
			camera.aspect = width / height
			camera.updateProjectionMatrix()
			render()
		})
		resizeObserver.observe(canvas)

		resetCameraView = () => {
			camera.position.copy(initialCameraPosition)
			controls.target.copy(center)
			controls.update()
		}
		const sceneCanvas = canvas
		captureSceneFrame = () => {
			render()
			return sceneCanvas.toDataURL("image/png")
		}
		const handleKeydown = (event: KeyboardEvent): void => {
			if (moveOrbitViewByKey(event, camera, controls, resetView)) {
				event.preventDefault()
			}
		}
		canvas.addEventListener("keydown", handleKeydown)

		return () => {
			isDisposed = true
			applyScene = resetCameraView = captureSceneFrame = undefined
			canvas?.removeEventListener("keydown", handleKeydown)
			resizeObserver.disconnect()
			controls.removeEventListener("change", render)
			controls.dispose()
			disposeModelObject(modelObject)
			renderer.dispose()
			// Each canvas keeps its WebGL context until the browser collects it,
			// and browsers allow only a few at a time.
			renderer.forceContextLoss()
		}
	})

	// The scene follows the camera and lighting controls without rebuilding.
	$effect(() => {
		const sceneSettings = { ...modelViewerScene }
		applyScene?.(sceneSettings)
	})

	// parseModelObject builds the meshes of the model. The MTL names each
	// texture by its file hash, which the loading manager resolves to the data
	// URL of the texture.
	function parseModelObject(
		renderedModel: RenderedModel,
		loadingManager: LoadingManager,
	): Object3D {
		loadingManager.setURLModifier((textureUrl) => {
			const fileHash = textureUrl.slice(textureUrl.lastIndexOf("/") + 1)
			return renderedModel.textureDataUrlByFileHash?.[fileHash] ?? textureUrl
		})
		const materials = new MTLLoader(loadingManager).parse(
			renderedModel.materials,
			"",
		)
		materials.preload()
		// Roblox exports an opacity map with every material, which would make the
		// whole model see-through.
		for (const material of Object.values(materials.materials)) {
			material.transparent = false
		}
		return new OBJLoader(loadingManager)
			.setMaterials(materials)
			.parse(renderedModel.mesh)
	}

	function disposeModelObject(modelObject: Object3D): void {
		modelObject.traverse((child) => {
			if (!(child instanceof Mesh)) return
			child.geometry.dispose()
			const childMaterials: Material[] = Array.isArray(child.material)
				? child.material
				: [child.material]
			for (const childMaterial of childMaterials) {
				for (const property of Object.values(childMaterial)) {
					if (property instanceof Texture) property.dispose()
				}
				childMaterial.dispose()
			}
		})
	}
</script>

<canvas
	bind:this={canvas}
	class="relative size-full animate-pop cursor-grab rounded-lg -outline-offset-2 focus-visible:outline-2 focus-visible:outline-focus active:cursor-grabbing"
	tabindex="0"
	aria-label={`3D view of ${label}`}
	aria-describedby={controlsDescriptionId}></canvas>
<span id={controlsDescriptionId} class="sr-only">{orbitViewKeysDescription}</span>
