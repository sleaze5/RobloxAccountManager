<script lang="ts">
	import LoadMore from "../shared/LoadMore.svelte"
	import { tick, untrack } from "svelte"
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
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import linkIcon from "@phosphor-icons/core/regular/link.svg?raw"
	import textTIcon from "@phosphor-icons/core/regular/text-t.svg?raw"
	import type { GamePlace } from "../backend/bridge"
	import type { MenuAction } from "../shared/AnchoredActionMenu.svelte"
	import CheckboxMark from "../shared/CheckboxMark.svelte"
	import ConfirmationDialog from "../shared/ConfirmationDialog.svelte"
	import Icon from "../shared/Icon.svelte"
	import SelectionActionsMenu from "../shared/SelectionActionsMenu.svelte"
	import SelectionModeButton from "../shared/SelectionModeButton.svelte"
	import SelectionToolbar from "../shared/SelectionToolbar.svelte"
	import SkeletonList from "../shared/SkeletonList.svelte"
	import SkeletonMediaRow from "../shared/SkeletonMediaRow.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import AccountSwitcher from "../layout/AccountSwitcher.svelte"
	import { getPageVisibility } from "../layout/page-visibility"
	import WorkbenchSidebar from "../layout/WorkbenchSidebar.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import { isMoveShortcut, moveListFocus } from "../shared/move-shortcut"
	import { isMultiSelectGesture } from "../shared/list-selection.svelte"
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import { menuIn, menuOut } from "../shared/presence"
	import GameContextMenu from "./GameContextMenu.svelte"
	import GameBadgesPage from "./GameBadgesPage.svelte"
	import GameDetails from "./GameDetails.svelte"
	import GamePassesPage from "./GamePassesPage.svelte"
	import GameServersPage from "./GameServersPage.svelte"
	import GameIcon from "./GameIcon.svelte"
	import {
		creatorByline,
		type GameDetailsPage,
		type GamesStore,
	} from "./games-store.svelte"
	import { openPageLabel, setProfilePages } from "../shared/profile-pages"
	import type { ProfilePages } from "../shared/profile-pages"
	import StatusTrail from "../layout/StatusTrail.svelte"

	let {
		store,
		accounts,
		notifications,
		onFillLaunch,
		onFillLaunchLink,
	}: {
		store: GamesStore
		accounts: AccountStore
		notifications: NotificationCenter
		onFillLaunch: (placeId: number, serverId?: string) => void
		onFillLaunchLink: (link: string) => void
	} = $props()
	const favoritesMode = $derived(store.mode === "favorites"),
		count = $derived(store.places.length),
		hasSearch = $derived(!favoritesMode && !!store.searchedText),
		hasMore = $derived(
			!favoritesMode && store.results.length > 0 && !!store.nextPageToken,
		),
		tabStopPlaceId = $derived(
			(
				store.places.find(
					(place) => place.placeId === store.selected?.placeId,
				) ?? store.places[0]
			)?.placeId,
		),
		selection = $derived(store.selection),
		shownPlaceIds = $derived(store.places.map((place) => place.placeId)),
		selectedPlaces = $derived(store.selectedPlaces),
		selectedFavoriteCount = $derived(
			selectedPlaces.filter((place) => store.isFavorite(place.placeId)).length,
		),
		isFavoriteBusy = $derived(
			store.savingFavorite || store.savingNickname || !store.favoritesLoaded,
		),
		selectionActions = $derived<MenuAction[]>([
			{
				label: "Add to favorites",
				icon: starIcon,
				disabled:
					isFavoriteBusy || selectedFavoriteCount === selectedPlaces.length,
				tooltip:
					selectedFavoriteCount === selectedPlaces.length
						? "Every selected game is a favorite"
						: undefined,
				onSelect: () => void store.setSelectedFavorites(true),
			},
			...(
				[
					{ field: "name", label: "Names", icon: textTIcon },
					{ field: "placeId", label: "Place IDs", icon: hashIcon },
					{ field: "universeId", label: "Universe IDs", icon: hashIcon },
					{ field: "link", label: "Game links", icon: linkIcon },
				] as const
			).map((copyOption) => ({
				label: copyOption.label,
				icon: copyOption.icon,
				section: "Copy, one per line",
				accessibleLabel: `Copy ${copyOption.label.toLowerCase()}`,
				onSelect: () => void store.copySelectedField(copyOption.field),
			})),
			...(selectedFavoriteCount > 0
				? [
						{
							label: "Remove from favorites",
							icon: starFillIcon,
							tone: "danger" as const,
							disabled: isFavoriteBusy,
							onSelect: () => (isConfirmingFavoriteRemoval = true),
						},
					]
				: []),
		])
	const pageVisibility = getPageVisibility()
	let searchInput = $state<HTMLInputElement | undefined>(undefined),
		grabbedPlaceId: number | null = null,
		draggedPlaceId = $state<number | null>(null),
		dropTarget = $state<{ placeId: number; after: boolean } | null>(null),
		moveAnnouncement = $state(""),
		isConfirmingFavoriteRemoval = $state(false)

	// A selected place that the list no longer has, such as after a new
	// search, leaves the selection.
	$effect(() => {
		const remainingPlaceIds = (favoritesMode ? store.favorites : store.results).map(
			(place) => place.placeId,
		)
		untrack(() => selection.keepOnly(remainingPlaceIds))
	})

	async function removeSelectedFavorites(): Promise<void> {
		await store.setSelectedFavorites(false)
		isConfirmingFavoriteRemoval = false
	}

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

	function selectPlace(event: MouseEvent | KeyboardEvent, place: GamePlace): void {
		if (!isMultiSelectGesture(event)) {
			store.select(place)
			return
		}
		const shownPlaceId = store.selected?.placeId
		store.setSelecting(true)
		selection.add(
			shownPlaceId === undefined
				? [place.placeId]
				: [shownPlaceId, place.placeId],
		)
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

	const gamePages: GameDetailsPage[] = ["servers", "badges", "gamePasses"]

	// Badges and game passes belong to the universe, so a subplace has only
	// its servers. They wait for the details, which tell whether the place
	// is a subplace.
	const profilePages: ProfilePages = {
		get label() {
			return `${store.selected ? store.displayName(store.selected) : "Game"} pages`
		},
		get tabs() {
			const universeUnavailableReason = store.game ? "" : "Loading game details…"
			return [
				{ id: "servers", label: "Servers" },
				...(store.game?.rootPlace
					? []
					: [
							{
								id: "badges",
								label: "Badges",
								unavailableReason: universeUnavailableReason,
							},
							{
								id: "gamePasses",
								label: "Passes and products",
								unavailableReason: universeUnavailableReason,
							},
						]),
			]
		},
		get currentPage() {
			return store.detailsPage === "details" ? "" : store.detailsPage
		},
		openPage: (page) => {
			const gamePage = gamePages.find((knownPage) => knownPage === page)
			if (!gamePage) return
			store.closeMenu()
			store.detailsPage = gamePage
		},
		openOverview: () => void returnToDetails(),
	}
	setProfilePages(profilePages)

	async function returnToDetails(): Promise<void> {
		const closedPage = store.detailsPage
		store.detailsPage = "details"
		await tick()
		document
			.querySelector<HTMLButtonElement>(`[data-subpage-button="${closedPage}"]`)
			?.focus()
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

	$effect(() => {
		if (!pageVisibility.isShown) return
		untrack(() => store.revalidate())
		return () => store.closeMenu()
	})

	notifyErrors("Action not completed", () => store.error)
	notifyErrors("Search not completed", () => store.searchError)
</script>

{#snippet placeSummary(place: GamePlace)}
	<GameIcon url={place.iconUrl} />
	<span class="flex min-w-0 flex-col leading-snug">
		<strong
			class="inline-flex min-w-0 items-center gap-1 text-xs font-semibold text-fg"
			><span class="truncate">{store.displayName(place)}</span
			>{#if store.isFavorite(place.placeId)}<Icon
					svg={starFillIcon}
					class="shrink-0 animate-pop text-accent-strong"
					size={12}
					aria-label="Favorite" />{/if}</strong>
		<span class="flex min-w-0 items-center gap-1 text-2xs text-fg-muted"
			><span class="truncate">{creatorByline(place)}</span
			>{#if place.creatorVerified}<Icon
					svg={sealCheckIcon}
					class="shrink-0"
					size={12}
					aria-label="Verified creator" />{/if}</span>
	</span>
{/snippet}

<main class="flex min-h-0 flex-1 flex-col bg-canvas" aria-label="Games">
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
			{#snippet actions()}
				<SelectionModeButton
					noun="games"
					isSelecting={selection.isSelecting}
					disabled={store.places.length === 0 && !selection.isSelecting}
					onToggle={() => store.setSelecting(!selection.isSelecting)} />
			{/snippet}
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
								aria-label="Clear search"
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
						"relative min-h-0 scrollbar-stable overflow-y-auto",
						(hasSearch || selection.isSelecting) && "pb-12",
					]}
					role="group"
					aria-label={favoritesMode ? "Favorite places" : "Search results"}
					data-list
					data-load-more-root
					aria-busy={favoritesMode
						? store.loadingFavorites
						: store.searching}>
					{#if store.places.length === 0 && (favoritesMode ? store.loadingFavorites : store.searching)}
						<SkeletonList
							label={favoritesMode
								? "Loading favorites…"
								: "Searching games"}>
							{#snippet row(index)}<SkeletonMediaRow {index} />{/snippet}
						</SkeletonList>
					{/if}
					{#each store.places as place (place.placeId)}
						{@const active = store.selected?.placeId === place.placeId}
						{#if selection.isSelecting}
							{@const isChecked = selection.has(place.placeId)}
							<label
								class="list-row relative flex h-12 w-full cursor-pointer items-center gap-2 border-b border-line-subtle px-3"
								data-selected={isChecked || undefined}>
								<span class="checkbox shrink-0">
									<input
										type="checkbox"
										data-list-item
										tabindex={place.placeId === tabStopPlaceId
											? 0
											: -1}
										aria-label={`Select ${store.displayName(place)}`}
										checked={isChecked}
										disabled={store.savingFavorite}
										onkeydown={moveListFocus}
										onchange={() =>
											selection.toggle(place.placeId)} />
									<CheckboxMark />
								</span>
								{@render placeSummary(place)}
							</label>
						{:else}
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
								data-open={store.menu?.place.placeId ===
									place.placeId || undefined}
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
								tabindex={place.placeId === tabStopPlaceId ? 0 : -1}
								aria-keyshortcuts={favoritesMode
									? "Control+Shift+ArrowUp Control+Shift+ArrowDown"
									: undefined}
								onclick={(event) => selectPlace(event, place)}
								oncontextmenu={(event) => store.openMenu(event, place)}
								onkeydown={(event) => {
									if (
										event.key === "ContextMenu" ||
										(event.shiftKey && event.key === "F10")
									)
										store.openMenu(event, place)
									if (
										(event.key === " " || event.key === "Enter") &&
										isMultiSelectGesture(event)
									) {
										event.preventDefault()
										selectPlace(event, place)
									}
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
								{@render placeSummary(place)}
							</button>
						{/if}
					{/each}
					{#if hasMore}
						<LoadMore
							pageKey={store.nextPageToken}
							label="Load more results"
							loadingLabel="Loading more results…"
							isLoadingMore={store.searching}
							hasFailed={store.loadMoreFailed}
							onLoadMore={() => void store.search(true)} />
					{/if}
					<span class="sr-only" aria-live="polite">{moveAnnouncement}</span>
				</div>
				{#if selection.isSelecting}
					<SelectionToolbar
						label="Selected games"
						selectedCount={selection.selectedCount}
						isAllShownSelected={selection.areAllSelected(shownPlaceIds)}
						canSelectAll={shownPlaceIds.length > 0}
						isBusy={store.savingFavorite}
						onToggleAll={() => selection.toggleAll(shownPlaceIds)}
						onStop={() => store.setSelecting(false)}>
						<SelectionActionsMenu
							noun="games"
							selectedCount={selection.selectedCount}
							actions={selectionActions}
							isBusy={store.savingFavorite} />
					</SelectionToolbar>
				{:else if hasSearch}
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
							<span>Clear search</span>
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
							{onFillLaunch}
							{onFillLaunchLink} />
					{:else if store.detailsPage === "badges"}
						<GameBadgesPage {store} {accounts} />
					{:else if store.detailsPage === "gamePasses"}
						<GamePassesPage {store} {accounts} />
					{:else}
						<GameDetails
							{store}
							{accounts}
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
							<h2>Favorites unavailable</h2>
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
							<button
								class="button"
								type="button"
								onclick={() => {
									store.favoritesQuery = ""
									searchInput?.focus()
								}}>Clear search</button>
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
							<p>Select a place to see its details.</p>{/if}
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
						<h2>Search unavailable</h2>
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
						<p>Select a game to see its details.</p>
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
		<div class="flex min-w-0 items-center gap-1">
			<AccountSwitcher store={accounts} />
			<StatusTrail
				parts={store.selected
					? [
							store.displayName(store.selected),
							openPageLabel(profilePages),
						].filter(Boolean)
					: []} />
		</div>
	</footer>
</main>

{#if store.menu}
	<GameContextMenu {store} menu={store.menu} {onFillLaunch} />
{/if}

{#if isConfirmingFavoriteRemoval}
	<ConfirmationDialog
		title="Remove from favorites"
		description={`${selectedFavoriteCount} selected ${selectedFavoriteCount === 1 ? "game leaves" : "games leave"} your favorites, with ${selectedFavoriteCount === 1 ? "its nickname" : "their nicknames"}. This cannot be undone.`}
		confirmLabel="Remove from favorites"
		progressLabel="Removing…"
		isWorking={store.savingFavorite}
		onConfirm={() => void removeSelectedFavorites()}
		onCancel={() => (isConfirmingFavoriteRemoval = false)} />
{/if}
