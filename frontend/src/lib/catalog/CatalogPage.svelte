<script lang="ts">
	import arrowRightIcon from "@phosphor-icons/core/regular/arrow-right.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import diamondIcon from "@phosphor-icons/core/regular/diamond.svg?raw"
	import eyeIcon from "@phosphor-icons/core/regular/eye.svg?raw"
	import funnelIcon from "@phosphor-icons/core/regular/funnel.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import storefrontIcon from "@phosphor-icons/core/regular/storefront.svg?raw"
	import tagIcon from "@phosphor-icons/core/regular/tag.svg?raw"
	import hourglassIcon from "@phosphor-icons/core/regular/hourglass.svg?raw"
	import userIcon from "@phosphor-icons/core/regular/user.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { onMount, tick, untrack } from "svelte"
	import {
		CatalogCreatorType,
		CatalogItemType,
		CatalogSalesType,
		CatalogSortOrder,
		CatalogSortPeriod,
	} from "../backend/bridge"
	import type { CatalogCategory, CategorizedInventoryItem } from "../backend/bridge"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import InventoryItemDialog from "../inventory/InventoryItemDialog.svelte"
	import InventoryItemTile from "../inventory/InventoryItemTile.svelte"
	import { inventoryCategoryOptionOf } from "../inventory/inventory-categories"
	import { formatRobux } from "../inventory/inventory-prices"
	import {
		categoryItemKey,
		InventoryState,
		inventoryItemKey,
	} from "../inventory/inventory-state.svelte"
	import AccountSwitcher from "../layout/AccountSwitcher.svelte"
	import WorkbenchSidebar from "../layout/WorkbenchSidebar.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import Icon from "../shared/Icon.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import Select from "../shared/Select.svelte"
	import SkeletonTile from "../shared/SkeletonTile.svelte"
	import { loadMoreWhenVisible } from "../shared/load-more-when-visible"
	import {
		allCatalogItemsCategory,
		backgroundsCategory,
		catalogCategoryGroups,
		catalogCategoryOptionOf,
		catalogGroupOf,
	} from "./catalog-categories"
	import type { CatalogCategoryOption } from "./catalog-categories"
	import CatalogFiltersPanel from "./CatalogFiltersPanel.svelte"
	import {
		countActiveCatalogFilters,
		defaultCatalogSearch,
	} from "./catalog-store.svelte"
	import type { CatalogStore } from "./catalog-store.svelte"

	let {
		store,
		accounts,
		notifications,
		onSearchPlace,
	}: {
		store: CatalogStore
		accounts: AccountStore
		notifications: NotificationCenter
		onSearchPlace: (placeId: number) => void
	} = $props()

	const sortOrderOptions = [
			{ value: CatalogSortOrder.CatalogSortByRelevance, label: "Relevance" },
			{
				value: CatalogSortOrder.CatalogSortByMostFavorited,
				label: "Most favorited",
			},
			{ value: CatalogSortOrder.CatalogSortByBestselling, label: "Bestselling" },
			{
				value: CatalogSortOrder.CatalogSortByRecentlyPublished,
				label: "Recently published",
			},
			{
				value: CatalogSortOrder.CatalogSortByHighestPrice,
				label: "Highest price",
			},
			{ value: CatalogSortOrder.CatalogSortByLowestPrice, label: "Lowest price" },
		],
		sortPeriodOptions = [
			{ value: CatalogSortPeriod.CatalogPastDay, label: "Past day" },
			{ value: CatalogSortPeriod.CatalogPastWeek, label: "Past week" },
			{ value: CatalogSortPeriod.CatalogAllTime, label: "All time" },
		],
		skeletonTileIndexes = Array.from({ length: 18 }, (_unused, index) => index),
		exactNumbers = new Intl.NumberFormat("en-US"),
		gridClass =
			"grid grid-cols-4 gap-2 @lg:grid-cols-5 @2xl:grid-cols-6 @3xl:grid-cols-7 @4xl:grid-cols-8 @6xl:grid-cols-10"

	let searchInput = $state<HTMLInputElement | undefined>(undefined),
		resultsElement = $state<HTMLElement | undefined>(undefined),
		filtersButton = $state<HTMLButtonElement | undefined>(undefined),
		isFiltersPanelOpen = $state(false),
		detailedItemKey = $state(""),
		// itemActions opens, buys, favorites, and tries on items for the
		// selected account, as the inventory does.
		itemActions = $state<InventoryState | null>(null)

	const search = $derived(store.search),
		category = $derived(catalogCategoryOptionOf(search.category)),
		selectedGroup = $derived(catalogGroupOf(search.category)),
		activeFilterCount = $derived(countActiveCatalogFilters(search)),
		hasPeriod = $derived(
			search.sortOrder === CatalogSortOrder.CatalogSortByMostFavorited ||
				search.sortOrder === CatalogSortOrder.CatalogSortByBestselling,
		),
		isNarrowed = $derived(search.keyword !== "" || activeFilterCount > 0),
		itemCountLabel = $derived(
			`${exactNumbers.format(store.items.length)}${store.nextCursor ? "+" : ""} ${store.items.length === 1 && !store.nextCursor ? "item" : "items"}`,
		),
		priceFilterLabel = $derived(
			priceRangeLabel(search.minPriceInRobux, search.maxPriceInRobux),
		),
		detailedItem = $derived(
			detailedItemKey
				? store.items.find(
						(categorizedItem) =>
							inventoryItemKey(categorizedItem.item) === detailedItemKey,
					)
				: undefined,
		)

	function priceRangeLabel(min: number | null, max: number | null): string {
		if (min === 0 && max === 0) return "Free"
		if (min !== null && max !== null) {
			return `${formatRobux(min, true)} to ${formatRobux(max, true)}`
		}
		if (min !== null) return `${formatRobux(min, true)} or more`
		if (max !== null) return `${formatRobux(max, true)} or less`
		return ""
	}

	function isWorn(categorizedItem: CategorizedInventoryItem): boolean {
		return (
			categorizedItem.item.catalogItemType === CatalogItemType.CatalogAsset &&
			!!itemActions?.wornAssetIds.has(categorizedItem.item.itemId)
		)
	}

	function isOwned(categorizedItem: CategorizedInventoryItem): boolean {
		return (
			itemActions?.isOwnedByItemKey.get(
				categoryItemKey(categorizedItem.category, categorizedItem.item.itemId),
			) === true
		)
	}

	function showCategory(categoryId: CatalogCategory): void {
		if (categoryId === search.category) return
		store.applySearch({ category: categoryId })
	}

	function openItem(categorizedItem: CategorizedInventoryItem): void {
		if (!itemActions) return
		itemActions.clearItemActionError()
		detailedItemKey = inventoryItemKey(categorizedItem.item)
		itemActions.loadItemDetails(
			categorizedItem.item,
			categorizedItem.category,
			true,
		)
		void itemActions.checkFavorite(categorizedItem.item)
	}

	function clearKeyword(): void {
		store.keywordInput = ""
		if (search.keyword) store.submitKeyword()
		searchInput?.focus()
	}

	async function clearAllFilters(): Promise<void> {
		store.clearFilters()
		await tick()
		searchInput?.focus()
	}

	// A new search replaces the results, so the grid starts at the top.
	$effect(() => {
		void store.resultsVersion
		untrack(() => resultsElement?.scrollTo({ top: 0 }))
	})

	$effect(() => {
		const account = accounts.selectedAccount
		if (account.id === 0 || !accounts.vault.unlocked) return
		const accountItemActions = untrack(
			() =>
				new InventoryState(
					account.id,
					Number(account.userId) || 0,
					accounts,
					notifications,
				),
		)
		itemActions = accountItemActions
		detailedItemKey = ""
		void accountItemActions.loadWornAssets()
		untrack(() => store.refreshWhenStale())
		return () => {
			accountItemActions.dispose()
			if (itemActions === accountItemActions) itemActions = null
		}
	})

	$effect(() => {
		const accountItemActions = itemActions,
			items = store.items
		if (accountItemActions && items.length > 0) {
			untrack(() => void accountItemActions.loadOwnership(items))
		}
	})

	onMount(() => {
		if (!store.hasResults) searchInput?.focus()
	})
