<script lang="ts">
	import SealCheckIcon from "phosphor-svelte/lib/SealCheckIcon"
	import StarIcon from "phosphor-svelte/lib/StarIcon"
	import type { GamePlaceSummary } from "../backend/bridge"
	import GameIcon from "../games/GameIcon.svelte"
	import { loadGamePlace } from "../games/game-place-cache"
	import { creatorByline, getGamesStore } from "../games/games-store.svelte"

	let { universeId, placeId }: { universeId: number; placeId: number } = $props()
	const games = getGamesStore()

	let summary = $state<GamePlaceSummary | null>(null),
		loading = $state(true)
	const place = $derived(summary?.place ?? null),
		rootPlace = $derived(summary?.rootPlace ?? null),
		rootName = $derived(rootPlace ? games.displayName(rootPlace) : "")

	$effect(() => {
		const request = loadGamePlace(universeId, placeId)
		summary = null
		loading = true
		let current = true
		void request.then((result) => {
			if (!current) return result
			summary = result
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
		<span class="logs-explorer-game-icon">
			<GameIcon url={place?.iconUrl ?? ""} />
			{#if rootPlace}<button
					class="hover-value logs-explorer-root-icon"
					type="button"
					aria-label={`Subplace of ${rootName}`}
					data-tooltip={`Subplace of ${rootName}`}
					data-tooltip-strong={rootName}
					><GameIcon url={rootPlace.iconUrl} /></button
				>{/if}
		</span>
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
