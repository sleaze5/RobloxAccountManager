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
	let stopResize: (() => void) | undefined

	function resize(nextWidth: number): void {
		const maximumWidth = Math.max(220, Math.min(420, window.innerWidth - 500))
		width = Math.min(maximumWidth, Math.max(220, nextWidth))
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
		if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return
		event.preventDefault()
		resize(width + (event.key === "ArrowLeft" ? -16 : 16))
	}

	onDestroy(() => stopResize?.())
</script>

<button
	class:disabled={collapsed}
	class="sidebar-resizer"
	type="button"
	disabled={collapsed}
	aria-label={`Resize ${name} sidebar`}
	data-tooltip={`Drag to resize the ${name} sidebar`}
	data-tooltip-side="right"
	onpointerdown={startResize}
	onkeydown={handleKeydown}></button>
