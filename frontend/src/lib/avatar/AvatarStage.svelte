<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import cubeIcon from "@phosphor-icons/core/regular/cube.svg?raw"
	import paintBrushIcon from "@phosphor-icons/core/regular/paint-brush.svg?raw"
	import personArmsSpreadIcon from "@phosphor-icons/core/regular/person-arms-spread.svg?raw"
	import { untrack } from "svelte"
	import Icon from "../shared/Icon.svelte"
	import type { AvatarState } from "./avatar-state.svelte"

	type AvatarView = "2D" | "3D"

	let { avatar, username }: { avatar: AvatarState; username: string } = $props()

	let failedImageUrl = $state(""),
		view = $state<AvatarView>("2D"),
		isWebGLUnavailable = $state(false)

	const imageUrl = $derived(avatar.avatar?.imageUrl ?? ""),
		savedType = $derived(avatar.avatar?.avatarType ?? ""),
		draftType = $derived(avatar.draft?.avatarType ?? ""),
		isModelShown = $derived(view === "3D" && !isWebGLUnavailable),
		// three.js is large, so the viewer loads only when the 3D view opens.
		modelViewerImport = $derived(
			isModelShown ? import("./AvatarModelViewer.svelte") : undefined,
		),
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
			shouldLoadModel = isModelShown && !!avatar.avatar
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
			{#if isWebGLUnavailable || (avatar.modelError && !avatar.model)}
				<Icon
					svg={cubeIcon}
					class="relative size-1/3 text-fg-faint"
					aria-label="The 3D avatar is unavailable" />
			{:else if avatar.model && modelViewerImport}
				{#await modelViewerImport then { default: AvatarModelViewer }}
					{#key avatar.model}
						<AvatarModelViewer
							model={avatar.model}
							{username}
							onerror={() => (isWebGLUnavailable = true)} />
					{/key}
				{:catch}
					<Icon
						svg={cubeIcon}
						class="relative size-1/3 text-fg-faint"
						aria-label="The 3D avatar is unavailable" />
				{/await}
			{:else}
				<span class="absolute inset-3 skeleton rounded-md" role="status">
					<span class="sr-only">Loading the 3D avatar</span>
				</span>
			{/if}
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
		{#if avatar.isRenderingImage || (isModelShown && avatar.isModelLoading && avatar.model)}
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
		<div class="segmented w-24 shrink-0" role="group" aria-label="Avatar view">
			{#each ["2D", "3D"] as const as option (option)}
				<button
					type="button"
					aria-pressed={view === option}
					onclick={() => (view = option)}>{option}</button>
			{/each}
		</div>
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

	{#if view === "3D"}
		{#if isWebGLUnavailable}
			<p class="animate-enter notice-error" role="alert">
				The 3D view needs graphics acceleration, which isn't available on this
				computer.
			</p>
		{:else if avatar.modelError && !avatar.isModelLoading}
			<div
				class="flex animate-enter flex-col items-start gap-2 notice-error"
				role="alert">
				<span>{avatar.modelError}</span>
				<button
					class="button"
					type="button"
					onclick={() => void avatar.loadModel(imageUrl)}>Try again</button>
			</div>
		{:else if avatar.model}
			<p id="avatar-model-controls" class="text-2xs text-fg-muted">
				Drag or use the arrow keys to turn. Scroll or use + and − to zoom.
			</p>
		{/if}
	{/if}

	{#if avatar.isEditable && avatar.isChanged}
		<p class="animate-enter notice-note" role="status">
			The {view === "3D" ? "model" : "image"} shows the saved avatar{savedType !==
			draftType
				? ` as ${savedType}`
				: ""}. Save to see your changes.
		</p>
	{/if}
</section>
