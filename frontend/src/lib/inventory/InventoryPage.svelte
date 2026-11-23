<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import LoadMore from "../shared/LoadMore.svelte"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import funnelIcon from "@phosphor-icons/core/regular/funnel.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { onMount, untrack } from "svelte"
	import AccountPageHeader from "../accounts/AccountPageHeader.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { InventoryCategory } from "../backend/bridge"
	import type { GamePlace, InventoryItem } from "../backend/bridge"
	import SidebarResizer from "../layout/SidebarResizer.svelte"
	import Icon from "../shared/Icon.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import Select from "../shared/Select.svelte"
	import SkeletonTile from "../shared/SkeletonTile.svelte"
	import { searchTerms } from "../shared/search"
	import {
		assetCategories,
		inventoryCategories,
		inventoryCategoryGroups,
		inventoryTypeLabel,
	} from "./inventory-categories"
	import {
		itemFilterOptions,
		shapeInventoryItems,
		sortOrderOptions,
	} from "./inventory-filters"
	import type { InventoryItemFilter, InventorySortOrder } from "./inventory-filters"
	import { formatRobux } from "./inventory-prices"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import InventoryItemDialog from "./InventoryItemDialog.svelte"
	import InventoryItemTile from "./InventoryItemTile.svelte"
	import { itemGridClass } from "./item-grid"
	import {
		categoryItemKey,
		InventoryState,
		ownedCopyKey,
	} from "./inventory-state.svelte"

	let {
		accounts,
		accountId,
		userId,
		username,
		notifications,
		sidebarWidth = $bindable(),
		onSearchPlace,
		onShowGamePass,
		onBack,
	}: {
		accounts: AccountStore
		notifications: NotificationCenter
		accountId: number
		userId: number
		username: string
		sidebarWidth: number
		onSearchPlace: (placeId: number) => void
		onShowGamePass: (place: GamePlace, gamePassId: number) => void
		onBack: () => void
	} = $props()
	const elementIdPrefix = $props.id()

	const inventory = untrack(
			() => new InventoryState(accountId, userId, accounts, notifications),
		),
		skeletonTileIndexes = Array.from({ length: 12 }, (_unused, index) => index),
		exactNumbers = new Intl.NumberFormat("en-US")

	let filterText = $state(""),
		filterInput = $state<HTMLInputElement | undefined>(undefined),
		contentElement = $state<HTMLElement | undefined>(undefined),
		typeName = $state(""),
		itemFilter = $state<InventoryItemFilter>("all"),
		sortOrder = $state<InventorySortOrder>("newest"),
		detailedCopyKey = $state("")
	const category = $derived(
			inventoryCategories.find((option) => option.id === inventory.category) ??
				inventoryCategories[0],
		),
		page = $derived(inventory.page),
		items = $derived(page?.items ?? []),
		terms = $derived(searchTerms(filterText)),
		typeOptions = $derived.by(() => {
			if (category.itemDetail !== "type") return []
			const typeNames = [...new Set(items.map((item) => item.typeName))].filter(
				(listedTypeName) => inventoryTypeLabel(listedTypeName),
			)
			return typeNames.length > 1
				? [
						{ value: "", label: "All types" },
						...typeNames.map((listedTypeName) => ({
							value: listedTypeName,
							label: inventoryTypeLabel(listedTypeName),
						})),
					]
				: []
		}),
		hasMarket = $derived(items.some((item) => item.market)),
		wornAssetIds = $derived(
			assetCategories.includes(category.id)
				? inventory.wornAssetIds
				: new Set<number>(),
		),
		isWearingAny = $derived(items.some((item) => wornAssetIds.has(item.itemId))),
		shownItemFilterOptions = $derived(
			itemFilterOptions.filter(
				(option) =>
					(option.value !== "wearing" || isWearingAny) &&
					(hasMarket || option.value === "all" || option.value === "wearing"),
			),
		),
		hasValue = $derived(
			hasMarket || items.some((item) => item.recentAveragePrice !== null),
		),
		shownItems = $derived(
			shapeInventoryItems(items, {
				searchTerms: terms,
				typeName,
				itemFilter,
				wornAssetIds,
				sortOrder,
			}),
		),
		isFiltered = $derived(
			terms.length > 0 || typeName !== "" || itemFilter !== "all",
		),
		collectiblesValue = $derived(
			category.itemDetail === "price"
				? items.reduce(
						(total, item) => total + (item.recentAveragePrice ?? 0),
						0,
					)
				: 0,
		),
		detailedItem = $derived(
			detailedCopyKey
				? items.find((item) => ownedCopyKey(item) === detailedCopyKey)
				: undefined,
		),
		isOwnInventory = $derived(Number(accounts.selectedAccount.userId) === userId),
		isPlaceCategory = $derived(category.id === InventoryCategory.InventoryPlaces),
		loadedCountLabel = $derived(
			`${exactNumbers.format(items.length)}${page?.nextCursor ? "+" : ""}`,
		),
		itemCountLabel = $derived(
			isFiltered
				? `${exactNumbers.format(shownItems.length)} of ${loadedCountLabel}`
				: `${loadedCountLabel} ${items.length === 1 && !page?.nextCursor ? "item" : "items"}`,
		)

	function clearFilters(): void {
		filterText = typeName = ""
		itemFilter = "all"
	}

	function showCategory(categoryId: InventoryCategory): void {
		if (categoryId === inventory.category) return
		clearFilters()
		inventory.show(categoryId)
		contentElement?.scrollTo({ top: 0 })
	}

	function openItem(item: InventoryItem): void {
		if (isPlaceCategory) {
			onSearchPlace(item.itemId)
			return
		}
		inventory.clearItemActionError()
		detailedCopyKey = ownedCopyKey(item)
		inventory.loadItemDetails(item, category.id, !isOwnInventory)
		void inventory.checkFavorite(item)
	}

	// The tiles of another user's items show which ones the account owns too.
	$effect(() => {
		if (isOwnInventory || items.length === 0) return
		const categoryId = category.id,
			categorizedItems = items.map((item) => ({ category: categoryId, item }))
		untrack(() => void inventory.loadOwnership(categorizedItems))
	})

	onMount(() => {
		void inventory.refresh()
		return () => inventory.dispose()
	})

	notifyErrors("Inventory not loaded", () => inventory.error)
