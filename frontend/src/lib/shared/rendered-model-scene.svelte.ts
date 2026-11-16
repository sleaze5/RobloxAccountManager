// RenderedModelScene is the camera and the lights of the 3D viewers.
export interface RenderedModelScene {
	fieldOfViewDegrees: number
	// The strength of the light that fills every side of the model, as a share
	// of the default.
	brightness: number
	// The strength of the light that shines on the model, as a share of the
	// default.
	lightIntensity: number
	// Where the light shines from around the view: 0 from the camera, 90 from
	// the right, and 180 from behind the model.
	lightAngleXDegrees: number
	// How far above the view the light shines from: 90 from straight above,
	// and -90 from straight below.
	lightAngleYDegrees: number
}

export const defaultRenderedModelScene: Readonly<RenderedModelScene> = {
	fieldOfViewDegrees: 30,
	brightness: 2,
	lightIntensity: 1,
	lightAngleXDegrees: 30,
	lightAngleYDegrees: 25,
}

// renderedModelScene applies to every viewer for the session, so that a change
// carries over to the expanded view and to the next model.
export const renderedModelScene = $state<RenderedModelScene>({
	...defaultRenderedModelScene,
})

export function resetRenderedModelScene(): void {
	Object.assign(renderedModelScene, defaultRenderedModelScene)
}
