<script lang="ts">
	import SignInIcon from "phosphor-svelte/lib/SignInIcon"
	import StarIcon from "phosphor-svelte/lib/StarIcon"
	import { PresenceType } from "../backend/bridge"
	import type { GamePlace, UserPresence } from "../backend/bridge"
	import { loadGamePlace } from "../games/game-place-cache"
	import { creatorByline, getGamesStore } from "../games/games-store.svelte"
	import {
		presenceClass,
		presenceDetail,
		presenceIcon,
		presenceLabel,
	} from "./account-model"

	let {
		presence,
		onJoin,
	}: {
		presence: UserPresence
		onJoin: (placeId: number, jobId: string) => void
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
		Icon = $derived(presenceIcon(presence)),
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

<div class="profile-presence">
	<button
		type="button"
		class="hover-value profile-presence-copy"
		aria-label={tooltip}
		data-tooltip={tooltip}>
		<span class={`profile-presence-label ${presenceClass(presence)}`}
			><Icon size={15} aria-hidden="true" />{label}{detail ? ":" : ""}</span>
		{#if detail}
			<span class="game-display-name">
				<span>{detail}</span>
				{#if inExperience && place && games.isFavorite(place.placeId)}
					<StarIcon
						class="favorite-name-star"
						size={12}
						weight="fill"
						aria-label="Favorite" />
				{/if}
			</span>
		{/if}
	</button>
	{#if joinable}
		<button
			class="icon-action profile-presence-join"
			type="button"
			aria-label="Fill launch options"
			data-tooltip="Fill launch options with this place and server"
			onclick={() => onJoin(joinPlaceId, presence.gameId ?? "")}>
			<SignInIcon size={16} aria-hidden="true" />
		</button>
	{/if}
</div>
