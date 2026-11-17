<script lang="ts">
	import {
		AmbientLight,
		CanvasTexture,
		CylinderGeometry,
		DirectionalLight,
		Group,
		LoadingManager,
		MathUtils,
		Mesh,
		MeshBasicMaterial,
		OrthographicCamera,
		PerspectiveCamera,
		Scene,
		SphereGeometry,
		Sprite,
		SpriteMaterial,
		SRGBColorSpace,
		Texture,
		Vector2,
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

	// ModelFraming is where the camera starts for a model, and how near and far
	// it may move.
	interface ModelFraming {
		center: Vector3
		initialCameraPosition: Vector3
		minimumDistance: number
		maximumDistance: number
	}

	let canvas = $state<HTMLCanvasElement>()
	// applyScene changes the camera and the lights of the built scene.
	let applyScene: ((sceneSettings: RenderedModelScene) => void) | undefined
	// showModel puts a model in the built scene in place of the shown one.
	let showModel: ((nextModel: RenderedModel) => void) | undefined

	const controlsDescriptionId = $props.id(),
		axisIndicatorSizePx = 56,
		axisIndicatorMarginPx = 4,
		turnStepRadians = Math.PI / 12,
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
		// The axis indicator renders over the model in a second pass, so the
		// canvas clears once per frame instead of before each pass.
		renderer.autoClear = false
		let isDisposed = false,
			requestedModel: RenderedModel | undefined,
			// The model that waits for its textures before it replaces the shown
			// one.
			pendingModelObject: Object3D | undefined,
			shownModelObject: Object3D | undefined,
			shownModelFraming: ModelFraming | undefined
		const initialScene = untrack(() => ({ ...renderedModelScene })),
			scene = new Scene(),
			camera = new PerspectiveCamera(
				initialScene.fieldOfViewDegrees,
				1,
				0.1,
				1000,
			),
			keyLight = new DirectionalLight(0xffffff, keyLightIntensity),
			ambientLight = new AmbientLight(0xffffff, ambientLightIntensity)
		scene.add(ambientLight)
		// The light and its target move with the camera, so that the light angle
		// stays relative to the view. A directional light shines from its
		// position toward its target, so only their direction matters.
		keyLight.target.position.set(0, 0, -1)
		camera.add(keyLight, keyLight.target)
		scene.add(camera)

		const controls = new OrbitControls(camera, canvas)
		// OrbitControls moves the view while Control, Command, or Shift is held
		// during a drag, and on a right drag.
		controls.enablePan = true

		const axisIndicator = createAxisIndicator(getComputedStyle(canvas).fontFamily),
			canvasSize = new Vector2()
		const render = (): void => {
			if (isDisposed || !shownModelObject) return
			renderer.clear()
			renderer.render(scene, camera)
			// The indicator turns opposite to the camera, so that each axis
			// points where the axis of the model points on screen.
			axisIndicator.axes.quaternion.copy(camera.quaternion).invert()
			fadeEdgeOnAxisLabels(axisIndicator.axes, axisIndicator.labels)
			renderer.getSize(canvasSize)
			renderer.clearDepth()
			renderer.setViewport(
				axisIndicatorMarginPx,
				axisIndicatorMarginPx,
				axisIndicatorSizePx,
				axisIndicatorSizePx,
			)
			renderer.render(axisIndicator.scene, axisIndicator.camera)
			renderer.setViewport(0, 0, canvasSize.x, canvasSize.y)
		}
		controls.addEventListener("change", render)

		// placeModelObject puts the model in the scene in place of the shown one.
		// The first model starts at the camera that Roblox chose. A later model
		// keeps the view and follows the center of the model, because a page
		// replaces a model only with another look of the same subject, such as
		// the avatar after a change.
		const placeModelObject = (
			modelObject: Object3D,
			modelFraming: ModelFraming,
		): void => {
			if (shownModelObject) {
				scene.remove(shownModelObject)
				disposeModelObject(shownModelObject)
			}
			scene.add(modelObject)
			shownModelObject = modelObject
			if (shownModelFraming) {
				const centerShift = modelFraming.center
					.clone()
					.sub(shownModelFraming.center)
				camera.position.add(centerShift)
				controls.target.add(centerShift)
			} else {
				camera.position.copy(modelFraming.initialCameraPosition)
				controls.target.copy(modelFraming.center)
			}
			shownModelFraming = modelFraming
			controls.minDistance = modelFraming.minimumDistance
			controls.maxDistance = modelFraming.maximumDistance
			controls.update()
			render()
		}

		// The model replaces the shown one once its textures decode, so that
		// the view never shows it without them. A model without textures
		// starts no load, so it replaces the shown one at once.
		showModel = (nextModel) => {
			if (nextModel === requestedModel) return
			requestedModel = nextModel
			if (pendingModelObject) disposeModelObject(pendingModelObject)
			const loadingManager = new LoadingManager()
			let hasTextures = false
			loadingManager.onStart = () => {
				hasTextures = true
			}
			const modelObject = parseModelObject(nextModel, loadingManager),
				modelFraming = framingOf(nextModel, camera.fov),
				placeDecodedModelObject = (): void => {
					if (isDisposed || pendingModelObject !== modelObject) return
					pendingModelObject = undefined
					placeModelObject(modelObject, modelFraming)
				}
			pendingModelObject = modelObject
			loadingManager.onLoad = placeDecodedModelObject
			if (!hasTextures) placeDecodedModelObject()
		}
		showModel(untrack(() => model))

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
			// The light sits on a sphere around its target, which lies in front
			// of the camera.
			keyLight.position.set(
				Math.sin(lightAngleXRadians) * Math.cos(lightAngleYRadians),
				Math.sin(lightAngleYRadians),
				Math.cos(lightAngleXRadians) * Math.cos(lightAngleYRadians) - 1,
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
					if (!shownModelFraming) return
					camera.position.copy(shownModelFraming.initialCameraPosition)
					controls.target.copy(shownModelFraming.center)
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
			applyScene = showModel = undefined
			canvas?.removeEventListener("keydown", handleKeydown)
			resizeObserver.disconnect()
			controls.removeEventListener("change", render)
			controls.dispose()
			if (pendingModelObject) disposeModelObject(pendingModelObject)
			if (shownModelObject) disposeModelObject(shownModelObject)
			disposeAxisIndicator(axisIndicator.axes)
			renderer.dispose()
			// Each canvas keeps its WebGL context until the browser collects it,
			// and browsers allow only a few at a time.
			renderer.forceContextLoss()
		}
	})

	// A model that replaces the shown one goes into the built scene.
	$effect(() => {
		const nextModel = model
		untrack(() => showModel?.(nextModel))
	})

	// The scene follows the camera and lighting controls without rebuilding.
	$effect(() => {
		const sceneSettings = { ...renderedModelScene }
		applyScene?.(sceneSettings)
	})

	// framingOf places the camera where Roblox placed it for the model. Roblox
	// places the camera for its own field of view, so the camera moves to keep
	// the model framed at the viewer's.
	function framingOf(
		renderedModel: RenderedModel,
		fieldOfViewDegrees: number,
	): ModelFraming {
		const { minimum, maximum } = renderedModel.bounds,
			center = new Vector3(
				(minimum.x + maximum.x) / 2,
				(minimum.y + maximum.y) / 2,
				(minimum.z + maximum.z) / 2,
			),
			boundsSize = new Vector3(
				maximum.x - minimum.x,
				maximum.y - minimum.y,
				maximum.z - minimum.z,
			).length(),
			framingScale =
				Math.tan(
					MathUtils.degToRad(renderedModel.camera.fieldOfViewDegrees) / 2,
				) / Math.tan(MathUtils.degToRad(fieldOfViewDegrees) / 2),
			initialCameraPosition = new Vector3(
				renderedModel.camera.position.x,
				renderedModel.camera.position.y,
				renderedModel.camera.position.z,
			)
				.sub(center)
				.multiplyScalar(framingScale)
				.add(center),
			initialDistance = initialCameraPosition.distanceTo(center),
			minimumDistance = Math.min(boundsSize * 0.3, initialDistance)
		return {
			center,
			initialCameraPosition,
			minimumDistance,
			maximumDistance: Math.max(initialDistance * 3, minimumDistance),
		}
	}

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

	// createAxisIndicator builds the X, Y, and Z axes of the model as short
	// rods with labels, colored red, green, and blue as in 3D tools. A ball
	// marks the positive end and a faint, shorter rod the negative end, so
	// that an axis that points at the camera still shows as a dot.
	function createAxisIndicator(labelFontFamily: string): {
		scene: Scene
		camera: OrthographicCamera
		axes: Group
		labels: { sprite: Sprite; direction: Vector3 }[]
	} {
		const scene = new Scene(),
			camera = new OrthographicCamera(-1.5, 1.5, 1.5, -1.5, 0, 4),
			axes = new Group(),
			labels: { sprite: Sprite; direction: Vector3 }[] = [],
			cylinderAxis = new Vector3(0, 1, 0)
		camera.position.set(0, 0, 2)
		scene.add(axes)
		for (const axis of [
			{ name: "X", color: "#e5484d", direction: new Vector3(1, 0, 0) },
			{ name: "Y", color: "#46a758", direction: new Vector3(0, 1, 0) },
			{ name: "Z", color: "#3e8ae6", direction: new Vector3(0, 0, 1) },
		]) {
			// A cylinder stands along the Y axis until it turns onto its axis.
			const rod = new Mesh(
					new CylinderGeometry(0.05, 0.05, 1, 8),
					new MeshBasicMaterial({ color: axis.color }),
				),
				negativeRod = new Mesh(
					new CylinderGeometry(0.035, 0.035, 0.6, 8),
					new MeshBasicMaterial({
						color: axis.color,
						transparent: true,
						opacity: 0.35,
					}),
				),
				tip = new Mesh(
					new SphereGeometry(0.11, 12, 8),
					new MeshBasicMaterial({ color: axis.color }),
				),
				axisLabel = new Sprite(
					new SpriteMaterial({
						map: createAxisLabelTexture(
							axis.name,
							axis.color,
							labelFontFamily,
						),
						depthTest: false,
						transparent: true,
					}),
				)
			rod.quaternion.setFromUnitVectors(cylinderAxis, axis.direction)
			rod.position.copy(axis.direction).multiplyScalar(0.5)
			negativeRod.quaternion.copy(rod.quaternion)
			negativeRod.position.copy(axis.direction).multiplyScalar(-0.3)
			tip.position.copy(axis.direction)
			axisLabel.position.copy(axis.direction).multiplyScalar(1.35)
			axisLabel.scale.setScalar(0.7)
			axes.add(rod, negativeRod, tip, axisLabel)
			labels.push({ sprite: axisLabel, direction: axis.direction })
		}
		return { scene, camera, axes, labels }
	}

	// fadeEdgeOnAxisLabels hides the label of an axis that points at or away
	// from the camera, because it would sit on the center of the indicator.
	function fadeEdgeOnAxisLabels(
		axes: Group,
		labels: { sprite: Sprite; direction: Vector3 }[],
	): void {
		const screenDirection = new Vector3()
		for (const { sprite, direction } of labels) {
			screenDirection.copy(direction).applyQuaternion(axes.quaternion)
			sprite.material.opacity = MathUtils.smoothstep(
				Math.hypot(screenDirection.x, screenDirection.y),
				0.2,
				0.5,
			)
		}
	}

	function createAxisLabelTexture(
		axisName: string,
		color: string,
		fontFamily: string,
	): CanvasTexture {
		const labelCanvas = document.createElement("canvas")
		labelCanvas.width = labelCanvas.height = 64
		const context = labelCanvas.getContext("2d")
		if (context) {
			context.font = `600 44px ${fontFamily}`
			context.fillStyle = color
			context.textAlign = "center"
			context.textBaseline = "middle"
			context.fillText(axisName, 32, 34)
		}
		const texture = new CanvasTexture(labelCanvas)
		texture.colorSpace = SRGBColorSpace
		return texture
	}

	// disposeAxisIndicator leaves the geometry of the labels, which three.js
	// shares among every sprite.
	function disposeAxisIndicator(axes: Group): void {
		for (const child of axes.children) {
			if (child instanceof Mesh) {
				child.geometry.dispose()
				;(child.material as Material).dispose()
			} else if (child instanceof Sprite) {
				child.material.map?.dispose()
				child.material.dispose()
			}
		}
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
<span id={controlsDescriptionId} class="sr-only"
	>Drag or use the arrow keys to turn it. Hold Control while you drag or press the
	arrow keys to move it. Scroll or use plus and minus to zoom. Home resets the view.</span>
