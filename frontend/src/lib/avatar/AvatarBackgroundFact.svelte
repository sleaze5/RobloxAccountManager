<script lang="ts">
	import panoramaIcon from "@phosphor-icons/core/regular/panorama.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import type { AvatarPortraitLoad } from "./avatar-portrait.svelte"

	let { portraitLoad }: { portraitLoad: AvatarPortraitLoad } = $props()

	const background = $derived(portraitLoad.portrait?.background ?? null),
		isUnavailable = $derived(
			portraitLoad.hasFailed || !!portraitLoad.portrait?.isBackgroundUnavailable,
		)
</script>

<div data-tone={background ? undefined : "muted"}>
	<dt><Icon svg={panoramaIcon} size={16} aria-hidden="true" />Avatar background</dt>
	<dd>
		{#if background}
			<span>{background.name || "Unnamed background"}</span>
		{:else if portraitLoad.isLoading}
			<Skeleton className="w-28" label="Loading avatar background…" />
		{:else if isUnavailable}
			<span>Unavailable</span>
		{:else}
			<span>None</span>
		{/if}
	</dd>
</div>
