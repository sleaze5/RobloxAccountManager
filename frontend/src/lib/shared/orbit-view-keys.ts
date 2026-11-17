import type { PerspectiveCamera, Vector3 } from "three"
import type { OrbitControls } from "three/addons/controls/OrbitControls.js"

const turnStepRadians = Math.PI / 12,
	zoomStepFactor = 1.15,
	panStepShare = 0.05

// orbitViewKeysDescription tells a screen reader how to move the view of a 3D
// viewer, for the element that receives the keys.
export const orbitViewKeysDescription =
	"Drag or use the arrow keys to turn it. Hold Control while you drag or press the arrow keys to move it. Scroll or use plus and minus to zoom. Home resets the view."

// moveOrbitViewByKey turns, moves, or zooms the view for the key, or calls
// resetView for Home, and reports whether the key changed the view. It builds
// every vector from a vector of the camera, so that it works with the copy of
// three.js that the avatar renderer bundles.
export function moveOrbitViewByKey(
	event: KeyboardEvent,
	camera: PerspectiveCamera,
	controls: OrbitControls,
	resetView: () => void,
): boolean {
	const cameraAxis = (column: number): Vector3 =>
		camera.position
			.clone()
			.setFromMatrixColumn(camera.matrixWorld, column)
			.normalize()
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
	const isPanning = event.ctrlKey || event.metaKey
	switch (event.key) {
		case "ArrowLeft":
			if (isPanning) panCamera(1, 0)
			else turnCamera(camera.up, -turnStepRadians)
			return true
		case "ArrowRight":
			if (isPanning) panCamera(-1, 0)
			else turnCamera(camera.up, turnStepRadians)
			return true
		case "ArrowUp":
			if (isPanning) panCamera(0, -1)
			else turnCamera(cameraAxis(0), -turnStepRadians)
			return true
		case "ArrowDown":
			if (isPanning) panCamera(0, 1)
			else turnCamera(cameraAxis(0), turnStepRadians)
			return true
		case "+":
		case "=":
			zoomCamera(1 / zoomStepFactor)
			return true
		case "-":
			zoomCamera(zoomStepFactor)
			return true
		case "Home":
			resetView()
			return true
		default:
			return false
	}
}
