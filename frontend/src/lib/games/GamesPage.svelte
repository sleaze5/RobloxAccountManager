<script lang="ts">
	import { onDestroy, onMount, tick } from "svelte"
	import arrowRightIcon from "@phosphor-icons/core/regular/arrow-right.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import dotsSixVerticalIcon from "@phosphor-icons/core/regular/dots-six-vertical.svg?raw"
	import gameControllerIcon from "@phosphor-icons/core/regular/game-controller.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import starIcon from "@phosphor-icons/core/regular/star.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import SkeletonList from "../shared/SkeletonList.svelte"
	import SkeletonMediaRow from "../shared/SkeletonMediaRow.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import WorkbenchSidebar from "../layout/WorkbenchSidebar.svelte"
	import type { WorkspaceState } from "../layout/workspace-state.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import { isMoveShortcut, moveListFocus } from "../shared/move-shortcut"
	import OperationError from "../shared/OperationError.svelte"
	import { menuIn, menuOut } from "../shared/presence"
	import GameContextMenu from "./GameContextMenu.svelte"
	import GameDetails from "./GameDetails.svelte"
	import GameServersPage from "./GameServersPage.svelte"
	import GameIcon from "./GameIcon.svelte"
	import { creatorByline, type GamesStore } from "./games-store.svelte"

	let {
		store,
		accounts,
		notifications,
		workspace,
		onFillLaunch,
		onFillLaunchLink,
	}: {
		store: GamesStore
		accounts: AccountStore
		notifications: NotificationCenter
		workspace: WorkspaceState
		onFillLaunch: (placeId: number, jobId?: string) => void
		onFillLaunchLink: (link: string) => void
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
		dropTarget = $state<{ placeId: number; after: boolean } | null>(null),
		moveAnnouncement = $state("")

	async function moveWithKeyboard(
		event: KeyboardEvent,
		placeId: number,
	): Promise<void> {
		if (!favoritesMode || !isMoveShortcut(event)) return
		event.preventDefault()
		if (!store.canReorder) return
		const button = event.currentTarget as HTMLElement,
			down = event.key === "ArrowDown",
			index = store.favorites.findIndex((place) => place.placeId === placeId),
			target = store.favorites[index + (down ? 1 : -1)]
		if (!target) return
		const moving = store.moveFavorite(placeId, target.placeId, down)
		await tick()
		button.focus()
		await moving
		const position =
			store.favorites.findIndex((place) => place.placeId === placeId) + 1
		if (position > 0) {
			moveAnnouncement = `Moved to position ${position} of ${store.favorites.length}`
		}
	}

	function grab(event: PointerEvent, placeId: number): void {
		grabbedPlaceId =
			event.target instanceof Element &&
			event.target.closest("[data-drag-handle]")
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

	async function searchUniverse(universeId: number): Promise<void> {
		void store.searchFor(`universe:${universeId}`)
		await tick()
		searchInput?.focus()
	}

	async function searchGames(): Promise<void> {
		store.show("search")
		await tick()
		searchInput?.focus()
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

<main class="flex min-h-0 flex-1 flex-col bg-canvas" aria-label="Games">
	<OperationError {store} />
	<div
		class="flex min-h-0 flex-1"
		style={`--sidebar-width: ${store.sidebarCollapsed ? 44 : store.sidebarWidth}px;`}>
		<WorkbenchSidebar
			label="Game browser"
			name="games"
			icon={gameControllerIcon}
			count={favoritesMode ? store.favorites.length : undefined}
			bind:collapsed={store.sidebarCollapsed}
			bind:width={store.sidebarWidth}>
			{#snippet title()}Games
				{#if favoritesMode}<span
						class="font-mono text-2xs font-normal text-fg-faint"
						>({store.favorites.length})</span
					>{/if}{/snippet}
			{#snippet tools()}
				<form
					novalidate
					class="flex items-center gap-1"
					role="search"
					onsubmit={(event) => {
						event.preventDefault()
						if (!favoritesMode && store.query.trim()) void store.search()
						else searchInput?.focus()
					}}>
					<div class="search-field">
						<Icon
							svg={magnifyingGlassIcon}
							size={16}
							aria-hidden="true" /><input
							type="text"
							placeholder={favoritesMode
								? "Search favorites"
								: "Search Roblox"}
							aria-label={favoritesMode
								? "Filter favorite places by nickname, name, or creator"
								: "Search Roblox games by name, by place ID with id:, or by universe ID with universe:"}
							maxlength={100}
							bind:this={searchInput}
							data-games-search
							bind:value={
								() =>
									favoritesMode ? store.favoritesQuery : store.query,
								(value) => {
									if (favoritesMode) store.favoritesQuery = value
									else store.query = value
								}
							} />{#if favoritesMode ? store.favoritesQuery : store.query}<button
								type="button"
								aria-label={favoritesMode
									? "Clear favorites filter"
									: "Clear game search"}
								onclick={() => {
									if (favoritesMode) store.favoritesQuery = ""
									else store.query = ""
									searchInput?.focus()
								}}
								><Icon
									svg={xIcon}
									size={14}
									aria-hidden="true" /></button
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
								{#if store.searching && !store.results.length}<Icon
										svg={circleNotchIcon}
										class="spinner"
										size={16}
										aria-hidden="true" />{:else}<Icon
										svg={arrowRightIcon}
										size={16}
										aria-hidden="true" />{/if}
							</button>
						</div>
					{/if}
				</form>
				<div class="segmented" role="group" aria-label="Show games">
					<button
						type="button"
						aria-pressed={favoritesMode}
						onclick={() => store.show("favorites")}>
						<Icon svg={starIcon} size={14} aria-hidden="true" />Favorites
					</button>
					<button
						type="button"
						aria-pressed={!favoritesMode}
						onclick={() => store.show("search")}>
						<Icon
							svg={magnifyingGlassIcon}
							size={14}
							aria-hidden="true" />Search
					</button>
				</div>
			{/snippet}
			<div class="relative grid min-h-0 flex-1 grid-rows-1">
				<div
					class={[
						"min-h-0 scrollbar-stable overflow-y-auto",
						hasSearch && "pb-12",
					]}
					aria-label={favoritesMode ? "Favorite places" : "Search results"}
					aria-busy={favoritesMode
						? store.loadingFavorites
						: store.searching}>
					{#if store.places.length === 0 && (favoritesMode ? store.loadingFavorites : store.searching)}
						<SkeletonList
							label={favoritesMode
								? "Loading favorites"
								: "Searching games"}>
							{#snippet row(index)}<SkeletonMediaRow {index} />{/snippet}
						</SkeletonList>
					{/if}
					{#each store.places as place (place.placeId)}
						{@const active = store.selected?.placeId === place.placeId}
						<button
							class={[
								"list-row relative flex h-12 w-full items-center gap-2 border-b border-line-subtle text-left",
								favoritesMode ? "pr-3 pl-2" : "px-3",
								draggedPlaceId === place.placeId && "opacity-45",
								dropTarget?.placeId === place.placeId &&
									(dropTarget.after
										? "shadow-drop-after"
										: "shadow-drop-before"),
							]}
							data-selected={active || undefined}
							data-open={store.menu?.place.placeId === place.placeId ||
								undefined}
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
							data-list-item
							aria-keyshortcuts={favoritesMode
								? "Control+Shift+ArrowUp Control+Shift+ArrowDown"
								: undefined}
							onclick={() => store.select(place)}
							oncontextmenu={(event) => store.openMenu(event, place)}
							onkeydown={(event) => {
								if (
									event.key === "ContextMenu" ||
									(event.shiftKey && event.key === "F10")
								)
									store.openMenu(event, place)
								void moveWithKeyboard(event, place.placeId)
								moveListFocus(event)
							}}>
							{#if favoritesMode}<span
									class={[
										"grid h-7 w-3 shrink-0 place-items-center text-fg-faint",
										store.canReorder
											? "cursor-grab active:cursor-grabbing"
											: "cursor-default",
									]}
									data-drag-handle
									aria-hidden="true"
									><Icon
										svg={dotsSixVerticalIcon}
										size={14}
										aria-hidden="true" /></span
								>{/if}
							<GameIcon url={place.iconUrl} />
							<span class="flex min-w-0 flex-col leading-snug">
								<strong
									class="inline-flex min-w-0 items-center gap-1 text-xs font-semibold text-fg"
									><span class="truncate"
										>{store.displayName(place)}</span
									>{#if store.isFavorite(place.placeId)}<Icon
											svg={starFillIcon}
											class="shrink-0 animate-pop text-accent-strong"
											size={12}
											aria-label="Favorite" />{/if}</strong>
								<span class="truncate text-2xs text-fg-muted"
									>{creatorByline(
										place,
									)}{#if place.creatorVerified}<Icon
											svg={sealCheckIcon}
											class="relative top-0.5 ml-1 inline-block"
											size={14}
											aria-label="Verified creator" />{/if}</span>
							</span>
						</button>
					{/each}
					{#if hasMore}
						<div
							class="flex min-h-12 items-center justify-center text-fg-faint"
							use:loadMoreWhenVisible={store.nextPageToken}>
							{#if store.loadMoreFailed}
								<button
									class="button"
									type="button"
									onclick={() => void store.search(true)}
									>Load more results</button>
							{:else}
								<Icon
									svg={circleNotchIcon}
									class="spinner"
									size={16}
									aria-label="Loading more results" />
							{/if}
						</div>
					{/if}
					<span class="sr-only" aria-live="polite">{moveAnnouncement}</span>
				</div>
				{#if hasSearch}
					<div
						class="pointer-events-none absolute inset-x-0 bottom-2 flex justify-center">
						<button
							class="pointer-events-auto button bg-raised shadow-float hover:bg-raised-hover"
							type="button"
							in:menuIn
							out:menuOut
							onclick={() => {
								store.clearSearch()
								searchInput?.focus()
							}}>
							<Icon svg={xIcon} size={14} aria-hidden="true" />
							<span>Clear results</span>
						</button>
					</div>
				{/if}
			</div>
		</WorkbenchSidebar>
		<div class="grid min-w-0 flex-1 grid-rows-1">
			{#if store.selected}
				{#key store.selected.placeId}
					{#if store.detailsPage === "servers"}
						<GameServersPage
							{store}
							{accounts}
							{notifications}
							onBack={() => void closeServers()}
							{onFillLaunch}
							{onFillLaunchLink} />
					{:else}
						<GameDetails
							{store}
							{workspace}
							onSearchUniverse={(universeId) =>
								void searchUniverse(universeId)} />
					{/if}
				{/key}
			{:else}
				<div class="empty-state" role="status">
					{#if favoritesMode}
						{#if store.loadingFavorites}<Icon
								svg={circleNotchIcon}
								class="spinner"
								size={28}
								aria-hidden="true" />
							<h2>Loading favorites…</h2>
						{:else if !store.favoritesLoaded}<Icon
								svg={warningIcon}
								size={28}
								aria-hidden="true" />
							<h2>Favorites are unavailable</h2>
							<button
								class="button"
								type="button"
								onclick={() => void store.loadFavorites()}
								>Try again</button>
						{:else if !count && store.favorites.length}<Icon
								svg={magnifyingGlassIcon}
								size={28}
								aria-hidden="true" />
							<h2>No matching favorites</h2>
							<p>Try a different nickname, name, or creator.</p>
						{:else if !count}<Icon
								svg={starIcon}
								size={28}
								aria-hidden="true" />
							<h2>No favorite places yet</h2>
							<p>Search for a game, then add it to your favorites.</p>
							<button
								class="button"
								type="button"
								onclick={() => void searchGames()}
								><Icon
									svg={magnifyingGlassIcon}
									size={16}
									aria-hidden="true" />Search games</button>
						{:else}<Icon svg={starIcon} size={28} aria-hidden="true" />
							<h2>Select a favorite place</h2>
							<p>Choose a place to see its details.</p>{/if}
					{:else if store.searching}<Icon
							svg={circleNotchIcon}
							class="spinner"
							size={28}
							aria-hidden="true" />
						<h2>Searching Roblox…</h2>
					{:else if store.searchError}<Icon
							svg={warningIcon}
							size={28}
							aria-hidden="true" />
						<h2>Search is unavailable</h2>
						<p>{store.searchError}</p>
					{:else if store.searchedText && !count}<Icon
							svg={magnifyingGlassIcon}
							size={28}
							aria-hidden="true" />
						<h2>No games found</h2>
						<p>Try a different name or place ID.</p>
					{:else if count}<Icon
							svg={gameControllerIcon}
							size={28}
							aria-hidden="true" />
						<h2>Select a game</h2>
						<p>Choose a game to see its details.</p>
					{:else}<Icon
							svg={gameControllerIcon}
							size={28}
							aria-hidden="true" />
						<h2>Search Roblox games</h2>
						<p>
							Enter a game name, a place ID like id:1818, or a universe ID
							like universe:13058 to list its places.
						</p>{/if}
				</div>
			{/if}
		</div>
	</div>
	<footer
		class="flex h-6 shrink-0 items-center justify-between border-t border-line-subtle bg-surface pl-3 text-2xs text-fg-faint"
		aria-label="Game list">
		{#if store.selected}
			<span
				>Place ID: <span class="font-mono">{store.selected.placeId}</span
				></span>
			<div class="flex items-center gap-4 pr-3">
				<span
					>Universe ID: <span class="font-mono"
						>{store.selected.universeId}</span
					></span>
			</div>
		{/if}
	</footer>
</main>

{#if store.menu}
	<GameContextMenu {store} menu={store.menu} {onFillLaunch} />
{/if}
