<script lang="ts">
	import { onDestroy, onMount, tick } from "svelte"
	import AlertTriangle from "@lucide/svelte/icons/triangle-alert"
	import BadgeCheck from "@lucide/svelte/icons/badge-check"
	import ArrowRight from "@lucide/svelte/icons/arrow-right"
	import ChevronLeft from "@lucide/svelte/icons/chevron-left"
	import ChevronRight from "@lucide/svelte/icons/chevron-right"
	import Gamepad2 from "@lucide/svelte/icons/gamepad-2"
	import GripVertical from "@lucide/svelte/icons/grip-vertical"
	import LoaderCircle from "@lucide/svelte/icons/loader-circle"
	import Search from "@lucide/svelte/icons/search"
	import Star from "@lucide/svelte/icons/star"
	import X from "@lucide/svelte/icons/x"
	import SidebarResizer from "../layout/SidebarResizer.svelte"
	import OperationError from "../shared/OperationError.svelte"
	import { menuIn, menuOut } from "../shared/presence"
	import GameContextMenu from "./GameContextMenu.svelte"
	import GameDetails from "./GameDetails.svelte"
	import GameServersPage from "./GameServersPage.svelte"
	import GameIcon from "./GameIcon.svelte"
	import { creatorByline, type GamesStore } from "./games-store.svelte"

	let {
		store,
		onFillLaunch,
	}: {
		store: GamesStore
		onFillLaunch: (placeId: number, jobId?: string) => void
	} = $props()
	const favoritesMode = $derived(store.mode === "favorites"),
		count = $derived(store.places.length),
		hasSearch = $derived(!favoritesMode && !!store.searchedText),
		hasMore = $derived(
			!favoritesMode && store.results.length > 0 && !!store.nextPageToken,
		)
	let searchInput = $state<HTMLInputElement | undefined>(undefined),
		grabbedPlaceId: number | null = null,
		draggedPlaceId = $state<number | null>(null),
		dropTarget = $state<{ placeId: number; after: boolean } | null>(null)

	function grab(event: PointerEvent, placeId: number): void {
		grabbedPlaceId =
			event.target instanceof Element && event.target.closest(".game-drag-handle")
				? placeId
				: null
	}

	function startDrag(event: DragEvent, placeId: number): void {
		if (!store.canReorder || grabbedPlaceId !== placeId) {
			event.preventDefault()
			return
		}
		draggedPlaceId = placeId
		dropTarget = null
		store.closeMenu()
		if (event.dataTransfer) {
			event.dataTransfer.effectAllowed = "move"
			event.dataTransfer.setData("text/plain", String(placeId))
		}
	}

	function updateDropTarget(event: DragEvent, placeId: number): void {
		if (draggedPlaceId === null || draggedPlaceId === placeId) return
		if (!(event.currentTarget instanceof HTMLElement)) return
		event.preventDefault()
		if (event.dataTransfer) event.dataTransfer.dropEffect = "move"
		const bounds = event.currentTarget.getBoundingClientRect()
		dropTarget = {
			placeId,
			after: event.clientY >= bounds.top + bounds.height / 2,
		}
	}

	function finishDrag(): void {
		grabbedPlaceId = draggedPlaceId = null
		dropTarget = null
	}

	function dropFavorite(event: DragEvent, placeId: number): void {
		event.preventDefault()
		const sourceId = draggedPlaceId,
			after = dropTarget?.placeId === placeId && dropTarget.after
		finishDrag()
		if (sourceId !== null && sourceId !== placeId) {
			void store.moveFavorite(sourceId, placeId, after)
		}
	}

	async function closeServers(): Promise<void> {
		store.detailsPage = "details"
		await tick()
		document.querySelector<HTMLButtonElement>("[data-game-servers]")?.focus()
	}

	onMount(() => store.revalidate())
	onDestroy(() => store.closeMenu())

	function loadMoreWhenVisible(node: HTMLElement, _pageToken: string) {
		const observer = new IntersectionObserver(
			(entries) => {
				if (
					entries.some((entry) => entry.isIntersecting) &&
					!store.searching &&
					!store.loadMoreFailed
				) {
					void store.search(true)
				}
			},
			{ root: node.parentElement, rootMargin: "0px 0px 160px 0px" },
		)
		observer.observe(node)
		return {
			update() {
				observer.unobserve(node)
				observer.observe(node)
			},
			destroy: () => observer.disconnect(),
		}
	}
</script>

