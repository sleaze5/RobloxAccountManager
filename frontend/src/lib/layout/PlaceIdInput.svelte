<script lang="ts">
	import { tick } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import type { GamePlace } from "../backend/bridge"
	import GameIcon from "../games/GameIcon.svelte"
	import { getGamesStore } from "../games/games-store.svelte"
	import { indexPlaces, matchPlaces } from "../games/place-match"
	import { menuIn, menuOut } from "../shared/presence"

	const maxMenuHeight = 238

	let { store }: { store: AccountStore } = $props()
	const games = getGamesStore(),
		listID = $props.id()
	let input = $state<HTMLInputElement>(),
		list = $state<HTMLDivElement>(),
		open = $state(false),
		active = $state(0),
		navigated = $state(false),
		listStyle = $state("")
	const favorites = $derived(indexPlaces(games.favorites)),
		suggestions = $derived(matchPlaces(favorites, store.launchInput.placeId)),
		shown = $derived(
			open && (suggestions.length > 0 || !store.launchInput.placeId.trim()),
		),
		activeIndex = $derived(Math.min(active, suggestions.length - 1))

	function position(): void {
		const bounds = input?.getBoundingClientRect()
		if (!bounds) return
		const width = Math.min(Math.max(bounds.width, 280), window.innerWidth - 24),
			left = Math.max(12, Math.min(bounds.left, window.innerWidth - width - 12)),
			below = window.innerHeight - bounds.bottom - 12,
			above = bounds.top - 12,
			opensUp = below < maxMenuHeight && above > below,
			edge = opensUp
				? `bottom: ${window.innerHeight - bounds.top + 4}px;`
				: `top: ${bounds.bottom + 4}px;`
		listStyle = `${edge} left: ${left}px; width: ${width}px; --place-menu-space: ${Math.max(28, opensUp ? above : below)}px; --menu-origin: ${opensUp ? "bottom" : "top"} left;`
	}

	function accept(place: GamePlace | undefined): void {
		if (!place) return
		store.launchInput.placeId = String(place.placeId)
		open = false
	}

	function showSuggestions(): void {
		active = 0
		navigated = false
		open = true
		position()
		void tick().then(() => {
			if (list) list.scrollTop = 0
			return undefined
		})
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (
			event.ctrlKey &&
			!event.altKey &&
			!event.metaKey &&
			event.code === "Space"
		) {
			event.preventDefault()
			showSuggestions()
			return
		}
		if (!shown) return
		if (event.key === "Escape") {
			event.preventDefault()
			event.stopPropagation()
			open = false
			return
		}
		if (suggestions.length === 0) return
		switch (event.key) {
			case "ArrowDown":
			case "ArrowUp":
				event.preventDefault()
				navigated = true
				active =
					(activeIndex +
						(event.key === "ArrowDown" ? 1 : -1) +
						suggestions.length) %
					suggestions.length
				void tick().then(() =>
					(
						list?.children[activeIndex] as HTMLElement | undefined
					)?.scrollIntoView({
						block: "nearest",
					}),
				)
				break
			case "Tab":
				if (event.shiftKey) {
					open = false
					return
				}
				event.preventDefault()
				accept(suggestions[activeIndex])
				break
			case "Enter":
				if (!navigated) return
				event.preventDefault()
				accept(suggestions[activeIndex])
				break
		}
	}
</script>

<svelte:window
	onresize={() => (open = false)}
	onclick={(event) => {
		if (
			open &&
			event.target instanceof Node &&
			event.target !== input &&
			!list?.contains(event.target)
		)
			open = false
	}}
	onfocusin={(event) => {
		if (
			open &&
			event.target instanceof Node &&
			event.target !== input &&
			!list?.contains(event.target)
		)
			open = false
	}}
	onscrollcapture={(event) => {
		if (open && !(event.target instanceof Node && list?.contains(event.target)))
			open = false
	}} />

<label
	><span>Place ID</span>
	<input
		bind:this={input}
		bind:value={store.launchInput.placeId}
		type="text"
		placeholder="ID or favorited game"
		role="combobox"
		autocomplete="off"
		spellcheck="false"
		aria-autocomplete="list"
		aria-keyshortcuts="Control+Space"
		aria-expanded={shown}
		aria-controls={shown ? listID : undefined}
		aria-activedescendant={shown && suggestions.length
			? `${listID}-${activeIndex}`
			: undefined}
		required
		disabled={store.launching}
		oninput={showSuggestions}
		onkeydown={handleKeydown} />
</label>
{#if shown}
	<div
		class="place-suggestions"
		bind:this={list}
		id={listID}
		role="listbox"
		aria-label="Favorite games"
		style={listStyle}
		in:menuIn
		out:menuOut>
		{#each suggestions as place, index (place.placeId)}
			<div
				class="place-suggestion"
				id={`${listID}-${index}`}
				role="option"
				tabindex="-1"
				aria-selected={index === activeIndex}
				onmousemove={() => (active = index)}
				onmousedown={(event) => {
					event.preventDefault()
					accept(place)
				}}>
				<GameIcon url={place.iconUrl} />
				<span class="place-suggestion-copy">
					<span>{place.nickname || place.name}</span>
					<small>{place.placeId}</small>
				</span>
				{#if index === activeIndex}<kbd>Tab</kbd>{/if}
			</div>
		{:else}
			<div class="place-suggestion-empty" role="status">
				{games.loadingFavorites
					? "Loading favorite games…"
					: games.favoritesLoaded
						? "No favorite games yet"
						: "Favorite games unavailable"}
			</div>
		{/each}
	</div>
{/if}
