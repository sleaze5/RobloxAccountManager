<script lang="ts">
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import type { RenderedModel } from "../backend/bridge"
	import Dialog from "./Dialog.svelte"
	import Icon from "./Icon.svelte"
	import ImageViewToggle from "./ImageViewToggle.svelte"
	import type { ImageView } from "./ImageViewToggle.svelte"
	import RenderedModelView from "./RenderedModelView.svelte"

	// ExpandedImageDialog shows an image or a 3D model as large as the window
	// allows. label names what it shows, such as "@builderman's avatar".
	let {
		label,
		imageUrl,
		fallbackIcon,
		view = $bindable(),
		hasModel,
		model,
		modelError,
		onRetryModel,
		onClose,
	}: {
		label: string
		imageUrl: string
		fallbackIcon: string
		view: ImageView
		hasModel: boolean
		model: RenderedModel | null
		modelError: string
		onRetryModel: () => void
		onClose: () => void
	} = $props()

	let failedImageUrl = $state("")

	const titleId = $props.id()
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
	class="h-full max-w-5xl"
	hasFlexibleBody
	aria-labelledby={titleId}
	onBackdropClick={onClose}>
	<div class="dialog-heading">
		<div>
			<h2 id={titleId} class="wrap-anywhere">{label}</h2>
		</div>
		<button type="button" aria-label="Close expanded view" onclick={onClose}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>
	<div
		class="relative grid min-h-0 flex-1 place-items-center overflow-hidden rounded-lg border border-line-subtle bg-canvas">
		{#if hasModel && view === "3D"}
			<RenderedModelView
				{model}
				error={modelError}
				{label}
				onRetry={onRetryModel} />
		{:else if imageUrl && failedImageUrl !== imageUrl}
			{#key imageUrl}
				<img
					class="relative size-full animate-pop object-contain"
					src={imageUrl}
					alt={label}
					decoding="async"
					draggable="false"
					referrerpolicy="no-referrer"
					onerror={() => (failedImageUrl = imageUrl)} />
			{/key}
		{:else}
			<Icon
				svg={fallbackIcon}
				class="relative size-1/4 text-fg-faint"
				aria-label="The image is unavailable" />
		{/if}
	</div>
	{#if hasModel}
		<div class="mt-3 flex items-center">
			<ImageViewToggle bind:view />
		</div>
	{/if}
</Dialog>
