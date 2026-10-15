<script lang="ts">
	import { onDestroy } from "svelte"
	import { installPointerResize } from "../shared/pointer-resize"

	let {
		width = $bindable(),
		collapsed,
		name,
	}: {
		width: number
		collapsed: boolean
		name: string
	} = $props()
	const minimumWidth = 220
	let stopResize: (() => void) | undefined,
		viewportWidth = $state(window.innerWidth)
	const maximumWidth = $derived(
		Math.max(minimumWidth, Math.min(420, viewportWidth - 500)),
	)

	function resize(nextWidth: number): void {
		width = Math.min(maximumWidth, Math.max(minimumWidth, nextWidth))
	}

	function startResize(event: PointerEvent): void {
		if (
			collapsed ||
			event.button !== 0 ||
			!(event.currentTarget instanceof HTMLElement)
		) {
			return
		}
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
		if (collapsed || (event.key !== "ArrowLeft" && event.key !== "ArrowRight"))
			return
		event.preventDefault()
		resize(width + (event.key === "ArrowLeft" ? -16 : 16))
	}

	onDestroy(() => stopResize?.())
</script>

<svelte:window bind:innerWidth={viewportWidth} />

<div
	class:disabled={collapsed}
	class="sidebar-resizer"
	role="slider"
	tabindex={collapsed ? -1 : 0}
	aria-disabled={collapsed}
	aria-label={`Resize ${name} sidebar`}
	aria-orientation="vertical"
	aria-valuenow={width}
	aria-valuemin={minimumWidth}
	aria-valuemax={maximumWidth}
	data-tooltip={`Drag to resize the ${name} sidebar`}
	data-tooltip-side="right"
	onpointerdown={startResize}
	onkeydown={handleKeydown}>
</div>
