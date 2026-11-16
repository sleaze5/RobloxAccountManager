<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import paintBrushIcon from "@phosphor-icons/core/regular/paint-brush.svg?raw"
	import personArmsSpreadIcon from "@phosphor-icons/core/regular/person-arms-spread.svg?raw"
	import { untrack } from "svelte"
	import Icon from "../shared/Icon.svelte"
	import ImageViewToggle from "../shared/ImageViewToggle.svelte"
	import type { ImageView } from "../shared/ImageViewToggle.svelte"
	import RenderedModelView from "../shared/RenderedModelView.svelte"
	import type { AvatarState } from "./avatar-state.svelte"

	let { avatar, username }: { avatar: AvatarState; username: string } = $props()

	let failedImageUrl = $state(""),
		view = $state<ImageView>("2D")

	const imageUrl = $derived(avatar.avatar?.imageUrl ?? ""),
		savedType = $derived(avatar.avatar?.avatarType ?? ""),
		draftType = $derived(avatar.draft?.avatarType ?? ""),
		canRedraw = $derived(
			!!avatar.avatar &&
				!avatar.loading &&
				!avatar.isChangingAvatar &&
				!avatar.isRedrawingImage &&
				!avatar.isRenderingImage,
		)

	// The 3D avatar loads when the 3D view opens, and again when the full-body
	// image changes, because Roblox renders both from the same avatar.
	$effect(() => {
		const shownImageUrl = imageUrl,
			shouldLoadModel = view === "3D" && !!avatar.avatar
		untrack(() => {
			if (shouldLoadModel) void avatar.loadModel(shownImageUrl)
		})
	})
</script>

<section
	class="flex w-60 shrink-0 flex-col gap-3 overflow-y-auto border-r border-line-subtle bg-surface p-3"
	aria-label="Avatar image">
	<figure
		class="relative grid aspect-square w-full place-items-center overflow-hidden rounded-lg border border-line-subtle bg-canvas">
		{#if view === "3D"}
			<RenderedModelView
				model={avatar.model.model}
				error={avatar.model.error}
				label={`@${username}'s avatar`}
				onRetry={() => void avatar.loadModel(imageUrl)} />
		{:else if imageUrl && failedImageUrl !== imageUrl}
			{#key imageUrl}
				<img
					class="relative size-full animate-pop object-contain"
					src={imageUrl}
					alt={`@${username}'s avatar`}
					decoding="async"
					draggable="false"
					referrerpolicy="no-referrer"
					onerror={() => (failedImageUrl = imageUrl)} />
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
		{#if savedType}
			<span
				class="pointer-events-none absolute top-2 left-2 rounded-sm border border-line-strong bg-overlay px-1 font-mono text-2xs leading-4 font-medium text-fg-muted"
				>{savedType}</span>
		{/if}
		{#if avatar.isRenderingImage || (view === "3D" && avatar.model.isLoading && avatar.model.model)}
			<span
				class="pointer-events-none absolute right-2 bottom-2 flex animate-enter items-center gap-1 rounded-sm border border-line-strong bg-overlay px-1.5 text-2xs leading-5 font-medium text-fg"
				role="status">
				<Icon
					svg={circleNotchIcon}
					class="spinner"
					size={12}
					aria-hidden="true" />
				{avatar.isRenderingImage ? "Rendering…" : "Loading…"}
			</span>
		{/if}
	</figure>

	<div class="flex items-center gap-2">
		<ImageViewToggle bind:view />
		{#if avatar.isEditable}
			<button
				class="button min-w-0 flex-1 justify-center"
				type="button"
				aria-busy={avatar.isRedrawingImage}
				disabled={!canRedraw}
				data-tooltip="Ask Roblox to render the avatar images again"
				onclick={() => void avatar.redrawImage()}>
				<Icon svg={paintBrushIcon} size={16} aria-hidden="true" />
				{avatar.isRedrawingImage ? "Redrawing…" : "Redraw"}
			</button>
		{/if}
	</div>

	{#if avatar.isEditable && avatar.isChanged}
		<p class="animate-enter notice-note" role="status">
			The {view === "3D" ? "model" : "image"} shows the saved avatar{savedType !==
			draftType
				? ` as ${savedType}`
				: ""}. Save to see your changes.
		</p>
	{/if}
</section>
