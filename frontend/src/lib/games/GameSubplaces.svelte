<script lang="ts">
	import gameControllerIcon from "@phosphor-icons/core/regular/game-controller.svg?raw"
	import type { GamePlace } from "../backend/bridge"
	import ItemTile from "../shared/ItemTile.svelte"
	import { getProfilePreviews } from "../shared/profile-preview"
	import TileCarousel from "../shared/TileCarousel.svelte"
	import { loadGameSubplaces } from "./game-subplaces"

	// GameSubplaces shows the subplaces of a game in a carousel below its root
	// place. A game without subplaces shows nothing.
	let { universeId }: { universeId: number } = $props()

	const previews = getProfilePreviews(),
		headingId = $props.id()
	let subplaces = $state.raw<GamePlace[]>([])

	$effect(() => {
		const shownUniverseId = universeId
		let isCurrent = true
		subplaces = []
		if (shownUniverseId <= 0) return
		void (async () => {
			try {
				const loadedSubplaces = await loadGameSubplaces(shownUniverseId)
				if (isCurrent) subplaces = loadedSubplaces
			} catch {
				// The backend logs the failure. The subplaces are secondary, so
				// the profile shows the game without them.
			}
		})()
		return () => {
			isCurrent = false
		}
	})
</script>

{#if subplaces.length > 0}
	<div class="min-w-0 animate-enter">
		<TileCarousel id={headingId} label="Subplaces">
			{#snippet heading()}
				Subplaces<span class="font-mono text-2xs font-normal text-fg-faint"
					>({subplaces.length})</span>
			{/snippet}
			{#each subplaces as subplace (subplace.placeId)}
				<li class="w-36 shrink-0">
					<ItemTile
						name={subplace.name}
						imageUrl={subplace.iconUrl}
						fallbackIcon={gameControllerIcon}
						detail={`ID: ${subplace.placeId}`}
						actionLabel="Show place"
						onOpen={() =>
							previews.show({
								kind: "Place",
								placeId: subplace.placeId,
							})} />
				</li>
			{/each}
		</TileCarousel>
	</div>
{/if}
