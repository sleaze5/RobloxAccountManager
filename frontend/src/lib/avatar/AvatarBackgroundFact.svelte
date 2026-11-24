<script lang="ts">
	import panoramaIcon from "@phosphor-icons/core/regular/panorama.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import type { EquippedAvatarBackground } from "./equipped-avatar-background.svelte"

	// AvatarBackgroundFact names the background that the user equipped, as a
	// row of a facts list.
	let { equipped }: { equipped: EquippedAvatarBackground } = $props()
</script>

<div data-tone={equipped.background ? undefined : "muted"}>
	<dt><Icon svg={panoramaIcon} size={16} aria-hidden="true" />Avatar background</dt>
	<dd>
		{#if equipped.background}
			<span>{equipped.background.name || "Unnamed background"}</span>
		{:else if equipped.isLoading}
			<Skeleton className="w-28" label="Loading avatar background…" />
		{:else if equipped.hasFailed}
			<span>Unavailable</span>
		{:else}
			<span>None</span>
		{/if}
	</dd>
</div>
