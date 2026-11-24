<script lang="ts">
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import type { Snippet } from "svelte"
	import Dialog from "./Dialog.svelte"
	import Icon from "./Icon.svelte"

	// ExpandedImageDialog shows what an image frame shows, such as an image or
	// a 3D model, as large as the window allows, with the controls of the
	// frame below it, if it has any. label names what it shows, such as
	// "@builderman's avatar". aspectRatio is the width of the frame divided by
	// its height, such as 16 / 9 for game media.
	let {
		label,
		frame,
		controls,
		aspectRatio = 1,
		onClose,
	}: {
		label: string
		frame: Snippet
		controls?: Snippet
		aspectRatio?: number
		onClose: () => void
	} = $props()

	let card = $state<HTMLDivElement>(),
		frameElement = $state<HTMLDivElement>(),
		frameWidth = $state(0)

	const titleId = $props.id()

	// The frame is the largest frame of its aspect ratio that fits the window
	// together with the rest of the dialog, so that the whole image or model
	// shows whichever side of the window is shorter.
	$effect(() => {
		const backdrop = card?.parentElement
		if (!card || !frameElement || !backdrop) return
		const fittedCard = card,
			fittedFrame = frameElement
		const fitFrame = (): void => {
			const backdropStyle = getComputedStyle(backdrop),
				availableWidth =
					backdrop.clientWidth -
					Number.parseFloat(backdropStyle.paddingLeft) -
					Number.parseFloat(backdropStyle.paddingRight),
				availableHeight =
					backdrop.clientHeight -
					Number.parseFloat(backdropStyle.paddingTop) -
					Number.parseFloat(backdropStyle.paddingBottom),
				surroundingWidth = fittedCard.offsetWidth - fittedFrame.offsetWidth,
				surroundingHeight = fittedCard.offsetHeight - fittedFrame.offsetHeight
			frameWidth = Math.max(
				0,
				Math.floor(
					Math.min(
						availableWidth - surroundingWidth,
						(availableHeight - surroundingHeight) * aspectRatio,
					),
				),
			)
		}
		const resizeObserver = new ResizeObserver(fitFrame)
		resizeObserver.observe(backdrop)
		resizeObserver.observe(fittedCard)
		return () => resizeObserver.disconnect()
	})
</script>

<!-- The dialog opens above another dialog, such as the item details, so it
handles Escape before the dialog below it. A panel inside it that closes on
Escape handles that Escape itself. -->
<svelte:window
	onkeydowncapture={(event) => {
		if (event.key !== "Escape") return
		if (
			event.target instanceof Element &&
			event.target.closest("[data-closes-on-escape]")
		) {
			return
		}
		event.preventDefault()
		event.stopImmediatePropagation()
		onClose()
	}} />

<Dialog
	class="w-auto max-w-full"
	aria-labelledby={titleId}
	bind:card
	onBackdropClick={onClose}>
	<!-- The heading and the controls take the width of the frame, so that a
	long name never widens the dialog. The name stays on one line, because the
	frame takes its size from the height that the heading leaves. -->
	<div class="dialog-heading w-0 min-w-full">
		<div class="min-w-0">
			<h2 id={titleId} class="truncate" data-tooltip={label}>{label}</h2>
		</div>
		<button type="button" aria-label="Close dialog" onclick={onClose}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>
	<div
		bind:this={frameElement}
		class="relative grid h-(--expanded-frame-height) w-(--expanded-frame-width) place-items-center overflow-hidden rounded-lg border border-line-subtle bg-canvas"
		style:--expanded-frame-width={`${frameWidth}px`}
		style:--expanded-frame-height={`${Math.floor(frameWidth / aspectRatio)}px`}>
		{@render frame()}
	</div>
	{#if controls}
		<div class="mt-3 flex w-0 min-w-full items-center gap-2">
			{@render controls()}
		</div>
	{/if}
</Dialog>
