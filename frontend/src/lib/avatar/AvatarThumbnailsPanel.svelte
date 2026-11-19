<script lang="ts">
	import imagesIcon from "@phosphor-icons/core/regular/images.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { onMount } from "svelte"
	import { AvatarThumbnailType } from "../backend/bridge"
	import type { AvatarThumbnailChange } from "../backend/bridge"
	import type { AnimationLibrary } from "../model-viewer/model-subject"
	import Icon from "../shared/Icon.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import SettingsRowsSkeleton from "../shared/SettingsRowsSkeleton.svelte"
	import AvatarThumbnailImage from "./AvatarThumbnailImage.svelte"
	import AvatarThumbnailSection from "./AvatarThumbnailSection.svelte"
	import type { AvatarState } from "./avatar-state.svelte"
	import type { AvatarThumbnailsState } from "./avatar-thumbnails-state.svelte"

	let {
		avatar,
		thumbnails,
		library,
		canEdit,
		onSave,
	}: {
		avatar: AvatarState
		thumbnails: AvatarThumbnailsState
		library: AnimationLibrary
		canEdit: boolean
		onSave: (change: AvatarThumbnailChange) => void
	} = $props()

	const customizations = $derived(thumbnails.thumbnails)

	onMount(() => {
		if (canEdit) library.loadOwned()
	})
</script>

<div class="flex flex-col gap-4 p-3">
	<div class="flex min-h-7 items-center gap-2">
		<p class="min-w-0 flex-1 text-xs text-fg-muted">
			{canEdit
				? "Select the pose and the camera of the profile images. The 3D view starts with the full-body pose."
				: "How Roblox poses and frames the profile images. The 3D view starts with the full-body pose."}
		</p>
		<button
			class="button-ghost"
			type="button"
			aria-label={thumbnails.isLoading
				? "Refreshing thumbnails…"
				: "Refresh thumbnails"}
			data-tooltip="Refresh thumbnails"
			aria-busy={thumbnails.isLoading}
			disabled={thumbnails.isLoading || !!thumbnails.savingType}
			onclick={() => void thumbnails.refresh()}>
			<RefreshIcon isRefreshing={thumbnails.isLoading} />
		</button>
	</div>

	{#if thumbnails.saveError}
		<div class="flex items-start gap-2 notice-error" role="alert">
			<span class="min-w-0 flex-1">{thumbnails.saveError}</span>
			<button
				class="button-close"
				type="button"
				aria-label="Dismiss error"
				onclick={() => thumbnails.clearSaveError()}>
				<Icon svg={xIcon} size={14} aria-hidden="true" />
			</button>
		</div>
	{/if}

	{#if !customizations && thumbnails.error}
		<div class="empty-state" role="alert">
			<Icon svg={imagesIcon} size={28} aria-hidden="true" />
			<h2>Thumbnails unavailable</h2>
			<p>{thumbnails.error}</p>
			<button
				class="button"
				type="button"
				onclick={() => void thumbnails.refresh()}>Try again</button>
		</div>
	{:else if !customizations}
		<SettingsRowsSkeleton label="Loading thumbnails…" hasIcon={false} />
	{:else}
		{#if thumbnails.error}
			<p class="notice-error" role="alert">{thumbnails.error}</p>
		{/if}
		<AvatarThumbnailSection
			title="Headshot"
			description="The close-up image that profiles, friend lists, and chats show."
			customization={customizations.closeup}
			images={[{ label: "Headshot", url: customizations.headshotImageUrl }]}
			{canEdit}
			isSaving={thumbnails.savingType ===
				AvatarThumbnailType.AvatarThumbnailCloseup}
			isLocked={!!thumbnails.savingType}
			{library}
			{onSave} />
		<AvatarThumbnailSection
			title="Full body"
			description="The full-body image that profiles and the avatar page show."
			customization={customizations.fullBody}
			images={[{ label: "Full body", url: avatar.avatar?.imageUrl ?? "" }]}
			{canEdit}
			isSaving={thumbnails.savingType ===
				AvatarThumbnailType.AvatarThumbnailFullBody}
			isLocked={!!thumbnails.savingType}
			{library}
			{onSave} />
		<section class="settings-section" aria-labelledby="avatar-bust-heading">
			<h3 id="avatar-bust-heading">Bust</h3>
			<p class="text-xs text-fg-muted">
				Roblox always frames the bust itself. It has no pose or camera settings.
			</p>
			<div class="flex">
				<AvatarThumbnailImage label="Bust" url={customizations.bustImageUrl} />
			</div>
		</section>
	{/if}
</div>
