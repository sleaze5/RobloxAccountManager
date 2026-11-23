<script lang="ts">
	import checkCircleFillIcon from "@phosphor-icons/core/fill/check-circle-fill.svg?raw"
	import hourglassIcon from "@phosphor-icons/core/regular/hourglass.svg?raw"
	import tShirtIcon from "@phosphor-icons/core/regular/t-shirt.svg?raw"
	import timerIcon from "@phosphor-icons/core/regular/timer.svg?raw"
	import type { InventoryItem } from "../backend/bridge"
	import ItemTile from "../shared/ItemTile.svelte"
	import type { ItemTileBadge } from "../shared/ItemTile.svelte"
	import { inventoryTypeLabel } from "./inventory-categories"
	import type { InventoryCategoryOption } from "./inventory-categories"
	import { formatRobux, marketPriceLabel } from "./inventory-prices"

	let {
		item,
		category,
		isWorn,
		isOwned,
		extraBadges = [],
		isDisabled = false,
		actionLabel,
		onOpen,
	}: {
		item: InventoryItem
		category: InventoryCategoryOption
		isWorn: boolean
		// Whether the account that the page acts for owns the item.
		isOwned: boolean
		// extraBadges are the marks that only one page knows, after the shared
		// ones.
		extraBadges?: ItemTileBadge[]
		// isDisabled keeps the tile focusable, so that its tooltip can say why
		// the action is unavailable.
		isDisabled?: boolean
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
		// The marks of the item, in priority order. The last ones clip first
		// when the tile is narrow.
		shownBadges = $derived<ItemTileBadge[]>(
			[
				isOwned && {
					label: "You own this",
					icon: checkCircleFillIcon,
					tone: "text-success" as const,
				},
				isWorn && {
					label: "Worn",
					icon: tShirtIcon,
					tone: "text-success" as const,
				},
				item.market?.isForSale &&
					item.market.offSaleDeadlineMs > 0 && {
						label: "Leaves sale soon",
						icon: timerIcon,
						tone: "text-warning" as const,
					},
				item.market?.hasTimedOptions && {
					label: "Timed options: can be rented",
					icon: hourglassIcon,
					tone: "text-fg-muted" as const,
				},
				...extraBadges,
			].filter((badge): badge is ItemTileBadge => !!badge),
		),
		isUnpriced = $derived(
			!!item.market &&
				!item.market.isForSale &&
				item.market.lowestResalePriceInRobux === null &&
				item.recentAveragePrice === null,
		)
</script>

<ItemTile
	{name}
	imageUrl={item.imageUrl}
	fallbackIcon={category.icon}
	badges={shownBadges}
	cornerLabel={item.serialNumber === null ? undefined : serialNumberLabel}
	limitedLabel={item.market?.isLimited ? "Limited" : ""}
	detail={typeLabel}
	{priceLabel}
	isPriceMuted={isUnpriced}
	{isDisabled}
	{actionLabel}
	onOpen={() => onOpen(item)} />

{#snippet serialNumberLabel()}<span class="sr-only"
		>Serial number
	</span>#{exactNumbers.format(item.serialNumber ?? 0)}{/snippet}
