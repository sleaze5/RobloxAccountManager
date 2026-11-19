<script lang="ts">
	import arrowsOutIcon from "@phosphor-icons/core/regular/arrows-out.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import paintBrushIcon from "@phosphor-icons/core/regular/paint-brush.svg?raw"
	import personArmsSpreadIcon from "@phosphor-icons/core/regular/person-arms-spread.svg?raw"
	import ExpandedImageDialog from "../shared/ExpandedImageDialog.svelte"
	import Icon from "../shared/Icon.svelte"
	import ImageViewToggle from "../shared/ImageViewToggle.svelte"
	import type { ImageView } from "../shared/ImageViewToggle.svelte"
	import { AvatarType } from "../backend/bridge"
	import type { InventoryItem } from "../backend/bridge"
	import type { AvatarState } from "./avatar-state.svelte"
	import AnimationControls from "../model-viewer/AnimationControls.svelte"
	import ModelFrame from "../model-viewer/ModelFrame.svelte"
	import type {
		AnimationLibrary,
		AnimationPlayback,
		ModelSubject,
		ThumbnailFraming,
	} from "../model-viewer/model-subject"

	// thumbnailAnimation is the pose of the full-body image, which the 3D view
	// starts playing. While thumbnailFraming is set, such as on the Thumbnails
	// tab, the 3D view previews the profile image of previewedThumbnailImage
	// as Roblox frames it.
	let {
		avatar,
		username,
		library,
		thumbnailAnimation,
		thumbnailFraming,
		previewedThumbnailImage = $bindable(),
	}: {
		avatar: AvatarState
		username: string
		library: AnimationLibrary
		thumbnailAnimation: InventoryItem | undefined
		thumbnailFraming: ThumbnailFraming | undefined
		previewedThumbnailImage: ThumbnailFraming["image"]
	} = $props()

	const thumbnailImageOptions: { image: ThumbnailFraming["image"]; label: string }[] =
		[
			{ image: "headshot", label: "Headshot" },
			{ image: "fullBody", label: "Full body" },
		]

	let failedImageUrl = $state(""),
		view = $state<ImageView>("2D"),
		isExpanded = $state(false),
		isModelLoadingByView = $state({ page: false, expanded: false }),
		playbackByView = $state<
			Record<"page" | "expanded", AnimationPlayback | undefined>
		>({
			page: undefined,
			expanded: undefined,
		})

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
		isPreviewingThumbnail = $derived(!!thumbnailFraming),
		isThreeDimensional = $derived(view === "3D" || isPreviewingThumbnail),
		// Roblox poses only an R15 avatar with an emote.
		isPoseShown = $derived(avatar.draft?.avatarType === AvatarType.AvatarR15),
		shownFraming = $derived<ThumbnailFraming | undefined>(
			thumbnailFraming && !isPoseShown
				? { ...thumbnailFraming, poseAnimationId: undefined }
				: thumbnailFraming,
		),
		// The 3D view builds the draft itself, so it always shows the draft.
		shownType = $derived(
			isThreeDimensional || avatar.isPreviewCurrent ? draftType : savedType,
		),
		// The 3D view builds the draft, so that every change shows at once.
		draftSubject = $derived<ModelSubject | null>(
			avatar.draft
				? {
						kind: "avatar",
						look: avatar.draft,
						thumbnailAnimation: isPoseShown
							? thumbnailAnimation
							: undefined,
					}
				: null,
		),
		busyLabel = $derived(
			isThreeDimensional
				? isModelLoadingByView[isExpanded ? "expanded" : "page"]
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

{#snippet imageFrame(isExpandedView: boolean)}
	{#if view === "3D" || (isPreviewingThumbnail && !isExpandedView)}
		<ModelFrame
			accountId={avatar.accountId}
			subject={draftSubject}
			{label}
			adjustmentGuide={avatar.adjustmentGuide}
			thumbnailFraming={isExpandedView ? undefined : shownFraming}
			bind:isLoading={isModelLoadingByView[isExpandedView ? "expanded" : "page"]}
			bind:playback={playbackByView[isExpandedView ? "expanded" : "page"]} />
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
			aria-label="Avatar image unavailable" />
	{:else}
		<span class="absolute inset-3 skeleton rounded-md" role="status">
			<span class="sr-only">Loading avatar image…</span>
		</span>
	{/if}
	{#if shownType || busyLabel}
		<span class="frame-readout-top-left divide-x divide-line-subtle leading-5">
			{#if shownType}
				<span class="px-1.5 font-medium">{shownType}</span>
			{/if}
			{#if busyLabel}
				<span
					class="flex animate-enter items-center gap-1 px-1.5 text-fg"
					role="status">
					<Icon
						svg={circleNotchIcon}
						class="spinner"
						size={12}
						aria-hidden="true" />
					{busyLabel}
				</span>
			{/if}
		</span>
	{/if}
{/snippet}

{#snippet imageControls(isExpandedView: boolean)}
	{#if isPreviewingThumbnail && !isExpandedView}
		<div
			class="segmented w-44 shrink-0"
			role="group"
			aria-label="Previewed thumbnail">
			{#each thumbnailImageOptions as option (option.image)}
				<button
					type="button"
					aria-pressed={previewedThumbnailImage === option.image}
					onclick={() => (previewedThumbnailImage = option.image)}
					>{option.label}</button>
			{/each}
		</div>
	{:else}
		<ImageViewToggle bind:view />
	{/if}
	{#if !isExpandedView && !isPreviewingThumbnail}
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
	{#if isExpandedView && view === "3D"}
		<div class="min-w-0 flex-1">
			<AnimationControls playback={playbackByView.expanded} {library} isWide />
		</div>
	{/if}
{/snippet}

<section
	class="flex w-1/3 max-w-lg min-w-80 shrink-0 flex-col gap-3 overflow-y-auto border-r border-line-subtle bg-surface p-4"
	aria-label="Avatar image">
	<figure
		class="relative grid aspect-square w-full place-items-center overflow-hidden rounded-lg border border-line-subtle bg-canvas">
		{@render imageFrame(false)}
	</figure>

	<div class="flex items-center gap-2">
		{@render imageControls(false)}
	</div>

	{#if isPreviewingThumbnail}
		<p class="animate-enter text-xs text-fg-muted" role="status">
			{previewedThumbnailImage === "headshot"
				? "This previews the headshot as Roblox frames it. Change its settings to see them at once."
				: "This previews the full-body image as Roblox frames it. Change its settings to see them at once."}
		</p>
	{:else if view === "3D"}
		<div class="animate-enter">
			<AnimationControls playback={playbackByView.page} {library} />
		</div>
	{/if}

	{#if avatar.isEditable && avatar.isChanged}
		<div class="flex animate-enter flex-col gap-1 notice-note">
			<p role="status">
				{#if isThreeDimensional || avatar.isPreviewCurrent}
					This shows your changes. Save to keep them.
				{:else if avatar.previewError}
					The image shows the saved avatar. {avatar.previewError}
				{:else}
					Previewing…
				{/if}
			</p>
			<!-- Roblox renders the 2D image of a draft only after the draft stops
			changing, while the 3D view builds every change itself. -->
			{#if !isThreeDimensional}
				<p class="flex flex-wrap items-center gap-x-2">
					<span>The 3D view shows each change at once.</span>
					<button
						class="button-text"
						type="button"
						onclick={() => (view = "3D")}>Switch to 3D</button>
				</p>
			{/if}
		</div>
	{/if}
</section>

{#if isExpanded}
	<ExpandedImageDialog {label} onClose={() => (isExpanded = false)}>
		{#snippet frame()}
			{@render imageFrame(true)}
		{/snippet}
		{#snippet controls()}
			{@render imageControls(true)}
		{/snippet}
	</ExpandedImageDialog>
{/if}
