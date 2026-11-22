<script lang="ts">
	import { onDestroy } from "svelte"
	import { installPointerResize } from "../shared/pointer-resize"

	// PropertyColumnResizer is the divider between the name and the value
	// columns of the properties, which drags, or moves with the arrow keys, to
	// resize the name column. It lies over the hairline that the rows draw.
	let {
		width = $bindable(),
		minimumWidth,
		maximumWidth,
	}: {
		width: number
		minimumWidth: number
		maximumWidth: number
	} = $props()

	let stopResize: (() => void) | undefined

	function resize(nextWidth: number): void {
		width = Math.round(Math.min(maximumWidth, Math.max(minimumWidth, nextWidth)))
	}

	function startResize(event: PointerEvent): void {
		if (event.button !== 0 || !(event.currentTarget instanceof HTMLElement)) return
		event.preventDefault()
		stopResize?.()
		const handle = event.currentTarget,
			{ pointerId } = event,
			startX = event.clientX,
			startWidth = width
		handle.setPointerCapture(pointerId)
		stopResize = installPointerResize(handle, pointerId, (move) =>
			resize(startWidth + move.clientX - startX),
		)
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return
		event.preventDefault()
		resize(width + (event.key === "ArrowLeft" ? -16 : 16))
	}

	onDestroy(() => stopResize?.())
</script>

<div
	class="group absolute inset-y-0 left-(--property-name-width) z-20 w-2 -translate-x-1/2 cursor-col-resize touch-none focus-visible:outline-0"
	role="slider"
	tabindex="0"
	aria-label="Resize the property name column"
	aria-orientation="vertical"
	aria-valuenow={width}
	aria-valuemin={minimumWidth}
	aria-valuemax={maximumWidth}
	data-tooltip="Drag to resize the name column"
	onpointerdown={startResize}
	onkeydown={handleKeydown}>
	<span
		class="mx-auto block h-full w-px transition-colors duration-100 group-hover:bg-line-focus group-focus-visible:bg-focus group-active:bg-line-focus"
	></span>
</div>
