<script lang="ts">
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { GamePlaceSummary } from "../backend/bridge"
	import { loadGamePlace } from "./game-place-cache"
	import PlaceIcon from "./PlaceIcon.svelte"
	import { creatorByline, getGamesStore } from "./games-store.svelte"

	let {
		universeId,
		placeId,
		onSearch,
	}: { universeId: number; placeId: number; onSearch?: () => void } = $props()
	const games = getGamesStore()

	let summary = $state<GamePlaceSummary | null>(null),
		loading = $state(true)
	const place = $derived(summary?.place ?? null),
		rootPlace = $derived(summary?.rootPlace ?? null)

	$effect(() => {
		const request = loadGamePlace(universeId, placeId)
		summary = null
		loading = true
		let current = true
		void request.then((loadedSummary) => {
			if (!current) return loadedSummary
			summary = loadedSummary
			loading = false
			return loadedSummary
		})
		return () => {
			current = false
		}
	})
</script>

{#if loading || place}
	<div class="flex min-w-0 items-center gap-3" aria-busy={loading}>
		<PlaceIcon {place} {rootPlace} size="lg" isRootNameShownNearby />
		<div class="flex min-w-0 flex-col leading-snug">
			{#if place}
				<strong
					class="inline-flex min-w-0 items-center gap-1 text-sm font-semibold text-fg"
					><span class="truncate">{games.displayName(place)}</span
					>{#if games.isFavorite(place.placeId)}<Icon
							svg={starFillIcon}
							class="shrink-0 animate-pop text-accent-strong"
							size={12}
							aria-label="Favorite" />{/if}</strong>
				{#if rootPlace}<span class="truncate text-xs text-fg-muted"
						>Subplace of <span class="font-medium text-fg"
							>{games.displayName(rootPlace)}</span
						></span
					>{/if}
				<span class="truncate text-xs text-fg-muted"
					>{creatorByline(place)}{#if place.creatorVerified}<Icon
							svg={sealCheckIcon}
							class="relative top-0.5 ml-1 inline-block"
							size={14}
							aria-label="Verified creator" />{/if}</span>
			{:else}
				<strong class="font-medium text-fg-faint">Loading game…</strong>
			{/if}
		</div>
		{#if place && onSearch}<button
				class="-ms-2 button-ghost-sm shrink-0 self-start"
				type="button"
				aria-label="Find this place in Games"
				data-tooltip="Find this place in Games"
				onclick={onSearch}
				><Icon svg={magnifyingGlassIcon} size={14} aria-hidden="true" /></button
			>{/if}
	</div>
{/if}
