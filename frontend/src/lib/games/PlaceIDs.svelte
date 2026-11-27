<script lang="ts">
	import type { Snippet } from "svelte"

	// PlaceIDs is the ID line of a game profile: the Place ID, the Universe ID
	// once it is known, and the Root place ID of a subplace. universeAction
	// follows the Universe ID, such as a search for the places of the game.
	let {
		placeId,
		universeId,
		rootPlaceId,
		universeAction,
	}: {
		placeId: number
		universeId: number
		rootPlaceId: number
		universeAction?: Snippet
	} = $props()
</script>

<div
	class="flex min-w-0 flex-wrap items-center gap-1 font-mono text-2xs wrap-anywhere text-fg-muted">
	<span class="select-text">Place ID: {placeId}</span>
	{#if universeId > 0}
		<span class="text-fg-faint" aria-hidden="true">·</span>
		<span class="select-text">Universe ID: {universeId}</span>
		{@render universeAction?.()}
	{/if}
	{#if rootPlaceId > 0}
		<span class="text-fg-faint" aria-hidden="true">·</span>
		<span class="select-text">Root place ID: {rootPlaceId}</span>
	{/if}
</div>
