<script lang="ts">
	import { onDestroy } from "svelte"
	import { tooltipIn, tooltipOut } from "./presence"

	const gap = 6,
		margin = 8,
		openDelayMs = 90

	type Side = "top" | "bottom" | "left" | "right"
	interface Placement {
		side: Side
		alignEnd: boolean
	}

	let text = $state(""),
		left = $state(0),
		top = $state(0),
		visible = $state(false),
		alignEnd = false,
		anchor: HTMLElement | null = null,
		frame = 0,
		ignoreFocusUntil = 0,
		openTimer = 0,
		requestedSide: Side = "bottom",
		tip: HTMLDivElement | undefined = $state()

	function placementOf(element: HTMLElement): Placement {
		switch (element.dataset.tooltipSide) {
			case "top":
				return { alignEnd: false, side: "top" }
			case "top-end":
				return { alignEnd: true, side: "top" }
			case "right":
				return { alignEnd: false, side: "right" }
			case "bottom-end":
				return { alignEnd: true, side: "bottom" }
			default:
				return { alignEnd: false, side: "bottom" }
		}
	}

	function schedule(event: Event): void {
		if (event.type === "focusin" && performance.now() < ignoreFocusUntil) {
			return
		}

		const { target } = event
		if (!(target instanceof Element)) {
			return
		}

		const element = target.closest<HTMLElement>("[data-tooltip]")
		if (!element || !element.dataset.tooltip?.trim()) {
			hide()
			return
		}

		if (element === anchor && (visible || openTimer !== 0)) {
			return
		}

		hide()
		const placement = placementOf(element)
		anchor = element
		requestedSide = placement.side
		alignEnd = placement.alignEnd
		text = element.dataset.tooltip.trim()
		openTimer = window.setTimeout(() => {
			openTimer = 0
			if (!anchor) {
				return
			}

			visible = true
			cancelAnimationFrame(frame)
			frame = requestAnimationFrame(place)
		}, openDelayMs)
	}

	function place(): void {
		if (!anchor || !tip) {
			return
		}

		const box = anchor.getBoundingClientRect(),
			tipBox = tip.getBoundingClientRect()
		let x: number, y: number

		if (requestedSide === "top") {
			x = alignEnd
				? box.right - tipBox.width
				: box.left + box.width / 2 - tipBox.width / 2
			y = box.top - gap - tipBox.height
			if (
				y < margin &&
				box.bottom + gap + tipBox.height <= window.innerHeight - margin
			) {
				y = box.bottom + gap
			}
		} else if (requestedSide === "right") {
			x = box.right + gap
			y = box.top + box.height / 2 - tipBox.height / 2
			if (
				x + tipBox.width > window.innerWidth - margin &&
				box.left - gap - tipBox.width >= margin
			) {
				x = box.left - gap - tipBox.width
			}
		} else {
			x = alignEnd
				? box.right - tipBox.width
				: box.left + box.width / 2 - tipBox.width / 2
			y = box.bottom + gap
			if (
				y + tipBox.height > window.innerHeight - margin &&
				box.top - gap - tipBox.height >= margin
			) {
				y = box.top - gap - tipBox.height
			}
		}

		x = Math.min(
			Math.max(x, margin),
			Math.max(margin, window.innerWidth - tipBox.width - margin),
		)
		y = Math.min(
			Math.max(y, margin),
			Math.max(margin, window.innerHeight - tipBox.height - margin),
		)
		left = x
		top = y
	}

	function hide(): void {
		window.clearTimeout(openTimer)
		openTimer = 0
		cancelAnimationFrame(frame)
		visible = false
		anchor = null
		text = ""
	}

	function leave(event: PointerEvent): void {
		if (!anchor) {
			return
		}

		const related = event.relatedTarget
		if (related instanceof Node && anchor.contains(related)) {
			return
		}

		hide()
	}

	function handlePointerDown(): void {
		ignoreFocusUntil = performance.now() + openDelayMs
		hide()
	}

	onDestroy(hide)
</script>

{#if visible && text}
	<div
		class="app-tooltip"
		class:app-tooltip-multiline={text.includes("\n")}
		role="tooltip"
		bind:this={tip}
		style="transform: translate({left}px, {top}px);">
		<span class="app-tooltip-inner" in:tooltipIn out:tooltipOut>{text}</span>
	</div>
{/if}

<svelte:document
	onpointerover={schedule}
	onfocusin={schedule}
	onpointerout={leave}
	onfocusout={hide}
	onpointerdown={handlePointerDown}
	onscrollcapture={hide} />
