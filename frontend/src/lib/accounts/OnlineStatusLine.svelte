<script lang="ts">
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { OnlineStatus } from "../backend/bridge"
	import { getGamesStore } from "../games/games-store.svelte"
	import {
		onlineStatusDetail,
		onlineStatusLabel,
		onlineStatusTextClass,
	} from "./account-model"
	import { OnlineStatusPlace } from "./online-status-place.svelte"

	let { onlineStatus }: { onlineStatus: OnlineStatus } = $props()

	const games = getGamesStore(),
		onlineStatusPlace = new OnlineStatusPlace(() => onlineStatus)
	const place = $derived(onlineStatusPlace.place)
</script>

<small class="flex w-full min-w-0 items-center gap-1 text-2xs text-fg">
	<span class={["shrink-0 font-medium", onlineStatusTextClass(onlineStatus)]}
		>{onlineStatusLabel(onlineStatus)}:</span>
	<span class="min-w-0 truncate"
		>{onlineStatusDetail(
			onlineStatus,
			place ? games.displayName(place) : "",
		)}</span>
	{#if place && games.isFavorite(place.placeId)}<Icon
			svg={starFillIcon}
			class="shrink-0 animate-pop text-accent-strong"
			size={12}
			aria-label="Favorite" />{/if}
</small>
