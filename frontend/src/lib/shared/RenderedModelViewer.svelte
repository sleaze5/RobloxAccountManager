<script lang="ts">
	import {
		AmbientLight,
		DirectionalLight,
		LoadingManager,
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
	import type { RenderedModel } from "../backend/bridge"

	// label names what the model shows, such as "@builderman's avatar".
	let {
		model,
		label,
		onerror,
	}: { model: RenderedModel; label: string; onerror: () => void } = $props()

	let canvas = $state<HTMLCanvasElement>()

	const turnStepRadians = Math.PI / 12,
		zoomStepFactor = 1.15,
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
		const loadingManager = new LoadingManager(),
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
				model.camera.fieldOfViewDegrees,
				1,
				0.1,
				1000,
			),
			initialCameraPosition = new Vector3(
				model.camera.position.x,
				model.camera.position.y,
				model.camera.position.z,
			),
			cameraLight = new DirectionalLight(0xffffff, 1.6)
		scene.add(modelObject)
		scene.add(new AmbientLight(0xffffff, 1.4))
		cameraLight.position.set(1, 1, 2)
		camera.add(cameraLight)
		scene.add(camera)
		camera.position.copy(initialCameraPosition)

		const controls = new OrbitControls(camera, canvas),
			initialDistance = initialCameraPosition.distanceTo(center)
		controls.target.copy(center)
		controls.enablePan = false
		controls.minDistance = Math.min(boundsSize * 0.6, initialDistance)
		controls.maxDistance = Math.max(initialDistance * 2, controls.minDistance)

		const render = (): void => {
			if (!isDisposed) renderer.render(scene, camera)
		}
		controls.addEventListener("change", render)
		// The textures decode after the first render, and the canvas renders
		// only when the view changes.
		loadingManager.onLoad = render
		controls.update()

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
		const rightAxis = (): Vector3 =>
			new Vector3().setFromMatrixColumn(camera.matrixWorld, 0).normalize()
		const handleKeydown = (event: KeyboardEvent): void => {
			switch (event.key) {
				case "ArrowLeft":
					turnCamera(upAxis, -turnStepRadians)
					break
				case "ArrowRight":
					turnCamera(upAxis, turnStepRadians)
					break
				case "ArrowUp":
					turnCamera(rightAxis(), -turnStepRadians)
					break
				case "ArrowDown":
					turnCamera(rightAxis(), turnStepRadians)
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
	aria-label={`3D view of ${label}. Drag or use the arrow keys to turn it, and scroll or use plus and minus to zoom.`}
	data-tooltip="Drag to turn, scroll to zoom"></canvas>
