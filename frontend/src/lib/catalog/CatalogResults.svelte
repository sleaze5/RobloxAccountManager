<script lang="ts">
	import arrowRightIcon from "@phosphor-icons/core/regular/arrow-right.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import type { Snippet } from "svelte"
	import { untrack } from "svelte"
	import { CatalogItemType } from "../backend/bridge"
	import type { CategorizedInventoryItem } from "../backend/bridge"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import InventoryItemDialog from "../inventory/InventoryItemDialog.svelte"
	import InventoryItemTile from "../inventory/InventoryItemTile.svelte"
	import { inventoryCategoryOptionOf } from "../inventory/inventory-categories"
	import { itemGridClass } from "../inventory/item-grid"
	import {
		categoryItemKey,
		InventoryState,
		inventoryItemKey,
	} from "../inventory/inventory-state.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import Icon from "../shared/Icon.svelte"
	import { loadMoreWhenVisible } from "../shared/load-more-when-visible"
	import SkeletonTile from "../shared/SkeletonTile.svelte"
	import type { CatalogStore } from "./catalog-store.svelte"

	let {
		store,
		accounts,
		notifications,
		label,
		noun,
		onSearchPlace,
		empty,
	}: {
		store: CatalogStore
		accounts: AccountStore
		notifications: NotificationCenter
		// label names the grid for screen readers, such as the category.
		label: string
		noun: string
		onSearchPlace: (placeId: number) => void
		// empty renders the empty state when the search finds nothing.
		empty: Snippet
	} = $props()

	const skeletonTileIndexes = Array.from({ length: 18 }, (_unused, index) => index)
	let detailedItemKey = $state(""),
		// itemActions opens, buys, favorites, and tries on items for the
		// selected account, as the inventory does.
		itemActions = $state<InventoryState | null>(null)

	const detailedItem = $derived(
		detailedItemKey
			? store.items.find(
					(categorizedItem) =>
						inventoryItemKey(categorizedItem.item) === detailedItemKey,
				)
			: undefined,
	)

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
</script>

{#if store.error && store.hasResults}
	<p class="mx-3 mt-3 notice-error" role="alert">{store.error}</p>
{/if}
{#if !store.hasResults && store.error}
	<div class="empty-state h-full" role="alert">
		<Icon svg={warningIcon} size={28} aria-hidden="true" />
		<h2>Marketplace unavailable</h2>
		<p>{store.error}</p>
		<button class="button" type="button" onclick={() => void store.refresh()}
			>Try again</button>
	</div>
{:else if !store.hasResults}
	<div class="p-3" role="status" aria-busy="true">
		<span class="sr-only">Loading {noun}…</span>
		<ul class={[itemGridClass, "animate-enter"]} aria-hidden="true">
			{#each skeletonTileIndexes as index (index)}
				<SkeletonTile />
			{/each}
		</ul>
	</div>
{:else if store.items.length === 0}
	{@render empty()}
{:else}
	{#key store.resultsVersion}
		<ul
			class={[itemGridClass, "animate-enter p-3"]}
			aria-label={label}
			aria-busy={store.loading}>
			{#each store.items as categorizedItem, index (`${inventoryItemKey(categorizedItem.item)}:${index}`)}
				<li class="min-w-0">
					<InventoryItemTile
						item={categorizedItem.item}
						category={inventoryCategoryOptionOf(categorizedItem.category)}
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
				loadMore: () => {
					// A failed page waits for the Try again button, so that it does
					// not retry while it stays in view.
					if (!store.error) void store.loadMore()
				},
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
