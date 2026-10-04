<script lang="ts">
	import SealCheckIcon from "phosphor-svelte/lib/SealCheckIcon"
	import StarIcon from "phosphor-svelte/lib/StarIcon"
	import type { GamePlace } from "../backend/bridge"
	import GameIcon from "../games/GameIcon.svelte"
	import { loadGamePlace } from "../games/game-place-cache"
	import { creatorByline, getGamesStore } from "../games/games-store.svelte"

	let { universeId }: { universeId: number } = $props()
	const games = getGamesStore()

	let place = $state<GamePlace | null>(null),
		loading = $state(true)

	$effect(() => {
		const id = universeId
		place = null
		loading = true
		let current = true
		void loadGamePlace(id).then((result) => {
			if (!current) return result
			place = result
			loading = false
			return result
		})
		return () => {
			current = false
		}
	})
</script>

{#if loading || place}
	<div class="logs-explorer-profile" aria-busy={loading}>
		<GameIcon url={place?.iconUrl ?? ""} />
		<div class="logs-explorer-profile-copy">
			{#if place}
				<strong class="game-display-name"
					><span>{games.displayName(place)}</span
					>{#if games.isFavorite(place.placeId)}<StarIcon
							class="favorite-name-star"
							size={12}
							weight="fill"
							aria-label="Favorite" />{/if}</strong>
				<span
					>{creatorByline(place)}{#if place.creatorVerified}<SealCheckIcon
							class="creator-verified"
							size={14}
							aria-label="Verified creator" />{/if}</span>
			{:else}
				<strong class="loading">Loading game…</strong>
			{/if}
		</div>
	</div>
{/if}