<main class="games-page" aria-label="Games">
	<OperationError {store} />
	<div
		class="workbench"
		style={`--sidebar-width: ${store.sidebarCollapsed ? 42 : store.sidebarWidth}px;`}>
		<aside
			class="account-explorer"
			class:collapsed={store.sidebarCollapsed}
			aria-label="Game browser">
			{#if store.sidebarCollapsed}
				<div class="collapsed-sidebar">
					<button
						type="button"
						aria-label="Expand games sidebar"
						data-tooltip="Expand the games sidebar"
						data-tooltip-side="right"
						onclick={() => (store.sidebarCollapsed = false)}
						><ChevronRight size={16} /></button>
					<Gamepad2 size={16} aria-hidden="true" />
				</div>
			{:else}
				<div class="explorer-controls">
					<div class="account-title">
						<h1>Games</h1>
						<div class="account-title-actions">
							<button
								type="button"
								aria-label="Collapse games sidebar"
								data-tooltip="Collapse the games sidebar"
								data-tooltip-side="bottom-end"
								onclick={() => (store.sidebarCollapsed = true)}
								><ChevronLeft size={15} /></button>
						</div>
					</div>
					<form
						novalidate
						class="account-filters"
						role="search"
						onsubmit={(event) => {
							event.preventDefault()
							if (!favoritesMode && store.query.trim())
								void store.search()
							else searchInput?.focus()
						}}>
						<div class="search-field">
							<Search size={14} aria-hidden="true" /><input
								type="text"
								placeholder={favoritesMode
									? "Search favorites"
									: "Search Roblox"}
								aria-label={favoritesMode
									? "Filter favorite places by nickname, name, or creator"
									: "Search Roblox games by name, by place ID with id:, or by universe ID with universe:"}
								maxlength={100}
								bind:this={searchInput}
								bind:value={
									() =>
										favoritesMode
											? store.favoritesQuery
											: store.query,
									(value) => {
										if (favoritesMode) store.favoritesQuery = value
										else store.query = value
									}
								} />{#if favoritesMode ? store.favoritesQuery : store.query}<button
									class="search-clear"
									type="button"
									aria-label={favoritesMode
										? "Clear favorites filter"
										: "Clear game search"}
									onclick={() => {
										if (favoritesMode) store.favoritesQuery = ""
										else store.query = ""
										searchInput?.focus()
									}}><X size={12} /></button
								>{/if}
						</div>
						{#if !favoritesMode}
							<div class="tag-filter-host">
								<button
									class="tag-filter-trigger"
									type="submit"
									aria-label="Search Roblox"
									aria-busy={store.searching}
									data-tooltip="Search Roblox"
									data-tooltip-side="bottom-end">
									{#if store.searching && !store.results.length}<LoaderCircle
											class="spinner"
											size={14}
											aria-hidden="true" />{:else}<ArrowRight
											size={14}
											aria-hidden="true" />{/if}
								</button>
							</div>
						{/if}
					</form>
					<div class="games-views" role="group" aria-label="Show games">
						<button
							type="button"
							aria-pressed={favoritesMode}
							onclick={() => store.show("favorites")}>
							<Star size={12} aria-hidden="true" />Favorites
							<span>{store.favorites.length}</span>
						</button>
						<button
							type="button"
							aria-pressed={!favoritesMode}
							onclick={() => store.show("search")}>
							<Search size={12} aria-hidden="true" />Search
						</button>
					</div>
				</div>
				<div class="account-list-region">
					<div
						class="games-list"
						class:has-search-results={hasSearch}
						aria-label={favoritesMode
							? "Favorite places"
							: "Search results"}
						aria-busy={favoritesMode
							? store.loadingFavorites
							: store.searching}>
						{#each store.places as place (place.placeId)}
							{@const active = store.selected?.placeId === place.placeId}
							<button
								class="game-row"
								class:active
								class:menu-open={store.menu?.place.placeId ===
									place.placeId}
								class:dragging={draggedPlaceId === place.placeId}
								class:drop-before={dropTarget?.placeId ===
									place.placeId && !dropTarget.after}
								class:drop-after={dropTarget?.placeId ===
									place.placeId && dropTarget.after}
								class:reorderable={favoritesMode}
								draggable={favoritesMode && store.canReorder}
								onpointerdown={(event) => grab(event, place.placeId)}
								ondragstart={(event) => startDrag(event, place.placeId)}
								ondragend={finishDrag}
								ondragover={(event) =>
									updateDropTarget(event, place.placeId)}
								ondrop={(event) => dropFavorite(event, place.placeId)}
								type="button"
								aria-current={active ? "true" : undefined}
								aria-haspopup="menu"
								onclick={() => store.select(place)}
								oncontextmenu={(event) => store.openMenu(event, place)}
								onkeydown={(event) => {
									if (
										event.key === "ContextMenu" ||
										(event.shiftKey && event.key === "F10")
									)
										store.openMenu(event, place)
								}}>
								{#if favoritesMode}<span
										class="game-drag-handle"
										class:disabled={!store.canReorder}
										aria-hidden="true"
										><GripVertical size={12} /></span
									>{/if}
								<GameIcon url={place.iconUrl} />
								<span class="game-row-text">
									<strong class="game-display-name"
										><span>{store.displayName(place)}</span
										>{#if store.isFavorite(place.placeId)}<Star
												class="favorite-name-star"
												size={11}
												fill="currentColor"
												aria-label="Favorite" />{/if}</strong>
									<span
										>{creatorByline(
											place,
										)}{#if place.creatorVerified}<BadgeCheck
												class="creator-verified"
												size={12}
												aria-label="Verified creator" />{/if}</span>
								</span>
							</button>
						{/each}
						{#if hasMore}
							<div
								class="games-list-footer"
								use:loadMoreWhenVisible={store.nextPageToken}>
								{#if store.loadMoreFailed}
									<button
										class="control-button games-text-button"
										type="button"
										onclick={() => void store.search(true)}
										>Load more results</button>
								{:else}
									<LoaderCircle
										class="spinner"
										size={14}
										aria-label="Loading more results" />
								{/if}
							</div>
						{/if}
					</div>
					{#if hasSearch}
						<div class="account-selection-actions">
							<button
								class="control-button"
								type="button"
								in:menuIn
								out:menuOut
								onclick={() => {
									store.clearSearch()
									searchInput?.focus()
								}}>
								<X size={12} aria-hidden="true" />
								<span>Clear results</span>
							</button>
						</div>
					{/if}
				</div>
			{/if}
		</aside>
		<SidebarResizer
			bind:width={store.sidebarWidth}
			collapsed={store.sidebarCollapsed}
			name="games" />
		{#if store.selected}
			{#key store.selected.placeId}
				{#if store.detailsPage === "servers"}
					<GameServersPage
						{store}
						onBack={() => void closeServers()}
						{onFillLaunch} />
				{:else}
					<GameDetails {store} />
				{/if}
			{/key}
		{:else}
			<div class="games-empty" role="status">
				{#if favoritesMode}
					{#if store.loadingFavorites}<LoaderCircle
							class="spinner"
							size={24}
							aria-hidden="true" />
						<h2>Loading favorites…</h2>
					{:else if !store.favoritesLoaded}<AlertTriangle
							size={24}
							aria-hidden="true" />
						<h2>Favorites are unavailable</h2>
						<button
							class="control-button games-text-button"
							type="button"
							onclick={() => void store.loadFavorites()}
							>Try again</button>
					{:else if !count && store.favorites.length}<Search
							size={24}
							aria-hidden="true" />
						<h2>No matching favorites</h2>
						<p>Try a different nickname, name, or creator.</p>
					{:else if !count}<Star size={24} aria-hidden="true" />
						<h2>No favorite places yet</h2>
						<p>Right-click a game and add it to your favorites.</p>
					{:else}<Star size={24} aria-hidden="true" />
						<h2>Select a favorite place</h2>
						<p>Choose a place to see its details.</p>{/if}
				{:else if store.searching}<LoaderCircle
						class="spinner"
						size={24}
						aria-hidden="true" />
					<h2>Searching Roblox…</h2>
				{:else if store.searchError}<AlertTriangle
						size={24}
						aria-hidden="true" />
					<h2>Search is unavailable</h2>
					<p>{store.searchError}</p>
				{:else if store.searchedText && !count}<Search
						size={24}
						aria-hidden="true" />
					<h2>No games found</h2>
					<p>Try a different name or place ID.</p>
				{:else if count}<Gamepad2 size={24} aria-hidden="true" />
					<h2>Select a game</h2>
					<p>Choose a game to see its details.</p>
				{:else}<Gamepad2 size={24} aria-hidden="true" />
					<h2>Search Roblox games</h2>
					<p>
						Enter a game name, a place ID like id:1818, or a universe ID
						like universe:13058 to list its places.
					</p>{/if}
			</div>
		{/if}
	</div>
	<footer class="status-bar" aria-label="Game list">
		<span
			>{store.favorites.length}
			{store.favorites.length === 1 ? "favorite place" : "favorite places"}</span>
		{#if store.selected}
			<div>
				<span>Place ID: {store.selected.placeId}</span>
				<span>Universe ID: {store.selected.universeId}</span>
			</div>
		{/if}
	</footer>
</main>

{#if store.menu}
	<GameContextMenu {store} menu={store.menu} {onFillLaunch} />
{/if}
