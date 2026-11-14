<script lang="ts">
	import funnelIcon from "@phosphor-icons/core/regular/funnel.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { onMount, untrack } from "svelte"
	import AccountPageHeader from "../accounts/AccountPageHeader.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import type { InventoryCategory } from "../backend/bridge"
	import SidebarResizer from "../layout/SidebarResizer.svelte"
	import Icon from "../shared/Icon.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import { matchesSearch, searchTerms } from "../shared/search"
	import { inventoryCategoryGroups } from "./inventory-categories"
	import InventoryItemTile from "./InventoryItemTile.svelte"
	import { InventoryState } from "./inventory-state.svelte"

	let {
		accounts,
		accountId,
		userId,
		username,
		sidebarWidth = $bindable(),
		onBack,
	}: {
		accounts: AccountStore
		accountId: number
		userId: number
		username: string
		sidebarWidth: number
		onBack: () => void
	} = $props()

	const inventory = untrack(() => new InventoryState(accountId, userId, accounts)),
		categories = inventoryCategoryGroups.flatMap((group) => group.categories),
		skeletonTileIndexes = Array.from({ length: 12 }, (_unused, index) => index),
		exactNumbers = new Intl.NumberFormat("en-US")

	let filterText = $state(""),
		filterInput = $state<HTMLInputElement | undefined>(undefined),
		contentElement = $state<HTMLElement | undefined>(undefined)
	const category = $derived(
			categories.find((option) => option.id === inventory.category) ??
				categories[0],
		),
		page = $derived(inventory.page),
		items = $derived(page?.items ?? []),
		terms = $derived(searchTerms(filterText)),
		shownItems = $derived(
			terms.length
				? items.filter((item) => matchesSearch(terms, item.name))
				: items,
		),
		itemCountLabel = $derived(
			`${exactNumbers.format(items.length)}${page?.nextCursor ? "+" : ""} ${items.length === 1 && !page?.nextCursor ? "item" : "items"}`,
		)

	function showCategory(categoryId: InventoryCategory): void {
		if (categoryId === inventory.category) return
		filterText = ""
		inventory.show(categoryId)
		contentElement?.scrollTo({ top: 0 })
	}

	onMount(() => {
		void inventory.refresh()
		return () => inventory.dispose()
	})
</script>

<section class="settings-page" aria-labelledby="inventory-title">
	<AccountPageHeader titleId="inventory-title" title="Inventory" {username} {onBack}>
		<button
			class="button-ghost"
			type="button"
			aria-label={inventory.loading
				? "Refreshing inventory"
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
		<aside class="settings-sidebar pt-1">
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
		</aside>
		<SidebarResizer bind:width={sidebarWidth} name="inventory" minimumWidth={176} />

		<main
			class="@container flex min-h-0 min-w-0 flex-1 flex-col"
			aria-labelledby="inventory-category-title">
			<div
				class="flex h-11 shrink-0 items-center gap-3 border-b border-line-subtle px-3">
				<h2
					id="inventory-category-title"
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
							aria-label={`Filter ${category.noun}`}
							placeholder={`Filter ${category.noun}`}
							disabled={items.length === 0} />
						{#if filterText}
							<button
								type="button"
								aria-label="Clear filter"
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

			<div
				class="min-h-0 flex-1 scrollbar-stable overflow-y-auto"
				bind:this={contentElement}>
				{#key inventory.category}
					{#if inventory.error && items.length > 0}
						<p class="mx-3 mt-3 notice-error" role="alert">
							{inventory.error}
						</p>
					{/if}
					{#if !page && inventory.error}
						<div class="empty-state h-full" role="alert">
							<Icon svg={category.icon} size={28} aria-hidden="true" />
							<h2>Inventory is unavailable</h2>
							<p>{inventory.error}</p>
							<button
								class="button"
								type="button"
								onclick={() => void inventory.refresh()}
								>Try again</button>
						</div>
					{:else if !page}
						<div class="p-3" role="status" aria-busy="true">
							<span class="sr-only">Loading {category.noun}</span>
							<ul
								class="grid animate-enter grid-cols-4 gap-2 @lg:grid-cols-5 @2xl:grid-cols-6 @3xl:grid-cols-7 @4xl:grid-cols-8 @6xl:grid-cols-10"
								aria-hidden="true">
								{#each skeletonTileIndexes as index (index)}
									<li
										class="flex flex-col gap-1 rounded-lg border border-line-subtle bg-surface p-1">
										<span
											class="block aspect-square w-full skeleton rounded-sm"
										></span>
										<span class="flex flex-col px-1 pb-0.5 text-xs">
											<span class="skeleton-line w-3/4"></span>
										</span>
									</li>
								{/each}
							</ul>
						</div>
					{:else if items.length === 0}
						<div class="empty-state h-full" role="status">
							<Icon svg={category.icon} size={28} aria-hidden="true" />
							<h2>No {category.noun}</h2>
							<p>@{username} doesn't own any {category.noun} yet.</p>
						</div>
					{:else if shownItems.length === 0}
						<div class="empty-state h-full" role="status">
							<Icon svg={funnelIcon} size={28} aria-hidden="true" />
							<h2>No matching {category.noun}</h2>
							<p>
								Nothing loaded matches “{filterText.trim()}”. Clear the
								filter or load more items.
							</p>
							<button
								class="button"
								type="button"
								onclick={() => {
									filterText = ""
									filterInput?.focus()
								}}>Clear filter</button>
						</div>
					{:else}
						<ul
							class="grid animate-enter grid-cols-4 gap-2 p-3 @lg:grid-cols-5 @2xl:grid-cols-6 @3xl:grid-cols-7 @4xl:grid-cols-8 @6xl:grid-cols-10"
							aria-label={category.label}
							aria-busy={inventory.loading}>
							{#each shownItems as item, index (`${item.itemId}:${index}`)}
								<li class="min-w-0">
									<InventoryItemTile
										{item}
										{category}
										onOpen={(openedItem) =>
											void inventory.openItem(openedItem)} />
								</li>
							{/each}
						</ul>
					{/if}
					{#if page?.nextCursor && items.length > 0}
						<div class="flex min-h-12 items-center justify-center pb-3">
							<button
								class="button"
								type="button"
								aria-busy={inventory.loadingMore}
								disabled={inventory.loading || inventory.loadingMore}
								onclick={() => void inventory.loadMore()}
								>{inventory.loadingMore
									? "Loading…"
									: "Load more"}</button>
						</div>
					{/if}
				{/key}
			</div>
		</main>
	</div>
</section>
