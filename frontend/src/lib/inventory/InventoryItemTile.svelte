<script lang="ts">
	import { InventoryCategory } from "../backend/bridge"
	import type { InventoryItem } from "../backend/bridge"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import { inventoryTypeLabel } from "./inventory-categories"
	import type { InventoryCategoryOption } from "./inventory-categories"
	import { formatRobux, marketPriceLabel } from "./inventory-prices"

	let {
		item,
		category,
		actionLabel,
		onOpen,
	}: {
		item: InventoryItem
		category: InventoryCategoryOption
		actionLabel: string
		onOpen: (item: InventoryItem) => void
	} = $props()

	const exactNumbers = new Intl.NumberFormat("en-US"),
		percent = new Intl.NumberFormat("en-US", {
			style: "percent",
			maximumFractionDigits: 1,
		}),
		name = $derived(item.name || "Unnamed item"),
		typeLabel = $derived(
			category.itemDetail === "type" ? inventoryTypeLabel(item.typeName) : "",
		),
		priceLabel = $derived(
			category.itemDetail === "price" && item.recentAveragePrice !== null
				? `RAP ${formatRobux(item.recentAveragePrice, true)}`
				: item.market
					? marketPriceLabel(item.market, true)
					: item.badge
						? `${percent.format(item.badge.pastDayWinFraction)} won`
						: "",
		),
		isUnpriced = $derived(
			!!item.market &&
				!item.market.isForSale &&
				item.market.lowestResalePriceInRobux === null &&
				item.recentAveragePrice === null,
		)
</script>

<button
	class="group/tile flex w-full min-w-0 flex-col gap-1 rounded-lg border border-line-subtle bg-surface p-1 text-left transition-colors duration-100 hover:border-line hover:bg-hover"
	type="button"
	data-tooltip={actionLabel}
	data-tooltip-strong={name}
	onclick={() => onOpen(item)}>
	<span
		class="relative grid aspect-square w-full place-items-center overflow-hidden rounded-sm bg-raised">
		<span
			class="grid size-full place-items-center transition-transform duration-150 ease-out motion-safe:group-hover/tile:scale-105"
			aria-hidden="true">
			<ProfileImage url={item.imageUrl} fallbackIcon={category.icon} />
		</span>
		{#if item.serialNumber !== null}
			<span
				class="absolute top-1 left-1 rounded-sm border border-line-strong bg-overlay px-1 font-mono text-2xs leading-4 font-medium text-fg"
				><span class="sr-only">Serial number </span>#{exactNumbers.format(
					item.serialNumber,
				)}</span>
		{/if}
		{#if item.market?.isLimited && category.id !== InventoryCategory.InventoryCollectibles}
			<span
				class="absolute right-1 bottom-1 rounded-sm border border-line-strong bg-overlay px-1 text-2xs leading-4 font-medium text-warning"
				>Limited</span>
		{/if}
	</span>
	<span class="flex min-w-0 flex-col px-1 pb-0.5">
		<span class="truncate text-xs font-medium text-fg">{name}</span>
		{#if typeLabel || priceLabel}
			<span class="flex min-w-0 items-baseline justify-between gap-2 text-2xs">
				<span class="truncate text-fg-muted">{typeLabel}</span>
				{#if priceLabel}
					<span
						class={[
							"shrink-0 font-mono",
							isUnpriced ? "text-fg-faint" : "text-fg",
						]}>{priceLabel}</span>
				{/if}
			</span>
		{/if}
	</span>
</button>
