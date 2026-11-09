<script lang="ts">
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import signInIcon from "@phosphor-icons/core/regular/sign-in.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { PresenceType } from "../backend/bridge"
	import type { UserPresence } from "../backend/bridge"
	import { getGamesStore } from "../games/games-store.svelte"
	import {
		presenceTextClass,
		presenceDetail,
		presenceIcon,
		presenceLabel,
	} from "./account-model"
	import PresenceCard from "./PresenceCard.svelte"
	import { isInExperience, PresencePlace } from "./presence-place.svelte"

	let {
		presence,
		onJoin,
		onSearchPlace,
		onSearchUniverse,
	}: {
		presence: UserPresence
		onJoin: (placeId: number, jobId: string) => void
		onSearchPlace: (placeId: number) => void
		onSearchUniverse: (universeId: number) => void
	} = $props()

	const cardHoverOpenDelayMs = 400,
		cardHoverCloseDelayMs = 150

	const games = getGamesStore(),
		presencePlace = new PresencePlace(() => presence),
		cardId = $props.id()
	let trigger = $state<HTMLButtonElement>(),
		cardOpenedBy = $state<"hover" | "click" | null>(null),
		cardHoverTimer = 0

	const place = $derived(presencePlace.place),
		inExperience = $derived(isInExperience(presence)),
		joinPlaceId = $derived(presence.placeId ?? 0),
		joinable = $derived(
			presence.userPresenceType === PresenceType.PresenceTypeInGame &&
				joinPlaceId > 0 &&
				!!presence.gameId,
		),
		label = $derived(presenceLabel(presence)),
		detail = $derived(
			presenceDetail(presence, place ? games.displayName(place) : ""),
		),
		summary = $derived(detail ? `${label}: ${detail}` : label),
		tooltip = $derived(
			presence.userPresenceType === PresenceType.PresenceTypeOnline && !detail
				? `${summary}\nOn the website or app`
				: summary,
		)

	$effect(() => {
		if (!inExperience) closeCard()
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
		onJoin(joinPlaceId, presence.gameId ?? "")
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
	{#if inExperience}
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
			<PresenceCard
				id={cardId}
				anchor={trigger}
				{presence}
				isJoinable={joinable}
				onPointerEnter={() => scheduleCardHover("hover", 0)}
				onPointerLeave={() => scheduleCardHover(null, cardHoverCloseDelayMs)}
				onFillLaunch={fillLaunch}
				{onSearchPlace}
				{onSearchUniverse} />
		{/if}
	{:else}
		<button
			type="button"
			class="flex min-w-0 cursor-help items-center gap-1 rounded-md text-left focus-visible:outline-offset-2"
			aria-label={tooltip}
			data-tooltip={tooltip}>{@render status()}</button>
	{/if}
	{#if joinable}
		<button
			class="button-ghost-sm"
			type="button"
			aria-label="Fill launch options"
			data-tooltip="Fill launch options with this place and server"
			onclick={fillLaunch}>
			<Icon svg={signInIcon} size={14} aria-hidden="true" />
		</button>
	{/if}
	{#if inExperience && joinPlaceId > 0}
		<button
			class="button-ghost-sm"
			type="button"
			aria-label="Find this place in Games"
			data-tooltip="Find this place in Games"
			onclick={() => onSearchPlace(joinPlaceId)}>
			<Icon svg={magnifyingGlassIcon} size={14} aria-hidden="true" />
		</button>
	{/if}
</div>

{#snippet status()}
	<span
		class={[
			"inline-flex shrink-0 items-center gap-1 font-medium",
			presenceTextClass(presence),
		]}
		><Icon
			svg={presenceIcon(presence)}
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
