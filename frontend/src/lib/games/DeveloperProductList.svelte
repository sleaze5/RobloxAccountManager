<script lang="ts">
	import funnelIcon from "@phosphor-icons/core/regular/funnel.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import shoppingBagIcon from "@phosphor-icons/core/regular/shopping-bag.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { tick } from "svelte"
	import type { DeveloperProduct } from "../backend/bridge"
	import { salePriceLabel } from "../inventory/inventory-prices"
	import { itemGridClass } from "../inventory/item-grid"
	import Icon from "../shared/Icon.svelte"
	import ItemTile from "../shared/ItemTile.svelte"
	import { matchesSearch, searchTerms } from "../shared/search"
	import SkeletonTile from "../shared/SkeletonTile.svelte"
	import DeveloperProductDialog from "./DeveloperProductDialog.svelte"
	import type { UniverseDeveloperProducts } from "./developer-products-state.svelte"

	let {
		universeDeveloperProducts,
		gameName,
	}: {
		universeDeveloperProducts: UniverseDeveloperProducts
		gameName: string
	} = $props()

	type DeveloperProductFilter = "all" | "forSale"

	const filterID = $props.id(),
		numbers = new Intl.NumberFormat(),
		filterOptions: { id: DeveloperProductFilter; label: string }[] = [
			{ id: "all", label: "All" },
			{ id: "forSale", label: "For sale" },
		],
		skeletonTileIndexes = Array.from({ length: 12 }, (_, index) => index)
	let filter = $state<DeveloperProductFilter>("all"),
		query = $state(""),
		searchInput = $state<HTMLInputElement | undefined>(undefined),
		grid = $state<HTMLUListElement | undefined>(undefined),
		shown = $state<DeveloperProduct | null>(null)
	const developerProducts = $derived(
			universeDeveloperProducts.developerProducts ?? [],
		),
		forSaleCount = $derived(
			developerProducts.filter((developerProduct) => developerProduct.isForSale)
				.length,
		),
		terms = $derived(searchTerms(query)),
		shownDeveloperProducts = $derived(
			developerProducts.filter(
				(developerProduct) =>
					(filter === "all" || developerProduct.isForSale) &&
					matchesSearch(
						terms,
						`${developerProduct.name} ${developerProduct.description}`,
					),
			),
		)

	async function closeDeveloperProduct(developerProductId: number): Promise<void> {
		shown = null
		await tick()
		grid?.querySelector<HTMLButtonElement>(
			`[data-developer-product="${developerProductId}"] button`,
		)?.focus()
	}

	async function clearFilters(): Promise<void> {
		filter = "all"
		query = ""
		await tick()
		searchInput?.focus()
	}
</script>

<div
	class="flex flex-wrap items-end gap-x-3 gap-y-2 border-b border-line-subtle p-3"
	role="group"
	aria-label="Developer product filters">
	<div class="flex flex-col gap-1 text-2xs text-fg-muted">
		<span id={`${filterID}-filter`}>Show</span>
		<div class="segmented" role="radiogroup" aria-labelledby={`${filterID}-filter`}>
			{#each filterOptions as option (option.id)}
				<label>
					<input
						type="radio"
						name={`${filterID}-filter`}
						value={option.id}
						checked={filter === option.id}
						disabled={!universeDeveloperProducts.developerProducts}
						onchange={() => (filter = option.id)} />
					<span>{option.label}</span>
				</label>
			{/each}
		</div>
	</div>
	<div
		class="search-field h-7 w-56 flex-none"
		data-closes-on-escape={query ? "" : undefined}>
		<Icon svg={magnifyingGlassIcon} size={14} aria-hidden="true" />
		<input
			bind:this={searchInput}
			type="search"
			placeholder="Search developer products"
			aria-label="Search developer products by name or description"
			autocomplete="off"
			spellcheck="false"
			disabled={!universeDeveloperProducts.developerProducts}
			bind:value={query}
			onkeydown={(event) => {
				if (event.key === "Escape" && query) {
					event.preventDefault()
					query = ""
				}
			}} />
		{#if query}
			<button
				type="button"
				aria-label="Clear search"
				onclick={() => (query = "")}>
				<Icon svg={xIcon} size={14} aria-hidden="true" />
			</button>
		{/if}
	</div>
	{#if developerProducts.length > 0}
		<p class="ml-auto font-mono text-2xs text-fg-muted" role="status">
			<strong class="font-semibold text-fg"
				>{numbers.format(forSaleCount)}</strong>
			of {numbers.format(developerProducts.length)} for sale
		</p>
	{/if}
</div>

<div class="@container min-h-0 flex-1 overflow-auto">
	{#if universeDeveloperProducts.error && !universeDeveloperProducts.developerProducts}
		<div class="empty-state h-full" role="alert">
			<Icon svg={warningIcon} size={28} aria-hidden="true" />
			<h2>Developer products unavailable</h2>
			<p>{universeDeveloperProducts.error}</p>
			<button
				class="button"
				type="button"
				onclick={() => void universeDeveloperProducts.refresh()}
				>Try again</button>
		</div>
	{:else if !universeDeveloperProducts.developerProducts}
		<div class="p-3" role="status" aria-busy="true">
			<span class="sr-only">Loading developer products…</span>
			<ul class={itemGridClass} aria-hidden="true">
				{#each skeletonTileIndexes as index (index)}
					<SkeletonTile />
				{/each}
			</ul>
		</div>
	{:else if developerProducts.length === 0}
		<div class="empty-state h-full" role="status">
			<Icon svg={shoppingBagIcon} size={28} aria-hidden="true" />
			<h2>No developer products</h2>
			<p>This game sells no developer products.</p>
		</div>
	{:else if shownDeveloperProducts.length === 0}
		<div class="empty-state h-full" role="status">
			<Icon svg={funnelIcon} size={28} aria-hidden="true" />
			<h2>No matching developer products</h2>
			<p>Change the search or show all developer products.</p>
			<button class="button" type="button" onclick={() => void clearFilters()}
				>Clear filters</button>
		</div>
	{:else}
		{#if universeDeveloperProducts.error}
			<div class="mx-3 mt-3 flex items-center gap-3 notice-error" role="alert">
				<span class="flex-1">{universeDeveloperProducts.error}</span>
				<button
					class="button"
					type="button"
					onclick={() => void universeDeveloperProducts.refresh()}
					>Try again</button>
			</div>
		{/if}
		<p
			class="flex items-center gap-2 border-b border-line-subtle px-3 py-2 text-xs text-fg-muted">
			<Icon svg={shoppingBagIcon} size={14} aria-hidden="true" />
			Players buy developer products inside the game.
		</p>
		<ul
			class={[itemGridClass, "p-3"]}
			aria-label="Developer products"
			aria-busy={universeDeveloperProducts.loading}
			bind:this={grid}>
			{#each shownDeveloperProducts as developerProduct (developerProduct.developerProductId)}
				<li
					class="min-w-0"
					data-developer-product={developerProduct.developerProductId}>
					<ItemTile
						name={developerProduct.name || "Unnamed developer product"}
						imageUrl={developerProduct.iconUrl}
						fallbackIcon={shoppingBagIcon}
						priceLabel={salePriceLabel(developerProduct, true)}
						isPriceMuted={!developerProduct.isForSale}
						actionLabel="Show details"
						onOpen={() => (shown = developerProduct)} />
				</li>
			{/each}
		</ul>
	{/if}
</div>

{#if shown}
	{@const developerProductId = shown.developerProductId}
	<DeveloperProductDialog
		developerProduct={shown}
		{gameName}
		onClose={() => void closeDeveloperProduct(developerProductId)} />
{/if}
