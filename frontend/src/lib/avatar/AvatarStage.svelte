<script lang="ts">
	import arrowsOutIcon from "@phosphor-icons/core/regular/arrows-out.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import cubeIcon from "@phosphor-icons/core/regular/cube.svg?raw"
	import paintBrushIcon from "@phosphor-icons/core/regular/paint-brush.svg?raw"
	import personArmsSpreadIcon from "@phosphor-icons/core/regular/person-arms-spread.svg?raw"
	import ExpandedImageDialog from "../shared/ExpandedImageDialog.svelte"
	import FrameUnavailable from "../shared/FrameUnavailable.svelte"
	import Icon from "../shared/Icon.svelte"
	import ImageViewToggle from "../shared/ImageViewToggle.svelte"
	import type { ImageView } from "../shared/ImageViewToggle.svelte"
	import type { AvatarState } from "./avatar-state.svelte"

	let { avatar, username }: { avatar: AvatarState; username: string } = $props()

	let failedImageUrl = $state(""),
		view = $state<ImageView>("2D"),
		isExpanded = $state(false),
		isLiveViewLoading = $state(false)

	// Roblox needs a moment between draft changes, so a preview renders only
	// once the draft stops changing.
	const previewDelayMs = 600,
		label = $derived(`@${username}'s avatar`),
		imageUrl = $derived(avatar.avatar?.imageUrl ?? ""),
		shownImageUrl = $derived(
			avatar.isChanged && avatar.previewImageUrl
				? avatar.previewImageUrl
				: imageUrl,
		),
		savedType = $derived(avatar.avatar?.avatarType ?? ""),
		draftType = $derived(avatar.draft?.avatarType ?? ""),
		// The 3D view builds the draft itself, so it always shows the draft.
		shownType = $derived(
			view === "3D" || avatar.isPreviewCurrent ? draftType : savedType,
		),
		// The 3D avatar loads only when the 3D view opens, because its renderer
		// is large.
		liveViewImport = $derived(
			view === "3D" ? import("./AvatarLiveView.svelte") : undefined,
		),
		busyLabel = $derived(
			view === "3D"
				? isLiveViewLoading
					? "Loading…"
					: ""
				: avatar.isRenderingImage
					? "Rendering…"
					: avatar.isRenderingPreview
						? "Previewing…"
						: "",
		),
		canRedraw = $derived(
			!!avatar.avatar &&
				!avatar.loading &&
				!avatar.isChangingAvatar &&
				!avatar.isRedrawingImage &&
				!avatar.isRenderingImage,
		)

	// The 2D view previews the unsaved draft.
	$effect(() => {
		const shouldPreview =
			view === "2D" && avatar.isEditable && avatar.isChanged && !!avatar.draft
		if (!shouldPreview) return
		const previewTimer = setTimeout(
			() => void avatar.renderPreview(),
			previewDelayMs,
		)
		return () => clearTimeout(previewTimer)
	})
</script>

{#snippet imageFrame()}
	{#if liveViewImport && avatar.draft}
		{#await liveViewImport then { default: AvatarLiveView }}
			<AvatarLiveView
				accountId={avatar.accountId}
				draft={avatar.draft}
				{label}
				bind:isLoading={isLiveViewLoading} />
		{:catch}
			<FrameUnavailable icon={cubeIcon} message="The 3D viewer couldn't start." />
		{/await}
	{:else if view === "3D"}
		<span class="absolute inset-3 skeleton rounded-md" role="status">
			<span class="sr-only">Loading the 3D avatar</span>
		</span>
	{:else if shownImageUrl && failedImageUrl !== shownImageUrl}
		{#key shownImageUrl}
			<img
				class="relative size-full animate-pop object-contain"
				src={shownImageUrl}
				alt={label}
				decoding="async"
				draggable="false"
				referrerpolicy="no-referrer"
				onerror={() => (failedImageUrl = shownImageUrl)} />
		{/key}
	{:else if avatar.avatar}
		<Icon
			svg={personArmsSpreadIcon}
			class="relative size-1/3 text-fg-faint"
			aria-label="The avatar image is unavailable" />
	{:else}
		<span class="absolute inset-3 skeleton rounded-md" role="status">
			<span class="sr-only">Loading the avatar image</span>
		</span>
	{/if}
	{#if shownType}
		<span
			class="pointer-events-none absolute top-2 left-2 rounded-sm border border-line-strong bg-overlay px-1 font-mono text-2xs leading-4 font-medium text-fg-muted"
			>{shownType}</span>
	{/if}
	{#if busyLabel}
		<span
			class="pointer-events-none absolute right-2 bottom-2 flex animate-enter items-center gap-1 rounded-sm border border-line-strong bg-overlay px-1.5 text-2xs leading-5 font-medium text-fg"
			role="status">
			<Icon svg={circleNotchIcon} class="spinner" size={12} aria-hidden="true" />
			{busyLabel}
		</span>
	{/if}
{/snippet}

{#snippet imageControls(isExpandedView: boolean)}
	<ImageViewToggle bind:view />
	{#if !isExpandedView}
		<button
			class="button-icon"
			type="button"
			aria-label="Expand avatar image"
			data-tooltip="Expand"
			disabled={!avatar.avatar}
			onclick={() => (isExpanded = true)}>
			<Icon svg={arrowsOutIcon} size={16} aria-hidden="true" />
		</button>
	{/if}
	{#if avatar.isEditable}
		<button
			class="ml-auto button"
			type="button"
			aria-busy={avatar.isRedrawingImage}
			disabled={!canRedraw}
			data-tooltip="Ask Roblox to render the avatar images again"
			onclick={() => void avatar.redrawImage()}>
			<Icon svg={paintBrushIcon} size={16} aria-hidden="true" />
			{avatar.isRedrawingImage ? "Redrawing…" : "Redraw"}
		</button>
	{/if}
{/snippet}

<section
	class="flex w-80 shrink-0 flex-col gap-3 overflow-y-auto border-r border-line-subtle bg-surface p-4"
	aria-label="Avatar image">
	<figure
		class="relative grid aspect-square w-full place-items-center overflow-hidden rounded-lg border border-line-subtle bg-canvas">
		{@render imageFrame()}
	</figure>

	<div class="flex items-center gap-2">
		{@render imageControls(false)}
	</div>

	{#if avatar.isEditable && avatar.isChanged}
		<p class="animate-enter notice-note" role="status">
			{#if view === "3D"}
				The model shows your changes. Save to keep them.
			{:else if avatar.isPreviewCurrent}
				The image is a preview. Save to keep your changes.
			{:else if avatar.previewError}
				The image shows the saved avatar. {avatar.previewError}
			{:else}
				Updating the preview…
			{/if}
		</p>
	{/if}
</section>

{#if isExpanded}
	<ExpandedImageDialog {label} onClose={() => (isExpanded = false)}>
		{#snippet frame()}
			{@render imageFrame()}
		{/snippet}
		{#snippet controls()}
			{@render imageControls(true)}
		{/snippet}
	</ExpandedImageDialog>
{/if}
