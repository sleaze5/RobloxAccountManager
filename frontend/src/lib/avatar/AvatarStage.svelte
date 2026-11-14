<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import personArmsSpreadIcon from "@phosphor-icons/core/regular/person-arms-spread.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { AvatarState } from "./avatar-state.svelte"

	let { avatar, username }: { avatar: AvatarState; username: string } = $props()

	let failedImageUrl = $state("")

	const imageUrl = $derived(avatar.avatar?.imageUrl ?? ""),
		savedType = $derived(avatar.avatar?.avatarType ?? ""),
		draftType = $derived(avatar.draft?.avatarType ?? "")
</script>

<aside
	class="flex w-60 shrink-0 flex-col gap-3 overflow-y-auto border-r border-line-subtle bg-surface p-3"
	aria-label="Avatar image">
	<figure
		class="relative grid aspect-square w-full place-items-center overflow-hidden rounded-lg border border-line-subtle bg-canvas">
		<!-- The plinth grounds the render, which Roblox draws without a floor. -->
		<span
			class="absolute bottom-5 h-3 w-28 rounded-full bg-hover-overlay"
			aria-hidden="true"></span>
		{#if imageUrl && failedImageUrl !== imageUrl}
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
				class="absolute top-2 left-2 rounded-sm border border-line-strong bg-overlay px-1 font-mono text-2xs leading-4 font-medium text-fg-muted"
				>{savedType}</span>
		{/if}
		{#if avatar.isRenderingImage}
			<span
				class="absolute right-2 bottom-2 flex animate-enter items-center gap-1 rounded-sm border border-line-strong bg-overlay px-1.5 text-2xs leading-5 font-medium text-fg"
				role="status">
				<Icon
					svg={circleNotchIcon}
					class="spinner"
					size={12}
					aria-hidden="true" />
				Rendering…
			</span>
		{/if}
	</figure>

	{#if avatar.isEditable && avatar.isChanged}
		<p class="animate-enter notice-note" role="status">
			The image shows the saved avatar{savedType !== draftType
				? ` as ${savedType}`
				: ""}. Save to see your changes.
		</p>
	{/if}
</aside>
