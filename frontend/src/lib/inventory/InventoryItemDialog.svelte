<script lang="ts">
	import barcodeIcon from "@phosphor-icons/core/regular/barcode.svg?raw"
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import heartIcon from "@phosphor-icons/core/regular/heart.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import storefrontIcon from "@phosphor-icons/core/regular/storefront.svg?raw"
	import tagIcon from "@phosphor-icons/core/regular/tag.svg?raw"
	import trendUpIcon from "@phosphor-icons/core/regular/trend-up.svg?raw"
	import trophyIcon from "@phosphor-icons/core/regular/trophy.svg?raw"
	import userCircleIcon from "@phosphor-icons/core/regular/user-circle.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import type { InventoryItem } from "../backend/bridge"
	import Dialog from "../shared/Dialog.svelte"
	import Icon from "../shared/Icon.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import { inventoryTypeLabel } from "./inventory-categories"
	import type { InventoryCategoryOption } from "./inventory-categories"
	import { formatRobux } from "./inventory-prices"

	let {
		item,
		category,
		onSearchPlace,
		onClose,
	}: {
		item: InventoryItem
		category: InventoryCategoryOption
		onSearchPlace: (placeId: number) => void
		onClose: () => void
	} = $props()

	const exactNumbers = new Intl.NumberFormat("en-US"),
		percent = new Intl.NumberFormat("en-US", {
			style: "percent",
			maximumFractionDigits: 1,
		}),
		name = $derived(item.name || "Unnamed item"),
		typeLabel = $derived(inventoryTypeLabel(item.typeName)),
		market = $derived(item.market),
		badge = $derived(item.badge),
		subtitle = $derived(
			[typeLabel || category.label, market?.isLimited ? "Limited" : ""]
				.filter(Boolean)
				.join(" · "),
		)
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") {
			event.preventDefault()
			onClose()
		}
	}} />

<Dialog
	class="max-w-md text-left"
	aria-labelledby="inventory-item-title"
	aria-describedby="inventory-item-subtitle"
	onBackdropClick={onClose}>
	<div class="dialog-heading">
		<div class="flex items-center gap-3">
			<span
				class="grid size-20 shrink-0 place-items-center overflow-hidden rounded-lg border border-line-subtle bg-raised"
				aria-hidden="true">
				<ProfileImage url={item.imageUrl} fallbackIcon={category.icon} />
			</span>
			<div class="min-w-0">
				<h2 id="inventory-item-title" class="wrap-anywhere">{name}</h2>
				<p id="inventory-item-subtitle">{subtitle}</p>
			</div>
		</div>
		<button type="button" aria-label="Close item details" onclick={onClose}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>

	<dl class="facts">
		{#if item.creatorName}
			<div>
				<dt>
					<Icon svg={userCircleIcon} size={16} aria-hidden="true" />Creator
				</dt>
				<dd>
					<span class="inline-flex min-w-0 items-center gap-1"
						>{item.creatorName}{#if item.creatorHasVerifiedBadge}<Icon
								svg={sealCheckIcon}
								class="shrink-0 text-accent-strong"
								size={14}
								aria-label="Verified" />{/if}</span>
				</dd>
			</div>
		{/if}
		{#if market}
			<div data-tone={market.isForSale ? undefined : "muted"}>
				<dt><Icon svg={storefrontIcon} size={16} aria-hidden="true" />Price</dt>
				<dd>
					<span class="font-mono"
						>{market.isForSale
							? market.priceInRobux === 0
								? "Free"
								: formatRobux(market.priceInRobux, false)
							: "Off sale"}</span>
				</dd>
			</div>
			{#if market.lowestResalePriceInRobux !== null}
				<div>
					<dt>
						<Icon svg={tagIcon} size={16} aria-hidden="true" />Lowest resale
					</dt>
					<dd>
						<span class="font-mono"
							>{formatRobux(
								market.lowestResalePriceInRobux,
								false,
							)}</span>
					</dd>
				</div>
			{/if}
		{/if}
		{#if item.recentAveragePrice !== null}
			<div>
				<dt>
					<Icon svg={trendUpIcon} size={16} aria-hidden="true" />Recent
					average price
				</dt>
				<dd>
					<span class="font-mono"
						>{formatRobux(item.recentAveragePrice, false)}</span>
				</dd>
			</div>
		{/if}
		{#if item.serialNumber !== null}
			<div>
				<dt>
					<Icon svg={barcodeIcon} size={16} aria-hidden="true" />Serial number
				</dt>
				<dd>
					<span class="font-mono"
						>#{exactNumbers.format(item.serialNumber)}</span>
				</dd>
			</div>
		{/if}
		{#if market?.favoriteCount != null}
			<div>
				<dt><Icon svg={heartIcon} size={16} aria-hidden="true" />Favorites</dt>
				<dd>
					<span class="font-mono"
						>{exactNumbers.format(market.favoriteCount)}</span>
				</dd>
			</div>
		{/if}
		{#if badge}
			<div>
				<dt><Icon svg={trophyIcon} size={16} aria-hidden="true" />Awarded</dt>
				<dd>
					<span class="font-mono"
						>{exactNumbers.format(badge.awardedCount)}</span>
					<small
						>Won by {percent.format(badge.pastDayWinFraction)} of players yesterday</small>
				</dd>
			</div>
		{/if}
		<div>
			<dt><Icon svg={hashIcon} size={16} aria-hidden="true" />ID</dt>
			<dd><span class="font-mono select-text">{item.itemId}</span></dd>
		</div>
	</dl>

	{#if item.description}
		<p class="mt-3 max-h-40 text-block overflow-y-auto" data-variant="prose">
			{item.description}
		</p>
	{/if}

	{#if badge && badge.awardingPlaceId > 0}
		<div class="dialog-actions">
			<button
				class="button"
				type="button"
				onclick={() => onSearchPlace(badge.awardingPlaceId)}>
				<Icon svg={magnifyingGlassIcon} size={16} aria-hidden="true" />
				Find game in Games
			</button>
		</div>
	{/if}
</Dialog>