</script>

<section class="settings-page" aria-labelledby={`${elementIdPrefix}-inventory-title`}>
	<AccountPageHeader
		titleId={`${elementIdPrefix}-inventory-title`}
		title="Inventory"
		{username}
		{onBack}>
		<button
			class="button-ghost"
			type="button"
			aria-label={inventory.loading
				? "Refreshing inventory…"
				: "Refresh inventory"}
			data-tooltip="Refresh inventory"
			data-tooltip-side="bottom-end"
			aria-busy={inventory.loading}
			disabled={inventory.loading || inventory.loadingMore}
			onclick={() => void inventory.refresh()}>
			<RefreshIcon isRefreshing={inventory.loading} />
		</button>
	</AccountPageHeader>

	<div class="settings-layout" style={`--sidebar-width: ${sidebarWidth}px;`}>
		<div class="settings-sidebar pt-1">
			<nav class="settings-nav" aria-label="Inventory categories">
				{#each inventoryCategoryGroups as group (group.label)}
					<h2 class="menu-heading">{group.label}</h2>
					{#each group.categories as option (option.id)}
						<button
							class="nav-item"
							type="button"
							aria-current={inventory.category === option.id
								? "page"
								: undefined}
							onclick={() => showCategory(option.id)}>
							<Icon svg={option.icon} size={16} aria-hidden="true" />
							<span>{option.label}</span>
						</button>
					{/each}
				{/each}
			</nav>
		</div>
		<SidebarResizer bind:width={sidebarWidth} name="inventory" minimumWidth={176} />

		<section
			class="@container flex min-h-0 min-w-0 flex-1 flex-col"
			aria-labelledby={`${elementIdPrefix}-inventory-category-title`}>
			<div
				class="flex h-11 shrink-0 items-center gap-3 border-b border-line-subtle px-3">
				<h2
					id={`${elementIdPrefix}-inventory-category-title`}
					class="flex min-w-0 items-baseline gap-2 text-sm font-semibold text-fg">
					<span class="truncate">{category.label}</span>
					{#if items.length > 0}
						<span
							class="shrink-0 animate-enter font-mono text-2xs font-normal text-fg-muted"
							aria-live="polite">{itemCountLabel}</span>
					{/if}
				</h2>
				<div class="ml-auto w-48 shrink-0">
					<div class="search-field">
						<Icon svg={magnifyingGlassIcon} size={16} aria-hidden="true" />
						<input
							bind:this={filterInput}
							bind:value={filterText}
							type="search"
							aria-label={`Filter ${category.noun} by name, creator, or ID`}
							placeholder={`Filter ${category.noun}`}
							disabled={items.length === 0} />
						{#if filterText}
							<button
								type="button"
								aria-label="Clear search"
								onclick={() => {
									filterText = ""
									filterInput?.focus()
								}}>
								<Icon svg={xIcon} size={14} aria-hidden="true" />
							</button>
						{/if}
					</div>
				</div>
			</div>
			{#if items.length > 0}
				<div
					class="flex h-10 shrink-0 animate-enter items-center gap-2 border-b border-line-subtle px-3"
					role="toolbar"
					aria-label={`Filter and sort ${category.noun}`}>
					{#if typeOptions.length > 0}
						<Select
							label="Type"
							value={typeName}
							options={typeOptions}
							className="w-32"
							onChange={(value) => (typeName = value)} />
					{/if}
					{#if shownItemFilterOptions.length > 1}
						<Select
							label="Show"
							value={itemFilter}
							options={shownItemFilterOptions}
							className="w-28"
							onChange={(value) => (itemFilter = value)} />
					{/if}
					<Select
						label="Sort by"
						value={sortOrder}
						options={hasValue
							? sortOrderOptions
							: sortOrderOptions.filter(
									(option) =>
										option.value === "newest" ||
										option.value === "name",
								)}
						className="w-32"
						onChange={(value) => (sortOrder = value)} />
					{#if collectiblesValue > 0}
						<span
							class="ml-auto shrink-0 text-2xs text-fg-muted"
							data-tooltip={`Total recent average price of the ${loadedCountLabel} loaded collectibles`}
							>Total RAP <span class="font-mono text-fg"
								>{formatRobux(collectiblesValue, true)}{page?.nextCursor
									? "+"
									: ""}</span
							></span>
					{/if}
				</div>
			{:else if !page && !inventory.error}
				<!-- The filters take their place while the first page loads, so
				     the items do not move down when they arrive. -->
				<div
					class="flex h-10 shrink-0 items-center gap-2 border-b border-line-subtle px-3"
					aria-hidden="true">
					<span class="h-7 w-28 skeleton rounded-md"></span>
					<span class="h-7 w-32 skeleton rounded-md"></span>
				</div>
			{/if}

			<div
				class="min-h-0 flex-1 scrollbar-stable overflow-y-auto"
				data-load-more-root
				bind:this={contentElement}>
				{#key inventory.category}
					{#if !page && inventory.error}
						<div class="empty-state h-full">
							<Icon svg={warningIcon} size={28} aria-hidden="true" />
							<h2>Inventory unavailable</h2>
							<button
								class="button"
								type="button"
								onclick={() => void inventory.refresh()}
								>Try again</button>
						</div>
					{:else if !page}
						<div class="p-3" role="status" aria-busy="true">
							<span class="sr-only">Loading {category.noun}…</span>
							<ul
								class={[itemGridClass, "animate-enter"]}
								aria-hidden="true">
								{#each skeletonTileIndexes as index (index)}
									<SkeletonTile hasDetailLine={!isPlaceCategory} />
								{/each}
							</ul>
						</div>
					{:else if items.length === 0}
						<div class="empty-state h-full" role="status">
							<Icon svg={category.icon} size={28} aria-hidden="true" />
							<h2>No {category.noun}</h2>
							<p>@{username} does not own any {category.noun} yet.</p>
						</div>
					{:else if shownItems.length === 0}
						<div class="empty-state h-full" role="status">
							<Icon svg={funnelIcon} size={28} aria-hidden="true" />
							<h2>No matching {category.noun}</h2>
							<p>
								Nothing loaded matches these filters. Clear them or load
								more items.
							</p>
							<button
								class="button"
								type="button"
								onclick={() => {
									clearFilters()
									filterInput?.focus()
								}}>Clear filters</button>
						</div>
					{:else}
						<ul
							class={[itemGridClass, "animate-enter p-3"]}
							aria-label={category.label}
							aria-busy={inventory.loading}>
							{#each shownItems as item, index (`${item.itemId}:${index}`)}
								<li class="min-w-0">
									<InventoryItemTile
										{item}
										{category}
										isWorn={wornAssetIds.has(item.itemId)}
										isOwned={isOwnInventory ||
											inventory.isOwnedByItemKey.get(
												categoryItemKey(
													category.id,
													item.itemId,
												),
											) === true}
										actionLabel={isPlaceCategory
											? "Show in Games"
											: "Show details"}
										onOpen={openItem} />
								</li>
							{/each}
						</ul>
					{/if}
					{#if page?.nextCursor && items.length > 0}
						<LoadMore
							pageKey={items.length}
							label="Load more items"
							loadingLabel="Loading more items…"
							isLoadingMore={inventory.loadingMore}
							hasFailed={!!inventory.error}
							isDisabled={inventory.loading}
							onLoadMore={() => void inventory.loadMore()} />
					{/if}
				{/key}
			</div>
		</section>
	</div>
</section>

{#if detailedItem}
	<InventoryItemDialog
		item={detailedItem}
		{category}
		{inventory}
		{username}
		{accounts}
		{isOwnInventory}
		isWorn={wornAssetIds.has(detailedItem.itemId)}
		onSearchPlace={(placeId) => {
			detailedCopyKey = ""
			onSearchPlace(placeId)
		}}
		onShowGamePass={(place, gamePassId) => {
			detailedCopyKey = ""
			onShowGamePass(place, gamePassId)
		}}
		onClose={() => (detailedCopyKey = "")} />
{/if}
