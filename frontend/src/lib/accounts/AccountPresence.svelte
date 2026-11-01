<script lang="ts">
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import signInIcon from "@phosphor-icons/core/regular/sign-in.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { PresenceType } from "../backend/bridge"
	import type { GamePlace, UserPresence } from "../backend/bridge"
	import { loadGamePlace } from "../games/game-place-cache"
	import { creatorByline, getGamesStore } from "../games/games-store.svelte"
	import {
		presenceTextClass,
		presenceDetail,
		presenceIcon,
		presenceLabel,
	} from "./account-model"

	let {
		presence,
		onJoin,
		onSearchPlace,
	}: {
		presence: UserPresence
		onJoin: (placeId: number, jobId: string) => void
		onSearchPlace: (placeId: number) => void
	} = $props()

	const games = getGamesStore()
	let place = $state<GamePlace | null>(null)

	const type = $derived(presence.userPresenceType),
		universeId = $derived(presence.universeId ?? 0),
		inExperience = $derived(
			type === PresenceType.PresenceTypeInGame ||
				type === PresenceType.PresenceTypeInStudio,
		),
		joinPlaceId = $derived(presence.placeId ?? 0),
		joinable = $derived(
			type === PresenceType.PresenceTypeInGame &&
				joinPlaceId > 0 &&
				!!presence.gameId,
		),
		label = $derived(presenceLabel(presence)),
		icon = $derived(presenceIcon(presence)),
		detail = $derived(
			presenceDetail(presence, (place && games.displayName(place)) || ""),
		),
		tooltip = $derived(
			[
				detail ? `${label}: ${detail}` : label,
				inExperience && place?.creatorName
					? `${creatorByline(place)}${place.creatorVerified ? " (Verified creator)" : ""}`
					: "",
				type === PresenceType.PresenceTypeOnline && !detail
					? "On the website or app"
					: "",
				joinable ? `Job ID: ${presence.gameId}` : "",
			]
				.filter(Boolean)
				.join("\n"),
		)

	$effect(() => {
		const id = inExperience ? universeId : 0,
			placeId = joinPlaceId
		place = null
		if (id <= 0 || placeId <= 0) return
		let current = true
		void loadGamePlace(id, placeId).then((result) => {
			if (current) place = result?.place ?? null
			return result
		})
		return () => {
			current = false
		}
	})
</script>

<div class="flex min-h-6 min-w-0 items-center gap-1 text-xs">
	<button
		type="button"
		class="flex min-w-0 cursor-help items-center gap-1 rounded-md text-left tabular-nums focus-visible:outline-offset-2"
		aria-label={tooltip}
		data-tooltip={tooltip}>
		<span
			class={[
				"inline-flex shrink-0 items-center gap-1 font-medium",
				presenceTextClass(presence),
			]}
			><Icon svg={icon} size={14} aria-hidden="true" />{label}{detail
				? ":"
				: ""}</span>
		{#if detail}
			<span class="inline-flex min-w-0 items-center gap-1">
				<span class="truncate">{detail}</span>
				{#if inExperience && place && games.isFavorite(place.placeId)}
					<Icon
						svg={starFillIcon}
						class="shrink-0 animate-pop text-accent-strong"
						size={12}
						aria-label="Favorite" />
				{/if}
			</span>
		{/if}
	</button>
	{#if joinable}
		<button
			class="btn-ghost-sm"
			type="button"
			aria-label="Fill launch options"
			data-tooltip="Fill launch options with this place and server"
			onclick={() => onJoin(joinPlaceId, presence.gameId ?? "")}>
			<Icon svg={signInIcon} size={14} aria-hidden="true" />
		</button>
	{/if}
	{#if inExperience && joinPlaceId > 0}
		<button
			class="btn-ghost-sm"
			type="button"
			aria-label="Find this place in Games"
			data-tooltip="Find this place in Games"
			onclick={() => onSearchPlace(joinPlaceId)}>
			<Icon svg={magnifyingGlassIcon} size={14} aria-hidden="true" />
		</button>
	{/if}
</div>
