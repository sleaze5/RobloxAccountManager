<script lang="ts">
	import SealCheckIcon from "phosphor-svelte/lib/SealCheckIcon"
	import MagnifyingGlassIcon from "phosphor-svelte/lib/MagnifyingGlassIcon"
	import StarIcon from "phosphor-svelte/lib/StarIcon"
	import type { GamePlaceSummary } from "../backend/bridge"
	import GameIcon from "../games/GameIcon.svelte"
	import { loadGamePlace } from "../games/game-place-cache"
	import { creatorByline, getGamesStore } from "../games/games-store.svelte"

	let {
		universeId,
		placeId,
		onSearch,
	}: { universeId: number; placeId: number; onSearch: () => void } = $props()
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
	<div class="flex min-w-0 items-center gap-3" aria-busy={loading}>
		<span class="relative shrink-0">
			<GameIcon url={place?.iconUrl ?? ""} size="lg" />
			{#if rootPlace}<button
					class="absolute -right-1.5 -bottom-1.5 block cursor-help rounded-sm ring-2 ring-canvas focus-visible:outline-offset-2"
					type="button"
					aria-label={`Subplace of ${rootName}`}
					data-tooltip={`Subplace of ${rootName}`}
					data-tooltip-strong={rootName}
					><GameIcon url={rootPlace.iconUrl} size="xs" /></button
				>{/if}
		</span>
		<div class="flex min-w-0 flex-col leading-snug">
			{#if place}
				<strong
					class="inline-flex min-w-0 items-center gap-1 text-sm font-semibold text-fg"
					><span class="truncate">{games.displayName(place)}</span
					>{#if games.isFavorite(place.placeId)}<StarIcon
							class="shrink-0 animate-pop text-accent-strong"
							size={12}
							weight="fill"
							aria-label="Favorite" />{/if}</strong>
				<span class="truncate text-xs text-fg-muted"
					>{creatorByline(place)}{#if place.creatorVerified}<SealCheckIcon
							class="relative top-0.5 ml-1 inline-block"
							size={14}
							aria-label="Verified creator" />{/if}</span>
			{:else}
				<strong class="font-medium text-fg-faint">Loading game…</strong>
			{/if}
		</div>
		{#if place}<button
				class="-ms-2 btn-ghost-sm shrink-0 self-start"
				type="button"
				aria-label="Find this place in Games"
				data-tooltip="Find this place in Games"
				onclick={onSearch}
				><MagnifyingGlassIcon size={14} aria-hidden="true" /></button
			>{/if}
	</div>
{/if}
