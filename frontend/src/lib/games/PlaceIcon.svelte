<script lang="ts">
	import type { GamePlace } from "../backend/bridge"
	import GameIcon from "./GameIcon.svelte"
	import { getGamesStore } from "./games-store.svelte"

	let {
		place,
		rootPlace,
		size = "md",
	}: {
		place: GamePlace | null
		rootPlace: GamePlace | null
		size?: "md" | "lg"
	} = $props()
	const games = getGamesStore(),
		rootName = $derived(rootPlace ? games.displayName(rootPlace) : "")
</script>

<span class="relative shrink-0">
	<GameIcon url={place?.iconUrl ?? ""} {size} />
	{#if rootPlace}<button
			class="absolute -right-1.5 -bottom-1.5 block cursor-help rounded-sm ring-2 ring-canvas focus-visible:outline-offset-2"
			type="button"
			aria-label={`Subplace of ${rootName}`}
			data-tooltip={`Subplace of ${rootName}`}
			data-tooltip-strong={rootName}
			><GameIcon url={rootPlace.iconUrl} size="xs" /></button
		>{/if}
</span>
