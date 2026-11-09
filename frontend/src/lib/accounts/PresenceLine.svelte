<script lang="ts">
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { UserPresence } from "../backend/bridge"
	import { getGamesStore } from "../games/games-store.svelte"
	import { presenceDetail, presenceLabel, presenceTextClass } from "./account-model"
	import { PresencePlace } from "./presence-place.svelte"

	let { presence }: { presence: UserPresence } = $props()

	const games = getGamesStore(),
		presencePlace = new PresencePlace(() => presence)
	const place = $derived(presencePlace.place)
</script>

<small class="flex w-full min-w-0 items-center gap-1 text-2xs text-fg">
	<span class={["shrink-0 font-medium", presenceTextClass(presence)]}
		>{presenceLabel(presence)}:</span>
	<span class="min-w-0 truncate"
		>{presenceDetail(presence, place ? games.displayName(place) : "")}</span>
	{#if place && games.isFavorite(place.placeId)}<Icon
			svg={starFillIcon}
			class="shrink-0 animate-pop text-accent-strong"
			size={12}
			aria-label="Favorite" />{/if}
</small>
