export function installPointerResize(
	handle: HTMLElement,
	pointerId: number,
	resize: (event: PointerEvent) => void,
): () => void {
	const stop = () => {
		handle.removeEventListener("pointermove", resize)
		handle.removeEventListener("pointerup", stop)
		handle.removeEventListener("pointercancel", stop)
		handle.removeEventListener("lostpointercapture", stop)
		if (handle.hasPointerCapture(pointerId)) {
			handle.releasePointerCapture(pointerId)
		}
	}

	handle.addEventListener("pointermove", resize)
	handle.addEventListener("pointerup", stop)
	handle.addEventListener("pointercancel", stop)
	handle.addEventListener("lostpointercapture", stop)
	return stop
}
