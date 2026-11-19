<script lang="ts" module>
	export interface ItemTileBadge {
		// label names the mark in its tooltip and for screen readers.
		label: string
		icon: string
		tone: "text-success" | "text-warning" | "text-fg" | "text-fg-muted"
	}
</script>

<script lang="ts">
	import checkCircleFillIcon from "@phosphor-icons/core/fill/check-circle-fill.svg?raw"
	import hourglassIcon from "@phosphor-icons/core/regular/hourglass.svg?raw"
	import tShirtIcon from "@phosphor-icons/core/regular/t-shirt.svg?raw"
	import timerIcon from "@phosphor-icons/core/regular/timer.svg?raw"
	import type { InventoryItem } from "../backend/bridge"
	import Icon from "../shared/Icon.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
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

<button
	class="group/tile flex h-full w-full min-w-0 flex-col gap-1 rounded-lg border border-line-subtle bg-surface p-1 text-left transition-colors duration-100 hover:border-line hover:bg-hover aria-disabled:cursor-not-allowed aria-disabled:opacity-45 aria-disabled:hover:border-line-subtle aria-disabled:hover:bg-surface"
	type="button"
	aria-disabled={isDisabled || undefined}
	data-tooltip={actionLabel}
	data-tooltip-strong={name}
	onclick={() => {
		if (!isDisabled) onOpen(item)
	}}>
	<span
		class="relative grid aspect-square w-full place-items-center overflow-hidden rounded-sm bg-raised">
		<span
			class="grid size-full place-items-center transition-transform duration-150 ease-out motion-safe:group-hover/tile:scale-105"
			aria-hidden="true">
			<ProfileImage url={item.imageUrl} fallbackIcon={category.icon} />
		</span>
		{#if item.serialNumber !== null}
			<span
				class="absolute top-1 left-1 max-w-full truncate rounded-sm border border-line-strong bg-overlay px-1 font-mono text-2xs leading-4 font-medium text-fg"
				><span class="sr-only">Serial number </span>#{exactNumbers.format(
					item.serialNumber,
				)}</span>
		{/if}
		{#if item.market?.isLimited || shownBadges.length > 0}
			<span class="absolute inset-x-1 bottom-1 flex items-end gap-1">
				{#if item.market?.isLimited}
					<!-- Roblox marks a limited item with a green label in the
					     bottom left corner of its image. -->
					<span
						class="shrink-0 rounded-sm bg-success px-1 text-2xs leading-4 font-semibold text-canvas"
						>Limited</span>
				{/if}
				{#if shownBadges.length > 0}
					<!-- Reversed, so that the first badge sits at the right edge
					     and the last ones clip first on a narrow tile. -->
					<span
						class="flex min-w-0 flex-1 flex-row-reverse gap-0.5 overflow-hidden">
						{#each shownBadges as badge (badge.label)}
							<span
								class={[
									"grid size-5 shrink-0 animate-pop place-items-center rounded-sm border border-line-strong bg-overlay",
									badge.tone,
								]}
								data-tooltip={badge.label}
								><Icon
									svg={badge.icon}
									size={12}
									aria-hidden="true" /><span class="sr-only"
									>{badge.label}</span
								></span>
						{/each}
					</span>
				{/if}
			</span>
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
