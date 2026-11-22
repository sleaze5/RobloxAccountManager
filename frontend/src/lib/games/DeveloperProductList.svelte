<script lang="ts">
	import funnelIcon from "@phosphor-icons/core/regular/funnel.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import shoppingBagIcon from "@phosphor-icons/core/regular/shopping-bag.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { tick } from "svelte"
	import { salePriceLabel } from "../inventory/inventory-prices"
	import Icon from "../shared/Icon.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import { matchesSearch, searchTerms } from "../shared/search"
	import SkeletonList from "../shared/SkeletonList.svelte"
	import SkeletonMediaRow from "../shared/SkeletonMediaRow.svelte"
	import type { UniverseDeveloperProducts } from "./developer-products-state.svelte"

	let {
		universeDeveloperProducts,
	}: {
		universeDeveloperProducts: UniverseDeveloperProducts
	} = $props()

	type DeveloperProductFilter = "all" | "forSale"

	const filterID = $props.id(),
		numbers = new Intl.NumberFormat(),
		filterOptions: { id: DeveloperProductFilter; label: string }[] = [
			{ id: "all", label: "All" },
			{ id: "forSale", label: "For sale" },
		]
	let filter = $state<DeveloperProductFilter>("all"),
		query = $state(""),
		searchInput = $state<HTMLInputElement | undefined>(undefined)
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

<div class="min-h-0 flex-1 overflow-auto">
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
		<SkeletonList label="Loading developer products…">
			{#snippet row(index)}<SkeletonMediaRow {index} className="h-16" />{/snippet}
		</SkeletonList>
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
		<ol
			aria-label="Developer products"
			aria-busy={universeDeveloperProducts.loading}>
			{#each shownDeveloperProducts as developerProduct (developerProduct.developerProductId)}
				<li
					class="flex min-h-16 items-center gap-3 border-b border-line-subtle px-3 py-2">
					<span
						class={[
							"grid size-9 shrink-0 place-items-center overflow-hidden rounded-md border border-line-strong bg-raised",
							!developerProduct.isForSale && "opacity-45",
						]}
						aria-hidden="true">
						<ProfileImage
							url={developerProduct.iconUrl}
							fallbackIcon={shoppingBagIcon} />
					</span>
					<span class="flex min-w-0 flex-1 flex-col gap-0.5">
						<strong class="truncate text-xs font-semibold text-fg"
							>{developerProduct.name ||
								"Unnamed developer product"}</strong>
						{#if developerProduct.description}
							<span class="truncate text-xs text-fg-muted"
								>{developerProduct.description}</span>
						{/if}
						<span class="font-mono text-2xs text-fg-faint"
							>{salePriceLabel(developerProduct, false)} ·
							<span class="select-text"
								>ID: {developerProduct.developerProductId}</span
							></span>
					</span>
				</li>
			{/each}
		</ol>
	{/if}
</div>
