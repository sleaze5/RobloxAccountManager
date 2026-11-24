<script lang="ts">
	import ImageMenuTarget from "../layout/ImageMenuTarget.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import type { AvatarPortraitLoad } from "./avatar-portrait.svelte"

	// AvatarPortraitFrame shows the full-body avatar in front of the background
	// that the user equipped, as the Roblox profile does. Until the portrait
	// loads, or when it has no full-body image, it shows the headshot.
	let {
		portraitLoad,
		headshotUrl,
		label,
		onOpenAvatar,
	}: {
		portraitLoad: AvatarPortraitLoad
		headshotUrl: string
		// label names the image, such as "@builderman's avatar".
		label: string
		onOpenAvatar: () => void
	} = $props()

	const fullBodyImageUrl = $derived(portraitLoad.portrait?.fullBodyImageUrl ?? ""),
		backgroundImageUrl = $derived(portraitLoad.portrait?.background?.imageUrl ?? "")
</script>

<ImageMenuTarget
	imageUrl={fullBodyImageUrl || headshotUrl}
	{label}
	openLabel="Show avatar"
	onOpen={onOpenAvatar}>
	<span
		class="relative grid size-36 place-items-center overflow-hidden rounded-md border border-line-strong bg-raised"
		aria-hidden="true">
		{#if fullBodyImageUrl}
			<ProfileImage
				url={fullBodyImageUrl}
				backgroundUrl={backgroundImageUrl}
				isContained />
		{:else if portraitLoad.isLoading}
			<span class="absolute inset-0 skeleton"></span>
		{:else}
			<ProfileImage url={headshotUrl} backgroundUrl={backgroundImageUrl} />
		{/if}
	</span>
</ImageMenuTarget>
