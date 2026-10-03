<script lang="ts">
	import BadgeCheck from "@lucide/svelte/icons/badge-check"
	import EyeOff from "@lucide/svelte/icons/eye-off"
	import Gamepad2 from "@lucide/svelte/icons/gamepad-2"
	import Globe2 from "@lucide/svelte/icons/earth"
	import Hammer from "@lucide/svelte/icons/hammer"
	import LogIn from "@lucide/svelte/icons/log-in"
	import Moon from "@lucide/svelte/icons/moon"
	import Radio from "@lucide/svelte/icons/radio"
	import Star from "@lucide/svelte/icons/star"
	import { PresenceType } from "../backend/bridge"
	import type { GamePlace, UserPresence } from "../backend/bridge"
	import { loadGamePlace } from "../games/game-place-cache"
	import { creatorByline, getGamesStore } from "../games/games-store.svelte"
	import { presenceClass } from "./account-model"

	let {
		presence,
		updatesEnabled,
		onJoin,
	}: {
		presence: UserPresence
		updatesEnabled: boolean
		onJoin: (placeId: number, jobId: string) => void
	} = $props()

	interface PresenceView {
		heading: string
		title: string
		detail: string
		status: string
	}

	const idleViews: Partial<Record<PresenceType, PresenceView>> = {
		[PresenceType.PresenceTypeOffline]: {
			heading: "Offline",
			title: "Not on Roblox",
			detail: "",
			status: "Offline",
		},
		[PresenceType.PresenceTypeInvisible]: {
			heading: "Invisible",
			title: "Appearing offline",
			detail: "",
			status: "Invisible",
		},
	}

	const games = getGamesStore()
	let place = $state<GamePlace | null>(null)

	const type = $derived(presence.userPresenceType),
		universeId = $derived(presence.universeId ?? 0),
		location = $derived(presence.lastLocation?.trim() ?? ""),
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
		onlineDetail = $derived(
			location && location.toLowerCase() !== "website"
				? location
				: "On the website or app",
		),
		view = $derived.by((): PresenceView => {
			if (type === PresenceType.PresenceTypeInGame) {
				return experienceView("Playing", "A Roblox experience", "In game")
			}
			if (type === PresenceType.PresenceTypeInStudio) {
				return experienceView("In Studio", "Roblox Studio", "Editing in Studio")
			}
			if (type === PresenceType.PresenceTypeOnline) {
				return {
					heading: "Online",
					title: "Browsing Roblox",
					detail: onlineDetail,
					status: "Online",
				}
			}
			return (
				idleViews[type] ?? {
					heading: "Presence",
					title: "Presence unavailable",
					detail: updatesEnabled
						? "Waiting for the next presence update."
						: "Turn on profile presence updates in Settings.",
					status: "Unknown",
				}
			)
		})

	function experienceView(
		heading: string,
		fallback: string,
		status: string,
	): PresenceView {
		return {
			heading,
			title: (place && games.displayName(place)) || location || fallback,
			detail: place?.creatorName ? creatorByline(place) : "",
			status,
		}
	}

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

<section
	class="presence-card"
	class:active={inExperience}
	aria-labelledby="presence-heading">
	<h3 id="presence-heading" class="profile-section-label">{view.heading}</h3>
	<div class="presence-body">
		<div class="presence-art" aria-hidden="true">
			{#if place?.iconUrl}
				<img src={place.iconUrl} alt="" draggable="false" />
			{:else if type === PresenceType.PresenceTypeInGame}
				<Gamepad2 size={22} />
			{:else if type === PresenceType.PresenceTypeInStudio}
				<Hammer size={22} />
			{:else if type === PresenceType.PresenceTypeOnline}
				<Globe2 size={22} />
			{:else if type === PresenceType.PresenceTypeOffline}
				<Moon size={22} />
			{:else if type === PresenceType.PresenceTypeInvisible}
				<EyeOff size={22} />
			{:else}
				<Radio size={22} />
			{/if}
		</div>
		<div class="presence-copy">
			<strong class="game-display-name" data-tooltip={view.title}
				><span>{view.title}</span
				>{#if inExperience && place && games.isFavorite(place.placeId)}<Star
						class="favorite-name-star"
						size={11}
						fill="currentColor"
						aria-label="Favorite" />{/if}</strong>
			{#if view.detail}<span class="presence-detail"
					>{view.detail}{#if place?.creatorVerified}<BadgeCheck
							class="creator-verified"
							size={12}
							aria-label="Verified creator" />{/if}</span
				>{/if}
			<span class="presence-status"
				><i class={`status-dot ${presenceClass(presence)}`} aria-hidden="true"
				></i
				>{view.status}{#if joinable}<span
						class="presence-server"
						data-tooltip={`Job ID: ${presence.gameId}`}
						>{presence.gameId?.slice(0, 8)}</span
					>{/if}</span>
		</div>
	</div>
	{#if joinable}
		<button
			class="launch-control presence-join"
			type="button"
			data-tooltip="Use this place and server in launch options"
			onclick={() => onJoin(joinPlaceId, presence.gameId ?? "")}>
			<LogIn size={14} aria-hidden="true" />Fill launch options
		</button>
	{/if}
</section>
