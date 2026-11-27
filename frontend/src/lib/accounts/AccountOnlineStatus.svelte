<script lang="ts">
	import { getProfilePreviews } from "../shared/profile-preview"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { OnlineStatusType, UserJoinability } from "../backend/bridge"
	import type { OnlineStatus } from "../backend/bridge"
	import { getGamesStore } from "../games/games-store.svelte"
	import {
		onlineStatusTextClass,
		onlineStatusDetail,
		onlineStatusIcon,
		onlineStatusLabel,
	} from "./account-model"
	import OnlineStatusCard from "./OnlineStatusCard.svelte"
	import { isInGame, OnlineStatusPlace } from "./online-status-place.svelte"
	import { onlineStatusJoinFor } from "./online-status-join"

	let {
		onlineStatus,
		joinability = UserJoinability.UserJoinabilityUnknown,
		onJoin,
		onJoinUser,
	}: {
		onlineStatus: OnlineStatus
		// joinability is whether the user's profile offers to join them, for a
		// user who is not one of the saved accounts.
		joinability?: UserJoinability
		onJoin: (placeId: number, serverId: string) => void
		// onJoinUser fills the launch options to follow the user, for a server
		// that the online status does not show.
		onJoinUser?: () => void
	} = $props()
	const previews = getProfilePreviews()

	const cardHoverOpenDelayMs = 400,
		cardHoverCloseDelayMs = 150

	const games = getGamesStore(),
		onlineStatusPlace = new OnlineStatusPlace(() => onlineStatus),
		cardId = $props.id()
	let trigger = $state<HTMLButtonElement>(),
		cardOpenedBy = $state<"hover" | "click" | null>(null),
		cardHoverTimer = 0

	const place = $derived(onlineStatusPlace.place),
		isOnlineStatusInGame = $derived(isInGame(onlineStatus)),
		join = $derived(onlineStatusJoinFor(onlineStatus, joinability, !!onJoinUser)),
		joinPlaceId = $derived(join.placeId),
		hasVisibleServer = $derived(join.hasVisibleServer),
		canFollowUser = $derived(join.canFollowUser),
		label = $derived(onlineStatusLabel(onlineStatus)),
		detail = $derived(
			onlineStatusDetail(onlineStatus, place ? games.displayName(place) : ""),
		),
		summary = $derived(detail ? `${label}: ${detail}` : label),
		tooltip = $derived(
			onlineStatus.statusType === OnlineStatusType.OnlineStatusTypeOnline &&
				!detail
				? `${summary}\nOn the website or app`
				: summary,
		)

	$effect(() => {
		if (!isOnlineStatusInGame) closeCard()
	})

	$effect(() => () => {
		clearTimeout(cardHoverTimer)
	})

	function scheduleCardHover(openedBy: "hover" | null, delayMs: number): void {
		clearTimeout(cardHoverTimer)
		if (cardOpenedBy === "click") return
		cardHoverTimer = window.setTimeout(() => {
			cardOpenedBy = openedBy
		}, delayMs)
	}

	function toggleCard(): void {
		clearTimeout(cardHoverTimer)
		cardOpenedBy = cardOpenedBy === "click" ? null : "click"
	}

	function closeCard(): void {
		clearTimeout(cardHoverTimer)
		cardOpenedBy = null
	}

	function isInsideCard(target: EventTarget | null): boolean {
		return (
			target instanceof Node &&
			(!!trigger?.contains(target) ||
				!!document.getElementById(cardId)?.contains(target))
		)
	}

	function fillLaunch(): void {
		closeCard()
		onJoin(joinPlaceId, onlineStatus.serverId ?? "")
	}

	function followUser(): void {
		closeCard()
		onJoinUser?.()
	}
</script>

<svelte:window
	onpointerdown={(event) => {
		if (cardOpenedBy && !isInsideCard(event.target)) closeCard()
	}}
	onscrollcapture={(event) => {
		if (cardOpenedBy && !isInsideCard(event.target)) closeCard()
	}}
	onresize={closeCard}
	onkeydown={(event) => {
		if (cardOpenedBy && event.key === "Escape") {
			event.preventDefault()
			closeCard()
			trigger?.focus()
		}
	}} />

<div
	class="flex min-h-6 min-w-0 items-center gap-1 text-xs"
	onfocusout={(event) => {
		if (cardOpenedBy && !isInsideCard(event.relatedTarget)) closeCard()
	}}>
	{#if isOnlineStatusInGame}
		<button
			bind:this={trigger}
			type="button"
			class="flex min-w-0 items-center gap-1 rounded-md text-left tabular-nums focus-visible:outline-offset-2"
			aria-label={`${summary}, show details`}
			aria-haspopup="dialog"
			aria-expanded={!!cardOpenedBy}
			aria-controls={cardOpenedBy ? cardId : undefined}
			onclick={toggleCard}
			onpointerenter={() => scheduleCardHover("hover", cardHoverOpenDelayMs)}
			onpointerleave={() => scheduleCardHover(null, cardHoverCloseDelayMs)}>
			{@render status()}
		</button>
		{#if cardOpenedBy && trigger}
			<OnlineStatusCard
				id={cardId}
				anchor={trigger}
				{onlineStatus}
				isJoinable={hasVisibleServer}
				{joinability}
				onFollowUser={canFollowUser ? followUser : undefined}
				onPointerEnter={() => scheduleCardHover("hover", 0)}
				onPointerLeave={() => scheduleCardHover(null, cardHoverCloseDelayMs)}
				onFillLaunch={fillLaunch} />
		{/if}
	{:else}
		<button
			type="button"
			class="flex min-w-0 cursor-help items-center gap-1 rounded-md text-left focus-visible:outline-offset-2"
			aria-label={tooltip}
			data-tooltip={tooltip}>{@render status()}</button>
	{/if}
	{#if isOnlineStatusInGame && joinPlaceId > 0}
		<button
			class="button-ghost-sm"
			type="button"
			aria-label="Show game"
			data-tooltip="Show game"
			onclick={() => previews.show({ kind: "Place", placeId: joinPlaceId })}>
			<Icon svg={magnifyingGlassIcon} size={14} aria-hidden="true" />
		</button>
	{/if}
</div>

{#snippet status()}
	<span
		class={[
			"inline-flex shrink-0 items-center gap-1 font-medium",
			onlineStatusTextClass(onlineStatus),
		]}
		><Icon
			svg={onlineStatusIcon(onlineStatus)}
			size={14}
			aria-hidden="true" />{label}{detail ? ":" : ""}</span>
	{#if detail}
		<span class="inline-flex min-w-0 items-center gap-1">
			<span class="truncate">{detail}</span>
			{#if place && games.isFavorite(place.placeId)}
				<Icon
					svg={starFillIcon}
					class="shrink-0 animate-pop text-accent-strong"
					size={12}
					aria-label="Favorite" />
			{/if}
		</span>
	{/if}
{/snippet}
