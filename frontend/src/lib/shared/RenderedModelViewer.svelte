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
	import { renderedModelScene } from "./rendered-model-scene.svelte"
	import type { RenderedModelScene } from "./rendered-model-scene.svelte"

	// label names what the model shows, such as "@builderman's avatar".
	let {
		model,
		label,
		onerror,
	}: { model: RenderedModel; label: string; onerror: () => void } = $props()

	let canvas = $state<HTMLCanvasElement>()
	// applyScene changes the camera and the lights of the built scene.
	let applyScene: ((sceneSettings: RenderedModelScene) => void) | undefined

	const turnStepRadians = Math.PI / 12,
		zoomStepFactor = 1.15,
		panStepShare = 0.05,
		keyLightIntensity = 1.6,
		ambientLightIntensity = 1.4,
		upAxis = new Vector3(0, 1, 0)

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
		const initialScene = untrack(() => ({ ...renderedModelScene })),
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
			framingScale =
				Math.tan(MathUtils.degToRad(model.camera.fieldOfViewDegrees) / 2) /
				Math.tan(MathUtils.degToRad(initialScene.fieldOfViewDegrees) / 2),
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
			if (!isDisposed) renderer.render(scene, camera)
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
			const lightAngleRadians = MathUtils.degToRad(
				sceneSettings.lightAngleDegrees,
			)
			keyLight.position.set(
				Math.sin(lightAngleRadians) * initialDistance,
				initialDistance / 2,
				(Math.cos(lightAngleRadians) - 1) * initialDistance,
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

		const cameraAxis = (column: number): Vector3 =>
			new Vector3().setFromMatrixColumn(camera.matrixWorld, column).normalize()
		const turnCamera = (axis: Vector3, angle: number): void => {
			const offset = camera.position.clone().sub(controls.target)
			offset.applyAxisAngle(axis, angle)
			camera.position.copy(controls.target).add(offset)
			controls.update()
		}
		const zoomCamera = (factor: number): void => {
			const offset = camera.position.clone().sub(controls.target)
			offset.multiplyScalar(factor)
			camera.position.copy(controls.target).add(offset)
			controls.update()
		}
		const panCamera = (rightSteps: number, upSteps: number): void => {
			const step = camera.position.distanceTo(controls.target) * panStepShare,
				offset = cameraAxis(0)
					.multiplyScalar(rightSteps * step)
					.add(cameraAxis(1).multiplyScalar(upSteps * step))
			camera.position.add(offset)
			controls.target.add(offset)
			controls.update()
		}
		const handleKeydown = (event: KeyboardEvent): void => {
			const isPanning = event.ctrlKey || event.metaKey
			switch (event.key) {
				case "ArrowLeft":
					if (isPanning) panCamera(1, 0)
					else turnCamera(upAxis, -turnStepRadians)
					break
				case "ArrowRight":
					if (isPanning) panCamera(-1, 0)
					else turnCamera(upAxis, turnStepRadians)
					break
				case "ArrowUp":
					if (isPanning) panCamera(0, -1)
					else turnCamera(cameraAxis(0), -turnStepRadians)
					break
				case "ArrowDown":
					if (isPanning) panCamera(0, 1)
					else turnCamera(cameraAxis(0), turnStepRadians)
					break
				case "+":
				case "=":
					zoomCamera(1 / zoomStepFactor)
					break
				case "-":
					zoomCamera(zoomStepFactor)
					break
				case "Home":
					camera.position.copy(initialCameraPosition)
					controls.target.copy(center)
					controls.update()
					break
				default:
					return
			}
			event.preventDefault()
		}
		canvas.addEventListener("keydown", handleKeydown)

		return () => {
			isDisposed = true
			applyScene = undefined
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
		const sceneSettings = { ...renderedModelScene }
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
	class="relative size-full animate-pop cursor-grab rounded-lg outline-offset-2 focus-visible:outline-2 focus-visible:outline-focus active:cursor-grabbing"
	tabindex="0"
	aria-label={`3D view of ${label}. Drag or use the arrow keys to turn it, hold Control while you drag or press the arrow keys to move it, and scroll or use plus and minus to zoom. Home resets the view.`}
	data-tooltip="Drag to turn, Ctrl+drag to move, scroll to zoom"></canvas>
