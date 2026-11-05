<script lang="ts">
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { UserPresence } from "../backend/bridge"
	import { getGamesStore } from "../games/games-store.svelte"
	import {
		presenceDetail,
		presenceIcon,
		presenceLabel,
		presenceTextClass,
	} from "./account-model"

	let { presence }: { presence: UserPresence } = $props()

	const games = getGamesStore()
	const location = $derived(presenceDetail(presence)),
		favorite = $derived(
			games.favorites.find((place) => place.placeId === presence.placeId) ??
				games.favorites.find((place) => place.placeId === presence.rootPlaceId),
		),
		detail = $derived(location && favorite ? games.displayName(favorite) : location)
</script>

<small
	class={[
		"flex min-w-0 items-center justify-end gap-1 text-2xs text-fg-muted",
		detail ? "w-full" : "ml-auto shrink-0",
	]}>
	<span
		class={[
			"inline-flex shrink-0 items-center gap-1 font-medium",
			presenceTextClass(presence),
		]}
		><Icon
			svg={presenceIcon(presence)}
			size={12}
			aria-hidden="true" />{presenceLabel(presence)}{detail ? ":" : ""}</span>
	{#if detail}<span class="min-w-0 truncate">{detail}</span>{/if}
	{#if detail && favorite}<Icon
			svg={starFillIcon}
			class="shrink-0 animate-pop text-accent-strong"
			size={12}
			aria-label="Favorite" />{/if}
</small>
