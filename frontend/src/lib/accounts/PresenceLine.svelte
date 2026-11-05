<script lang="ts">
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { UserPresence } from "../backend/bridge"
	import { getGamesStore } from "../games/games-store.svelte"
	import { presenceDetail, presenceLabel, presenceTextClass } from "./account-model"

	let { presence }: { presence: UserPresence } = $props()

	const games = getGamesStore()
	const favorite = $derived(
		games.favorites.find((place) => place.placeId === presence.placeId) ??
			games.favorites.find((place) => place.placeId === presence.rootPlaceId),
	)
</script>

<small class="flex w-full min-w-0 items-center gap-1 text-2xs text-fg">
	<span class={["shrink-0 font-medium", presenceTextClass(presence)]}
		>{presenceLabel(presence)}:</span>
	<span class="min-w-0 truncate"
		>{favorite ? games.displayName(favorite) : presenceDetail(presence)}</span>
	{#if favorite}<Icon
			svg={starFillIcon}
			class="shrink-0 animate-pop text-accent-strong"
			size={12}
			aria-label="Favorite" />{/if}
</small>
