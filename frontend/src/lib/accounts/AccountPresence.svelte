<script lang="ts">
	import LogIn from "@lucide/svelte/icons/log-in"
	import Star from "@lucide/svelte/icons/star"
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
		updatesEnabled,
		onJoin,
	}: {
		presence: UserPresence
		updatesEnabled: boolean
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
				type === PresenceType.PresenceTypeUnknown
					? updatesEnabled
						? "Waiting for the next presence update."
						: "Turn on profile presence updates in Settings."
					: "",
			]
				.filter(Boolean)
				.join("\n"),
		)

	$effect(() => {
		const id = inExperience ? universeId : 0
		place = null
		if (id <= 0) return
		let current = true
		void loadGamePlace(id).then((result) => {
			if (current) place = result
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
			><Icon size={13} aria-hidden="true" />{label}{detail ? ":" : ""}</span>
		{#if detail}
			<span class="game-display-name">
				<span>{detail}</span>
				{#if inExperience && place && games.isFavorite(place.placeId)}
					<Star
						class="favorite-name-star"
						size={11}
						fill="currentColor"
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
			<LogIn size={14} aria-hidden="true" />
		</button>
	{/if}
</div>
