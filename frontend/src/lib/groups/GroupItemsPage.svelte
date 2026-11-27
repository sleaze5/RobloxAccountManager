<script lang="ts">
	import arrowSquareOutIcon from "@phosphor-icons/core/regular/arrow-square-out.svg?raw"
	import storefrontIcon from "@phosphor-icons/core/regular/storefront.svg?raw"
	import { untrack } from "svelte"
	import { CatalogCreatorType, CatalogSortOrder } from "../backend/bridge"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import CatalogResults from "../catalog/CatalogResults.svelte"
	import { CatalogStore, defaultCatalogSearch } from "../catalog/catalog-store.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import Icon from "../shared/Icon.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import Select from "../shared/Select.svelte"
	import ProfilePageHeader from "../shared/ProfilePageHeader.svelte"
	import type { GroupsStore } from "./groups-store.svelte"

	let {
		store,
		accounts,
		notifications,
		onShowCreations,
	}: {
		store: GroupsStore
		accounts: AccountStore
		notifications: NotificationCenter
		onShowCreations: (groupName: string) => void
	} = $props()

	const groupName = untrack(() => store.profile?.details.group.name ?? ""),
		sortOrderOptions = [
			{
				value: CatalogSortOrder.CatalogSortByRecentlyPublished,
				label: "Recently published",
			},
			{ value: CatalogSortOrder.CatalogSortByBestselling, label: "Bestselling" },
			{
				value: CatalogSortOrder.CatalogSortByMostFavorited,
				label: "Most favorited",
			},
			{
				value: CatalogSortOrder.CatalogSortByHighestPrice,
				label: "Highest price",
			},
			{ value: CatalogSortOrder.CatalogSortByLowestPrice, label: "Lowest price" },
		],
		hiddenSwitchId = $props.id(),
		// The items of the group search the Marketplace on their own, so that
		// the Marketplace page keeps its own search.
		groupCatalog = untrack(() => new CatalogStore(accounts))
	let resultsElement = $state<HTMLElement | undefined>(undefined)
	groupCatalog.search = {
		...defaultCatalogSearch(),
		creatorName: groupName,
		creatorType: CatalogCreatorType.CatalogCreatorGroup,
		sortOrder: CatalogSortOrder.CatalogSortByRecentlyPublished,
		shouldIncludeHiddenItems: true,
	}

	const search = $derived(groupCatalog.search),
		itemCountLabel = $derived(
			`${groupCatalog.items.length}${groupCatalog.nextCursor ? "+" : ""} ${groupCatalog.items.length === 1 && !groupCatalog.nextCursor ? "item" : "items"}`,
		)

	// A new sort replaces the items, so the grid starts at the top.
	$effect(() => {
		void groupCatalog.resultsVersion
		untrack(() => resultsElement?.scrollTo({ top: 0 }))
	})
</script>

<section class="settings-page min-w-0" aria-labelledby="group-items-title">
	<ProfilePageHeader titleId="group-items-title" title="Items" subtitle={groupName}>
		<button
			class="button-ghost"
			type="button"
			aria-label={groupCatalog.loading ? "Refreshing items…" : "Refresh items"}
			data-tooltip="Refresh items"
			data-tooltip-side="bottom-end"
			aria-busy={groupCatalog.loading}
			disabled={groupCatalog.loading || groupCatalog.loadingMore}
			onclick={() => void groupCatalog.refresh()}>
			<RefreshIcon
				isRefreshing={groupCatalog.loading && groupCatalog.hasResults} />
		</button>
	</ProfilePageHeader>
	<div
		class="flex min-h-10 shrink-0 flex-wrap items-center gap-2 border-b border-line-subtle px-3 py-1.5"
		role="toolbar"
		aria-label="Sort and filter items">
		<Select
			label="Sort by"
			value={search.sortOrder}
			options={sortOrderOptions}
			className="w-40"
			onChange={(sortOrder) => groupCatalog.applySearch({ sortOrder })} />
		<label
			class="flex items-center gap-2 text-xs text-fg-muted"
			for={hiddenSwitchId}>
			<span class="switch">
				<input
					id={hiddenSwitchId}
					type="checkbox"
					role="switch"
					checked={search.shouldIncludeHiddenItems}
					onchange={(event) =>
						groupCatalog.applySearch({
							shouldIncludeHiddenItems: event.currentTarget.checked,
						})} />
				<span aria-hidden="true"></span>
			</span>
			Show hidden items
		</label>
		{#if groupCatalog.hasResults && groupCatalog.items.length > 0}
			<span
				class="animate-enter font-mono text-2xs text-fg-muted"
				aria-live="polite">{itemCountLabel}</span>
		{/if}
		<button
			class="ml-auto button"
			type="button"
			onclick={() => onShowCreations(groupName)}>
			<Icon svg={arrowSquareOutIcon} size={16} aria-hidden="true" />Show in
			Marketplace
		</button>
	</div>
	<div
		class="@container min-h-0 flex-1 scrollbar-stable overflow-y-auto"
		data-load-more-root
		bind:this={resultsElement}>
		<CatalogResults
			store={groupCatalog}
			{accounts}
			{notifications}
			label={`Items by ${groupName}`}
			noun="items">
			{#snippet empty()}
				<div class="empty-state h-full" role="status">
					<Icon svg={storefrontIcon} size={28} aria-hidden="true" />
					<h2>No items</h2>
					<p>
						{search.shouldIncludeHiddenItems
							? `${groupName} has no items in the Marketplace.`
							: `${groupName} sells no items. Show hidden items to see off-sale ones.`}
					</p>
				</div>
			{/snippet}
		</CatalogResults>
	</div>
</section>
