<script lang="ts">
	import { onDestroy } from "svelte"
	import { liftToTopLayer, tooltipIn, tooltipOut } from "./presence"

	const gap = 6,
		margin = 8,
		openDelayMs = 500,
		skipDelayMs = 300,
		// Text longer than this wraps inside a fixed width instead of running
		// past the window, as a long badge or game description would.
		maxSingleLineLength = 48

	type Side = "top" | "bottom" | "left" | "right"
	interface Placement {
		side: Side
		alignEnd: boolean
	}

	let text = $state(""),
		strong = $state(""),
		left = $state(0),
		top = $state(0),
		visible = $state(false),
		instant = $state(false),
		alignEnd = false,
		anchor: HTMLElement | null = null,
		anchorTextObserver: MutationObserver | null = null,
		frame = 0,
		ignoreFocusUntil = 0,
		skipDelayUntil = 0,
		openTimer = 0,
		requestedSide: Side = "bottom",
		tip: HTMLDivElement | undefined = $state()
	const strongAt = $derived(strong ? text.indexOf(strong) : -1),
		isWrapping = $derived(text.includes("\n") || text.length > maxSingleLineLength)

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
		strong = element.dataset.tooltipStrong?.trim() ?? ""
		// A countdown changes its tooltip text while the tooltip is open.
		anchorTextObserver = new MutationObserver(updateAnchorText)
		anchorTextObserver.observe(element, {
			attributeFilter: ["data-tooltip", "data-tooltip-strong"],
		})
		instant = performance.now() < skipDelayUntil
		const delayMs = instant ? 0 : openDelayMs
		openTimer = window.setTimeout(() => {
			openTimer = 0
			if (!anchor) {
				return
			}

			visible = true
			cancelAnimationFrame(frame)
			frame = requestAnimationFrame(place)
		}, delayMs)
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

	function updateAnchorText(): void {
		const nextText = anchor?.dataset.tooltip?.trim()
		if (!anchor || !nextText) {
			hide()
			return
		}
		text = nextText
		strong = anchor.dataset.tooltipStrong?.trim() ?? ""
		if (!visible) return
		cancelAnimationFrame(frame)
		frame = requestAnimationFrame(place)
	}

	function hide(): void {
		if (visible) skipDelayUntil = performance.now() + skipDelayMs
		anchorTextObserver?.disconnect()
		anchorTextObserver = null
		window.clearTimeout(openTimer)
		openTimer = 0
		cancelAnimationFrame(frame)
		visible = false
		anchor = null
		text = ""
		strong = ""
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
		skipDelayUntil = 0
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (event.key !== "Escape" || (!visible && openTimer === 0)) {
			return
		}

		if (visible) {
			event.stopPropagation()
		}
		hide()
	}

	onDestroy(hide)
</script>

{#if visible && text}
	<div
		class={[
			"pointer-events-none fixed top-0 left-0 rounded-md border border-line bg-raised px-2 py-1 text-2xs font-medium text-fg shadow-float",
			isWrapping
				? "max-w-xs leading-snug whitespace-pre-line"
				: "leading-none whitespace-nowrap",
		]}
		role="tooltip"
		bind:this={tip}
		use:liftToTopLayer
		style="top: 0; left: 0; transform: translate({left}px, {top}px);">
		<span class="block" in:tooltipIn={{ instant }} out:tooltipOut
			>{#if strongAt < 0}{text}{:else}{text.slice(0, strongAt)}<strong
					class="font-semibold">{strong}</strong
				>{text.slice(strongAt + strong.length)}{/if}</span>
	</div>
{/if}

<svelte:document
	onpointerover={schedule}
	onfocusin={schedule}
	onpointerout={leave}
	onfocusout={hide}
	onpointerdown={handlePointerDown}
	onkeydowncapture={handleKeydown}
	onscrollcapture={hide} />