</script>

{#snippet categoryButton(option: CatalogCategoryOption, isBranch: boolean)}
	<button
		class="nav-item"
		type="button"
		aria-current={search.category === option.id ? "page" : undefined}
		data-current-branch={isBranch || undefined}
		onclick={() => showCategory(option.id)}>
		<Icon svg={option.icon} size={16} aria-hidden="true" />
		<span>{option.label}</span>
	</button>
{/snippet}

{#snippet filterChip(
	icon: string,
	label: string,
	clearLabel: string,
	onClear: () => void,
)}
	<li class="tag-pill shrink-0 animate-enter pr-0.5">
		<Icon svg={icon} size={14} aria-hidden="true" />
		<span>{label}</span>
		<button
			class="button-ghost-sm size-5"
			type="button"
			aria-label={clearLabel}
			data-tooltip={clearLabel}
			onclick={onClear}>
			<Icon svg={xIcon} size={12} aria-hidden="true" />
		</button>
	</li>
{/snippet}

<main class="flex min-h-0 flex-1 flex-col bg-canvas" aria-label="Marketplace">
	<div
		class="flex min-h-0 flex-1"
		style={`--sidebar-width: ${store.sidebarCollapsed ? 44 : store.sidebarWidth}px;`}>
		<WorkbenchSidebar
			label="Marketplace categories"
			name="Marketplace"
			icon={storefrontIcon}
			bind:collapsed={store.sidebarCollapsed}
			bind:width={store.sidebarWidth}>
			{#snippet title()}Marketplace{/snippet}
			{#snippet tools()}
				<form
					novalidate
					role="search"
					onsubmit={(event) => {
						event.preventDefault()
						store.submitKeyword()
					}}>
					<div class="search-field">
						<Icon svg={magnifyingGlassIcon} size={16} aria-hidden="true" />
						<input
							type="text"
							placeholder="Search Marketplace"
							aria-label="Search Marketplace items by name or keyword"
							maxlength={100}
							autocomplete="off"
							spellcheck="false"
							data-catalog-search
							bind:this={searchInput}
							bind:value={store.keywordInput} />
						{#if store.keywordInput}
							<button
								type="button"
								aria-label="Clear search"
								onclick={clearKeyword}>
								<Icon svg={xIcon} size={14} aria-hidden="true" />
							</button>
						{/if}
					</div>
				</form>
			{/snippet}
			<nav
				class="settings-nav min-h-0 flex-1 scrollbar-stable overflow-y-auto pt-1"
				aria-label="Marketplace categories">
				{@render categoryButton(allCatalogItemsCategory, false)}
				{#each catalogCategoryGroups as group (group.label)}
					<h2 class="menu-heading">{group.label}</h2>
					{@render categoryButton(
						group.parent,
						selectedGroup === group && search.category !== group.parent.id,
					)}
					{#each group.categories as option (option.id)}
						{@render categoryButton(option, false)}
					{/each}
				{/each}
				<h2 class="menu-heading">Other</h2>
				{@render categoryButton(backgroundsCategory, false)}
			</nav>
		</WorkbenchSidebar>

		<section
			class="@container flex min-h-0 min-w-0 flex-1 flex-col border-l border-line-subtle"
			aria-labelledby="catalog-category-title">
			<div
				class="flex h-11 shrink-0 items-center gap-3 border-b border-line-subtle px-3">
				<h2
					id="catalog-category-title"
					class="flex min-w-0 items-baseline gap-2 text-sm font-semibold text-fg">
					<span class="truncate"
						>{search.keyword
							? `"${search.keyword}"`
							: category.label}</span>
					{#if search.keyword}
						<span class="shrink-0 text-xs font-normal text-fg-muted"
							>in {category.label}</span>
					{/if}
					{#if store.hasResults && store.items.length > 0}
						<span
							class="shrink-0 animate-enter font-mono text-2xs font-normal text-fg-muted"
							aria-live="polite">{itemCountLabel}</span>
					{/if}
				</h2>
				<button
					class="ml-auto button-ghost"
					type="button"
					aria-label={store.loading
						? "Refreshing results…"
						: "Refresh results"}
					data-tooltip="Refresh results"
					data-tooltip-side="bottom-end"
					aria-busy={store.loading}
					disabled={store.loading || store.loadingMore}
					onclick={() => void store.refresh()}>
					<RefreshIcon isRefreshing={store.loading && store.hasResults} />
				</button>
			</div>

			<div
				class="flex min-h-10 shrink-0 items-center gap-2 border-b border-line-subtle px-3 py-1.5"
				role="toolbar"
				aria-label="Sort and filter items">
				<Select
					label="Sort by"
					value={search.sortOrder}
					options={sortOrderOptions}
					className="w-36"
					onChange={(sortOrder) => store.applySearch({ sortOrder })} />
				{#if hasPeriod}
					<span class="animate-enter">
						<Select
							label="Time period"
							value={search.sortPeriod}
							options={sortPeriodOptions}
							className="w-28"
							onChange={(sortPeriod) =>
								store.applySearch({ sortPeriod })} />
					</span>
				{/if}
				<button
					class="relative button shrink-0"
					type="button"
					aria-haspopup="dialog"
					aria-expanded={isFiltersPanelOpen}
					data-open={isFiltersPanelOpen || undefined}
					bind:this={filtersButton}
					onclick={() => (isFiltersPanelOpen = !isFiltersPanelOpen)}>
					<Icon svg={funnelIcon} size={16} aria-hidden="true" />
					<span>Filters</span>
					{#if activeFilterCount > 0}
						<span
							class="grid h-4 min-w-4 animate-bloom place-items-center rounded-sm bg-accent-strong px-0.5 font-mono text-2xs font-bold text-canvas"
							><span class="sr-only">, </span>{activeFilterCount}<span
								class="sr-only">
								active</span
							></span>
					{/if}
				</button>
				{#if activeFilterCount > 0}
					<ul
						class="flex min-w-0 flex-1 items-center gap-1 overflow-hidden"
						aria-label="Active filters">
						{#if search.creatorName}
							{@render filterChip(
								search.creatorType ===
									CatalogCreatorType.CatalogCreatorGroup
									? usersThreeIcon
									: userIcon,
								search.creatorType ===
									CatalogCreatorType.CatalogCreatorGroup
									? search.creatorName
									: `@${search.creatorName}`,
								"Clear creator",
								() =>
									store.applySearch({
										creatorName: "",
										creatorType: defaultCatalogSearch().creatorType,
									}),
							)}
						{/if}
						{#if search.shouldIncludeHiddenItems}
							{@render filterChip(
								eyeIcon,
								"Hidden items",
								"Hide hidden items",
								() =>
									store.applySearch({
										shouldIncludeHiddenItems: false,
									}),
							)}
						{/if}
						{#if search.salesType !== CatalogSalesType.CatalogAnySalesType}
							{@render filterChip(
								search.salesType === CatalogSalesType.CatalogLimited
									? diamondIcon
									: hourglassIcon,
								search.salesType === CatalogSalesType.CatalogLimited
									? "Limited"
									: "Timed options",
								"Clear sales type",
								() =>
									store.applySearch({
										salesType: CatalogSalesType.CatalogAnySalesType,
									}),
							)}
						{/if}
						{#if priceFilterLabel}
							{@render filterChip(
								tagIcon,
								priceFilterLabel,
								"Clear price",
								() =>
									store.applySearch({
										minPriceInRobux: null,
										maxPriceInRobux: null,
									}),
							)}
						{/if}
					</ul>
				{/if}
			</div>

			<div
				class="min-h-0 flex-1 scrollbar-stable overflow-y-auto"
				data-load-more-root
				bind:this={resultsElement}>
				{#if store.error && store.hasResults}
					<p class="mx-3 mt-3 notice-error" role="alert">{store.error}</p>
				{/if}
				{#if !store.hasResults && store.error}
					<div class="empty-state h-full" role="alert">
						<Icon svg={warningIcon} size={28} aria-hidden="true" />
						<h2>Marketplace unavailable</h2>
						<p>{store.error}</p>
						<button
							class="button"
							type="button"
							onclick={() => void store.refresh()}>Try again</button>
					</div>
				{:else if !store.hasResults}
					<div class="p-3" role="status" aria-busy="true">
						<span class="sr-only">Loading {category.noun}…</span>
						<ul class={[gridClass, "animate-enter"]} aria-hidden="true">
							{#each skeletonTileIndexes as index (index)}
								<SkeletonTile />
							{/each}
						</ul>
					</div>
				{:else if store.items.length === 0}
					<div class="empty-state h-full" role="status">
						<Icon
							svg={isNarrowed ? funnelIcon : category.icon}
							size={28}
							aria-hidden="true" />
						<h2>No matching {category.noun}</h2>
						{#if isNarrowed}
							<p>
								Nothing in {category.label} matches this search. Clear the
								search and filters, or select another category.
							</p>
							<button
								class="button"
								type="button"
								onclick={() => void clearAllFilters()}
								>Clear filters</button>
						{:else}
							<p>
								The Marketplace lists nothing here right now. Select
								another category.
							</p>
						{/if}
					</div>
				{:else}
					{#key store.resultsVersion}
						<ul
							class={[gridClass, "animate-enter p-3"]}
							aria-label={category.label}
							aria-busy={store.loading}>
							{#each store.items as categorizedItem, index (`${inventoryItemKey(categorizedItem.item)}:${index}`)}
								<li class="min-w-0">
									<InventoryItemTile
										item={categorizedItem.item}
										category={inventoryCategoryOptionOf(
											categorizedItem.category,
										)}
										isWorn={isWorn(categorizedItem)}
										isOwned={isOwned(categorizedItem)}
										isDisabled={!itemActions}
										actionLabel="Show details"
										onOpen={() => openItem(categorizedItem)} />
								</li>
							{/each}
						</ul>
					{/key}
					{#if store.nextCursor}
						<div
							class="flex min-h-12 items-center justify-center pb-3"
							use:loadMoreWhenVisible={{
								pageKey: store.items.length,
								loadMore: () => void store.loadMore(),
							}}>
							{#if store.loadingMore}
								<span
									class="flex items-center gap-2 text-xs text-fg-muted"
									role="status"
									><Icon
										svg={circleNotchIcon}
										class="spinner"
										size={16}
										aria-hidden="true" />Loading more items…</span>
							{:else}
								<button
									class="button"
									type="button"
									disabled={store.loading}
									onclick={() => void store.loadMore()}
									>{store.error ? "Try again" : "Load more"}<Icon
										svg={arrowRightIcon}
										data-nudge="right"
										size={14}
										aria-hidden="true" /></button>
							{/if}
						</div>
					{/if}
				{/if}
			</div>
		</section>
	</div>
	<footer
		class="flex h-6 shrink-0 items-center border-t border-line-subtle bg-surface pl-3 text-2xs text-fg-faint"
		aria-label="Marketplace status">
		<div class="flex min-w-0 items-center gap-4">
			<AccountSwitcher store={accounts} />
		</div>
	</footer>
</main>

{#if isFiltersPanelOpen}
	<CatalogFiltersPanel
		anchor={filtersButton}
		{search}
		onApply={(change) => {
			isFiltersPanelOpen = false
			filtersButton?.focus()
			store.applySearch(change)
		}}
		onClose={() => {
			isFiltersPanelOpen = false
			filtersButton?.focus()
		}} />
{/if}

{#if detailedItem && itemActions}
	<InventoryItemDialog
		item={detailedItem.item}
		category={inventoryCategoryOptionOf(detailedItem.category)}
		inventory={itemActions}
		username={accounts.selectedAccount.username}
		{accounts}
		isOwnInventory={false}
		isWorn={isWorn(detailedItem)}
		onSearchPlace={(placeId) => {
			detailedItemKey = ""
			onSearchPlace(placeId)
		}}
		onClose={() => (detailedItemKey = "")} />
{/if}
