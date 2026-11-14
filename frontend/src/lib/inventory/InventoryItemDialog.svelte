<script lang="ts">
	import barcodeIcon from "@phosphor-icons/core/regular/barcode.svg?raw"
	import heartFillIcon from "@phosphor-icons/core/fill/heart-fill.svg?raw"
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import heartIcon from "@phosphor-icons/core/regular/heart.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import storefrontIcon from "@phosphor-icons/core/regular/storefront.svg?raw"
	import tagIcon from "@phosphor-icons/core/regular/tag.svg?raw"
	import trashIcon from "@phosphor-icons/core/regular/trash.svg?raw"
	import trendUpIcon from "@phosphor-icons/core/regular/trend-up.svg?raw"
	import trophyIcon from "@phosphor-icons/core/regular/trophy.svg?raw"
	import userCircleIcon from "@phosphor-icons/core/regular/user-circle.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { tick } from "svelte"
	import type { InventoryItem } from "../backend/bridge"
	import Dialog from "../shared/Dialog.svelte"
	import Icon from "../shared/Icon.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import { inventoryTypeLabel } from "./inventory-categories"
	import type { InventoryCategoryOption } from "./inventory-categories"
	import { formatRobux } from "./inventory-prices"
	import { inventoryItemKey } from "./inventory-state.svelte"
	import type { InventoryState } from "./inventory-state.svelte"

	let {
		item,
		category,
		inventory,
		username,
		isOwnInventory,
		onSearchPlace,
		onClose,
	}: {
		item: InventoryItem
		category: InventoryCategoryOption
		inventory: InventoryState
		username: string
		isOwnInventory: boolean
		onSearchPlace: (placeId: number) => void
		onClose: () => void
	} = $props()

	let isConfirmingDeletion = $state(false),
		card = $state<HTMLDivElement | undefined>(undefined)

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
		),
		key = $derived(inventoryItemKey(item)),
		canFavorite = $derived(!!item.catalogItemType && !!market),
		isFavorite = $derived(inventory.favoriteByItemKey.get(key)),
		isCheckingFavorite = $derived(inventory.checkingFavoriteKey === key),
		// Roblox refuses to delete limited items and items that Roblox created.
		canDelete = $derived(
			isOwnInventory &&
				category.isDeletable &&
				!market?.isLimited &&
				item.creatorName !== "Roblox",
		),
		isBusy = $derived(inventory.isSavingFavorite || inventory.isDeletingItem)

	// The dialog keeps its card while it switches between the details and the
	// deletion confirmation, so focus moves to the control that replaced the
	// one the user pressed.
	async function showDeletionConfirmation(isShown: boolean): Promise<void> {
		inventory.clearItemActionError()
		isConfirmingDeletion = isShown
		await tick()
		card?.querySelector<HTMLButtonElement>(
			isShown ? "[data-cancel-deletion]" : "[data-begin-deletion]",
		)?.focus()
	}

	function close(): void {
		if (inventory.isDeletingItem) return
		if (isConfirmingDeletion) void showDeletionConfirmation(false)
		else onClose()
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") {
			event.preventDefault()
			close()
		}
	}} />

<Dialog
	class="max-w-md text-left"
	role={isConfirmingDeletion ? "alertdialog" : "dialog"}
	aria-labelledby={isConfirmingDeletion
		? "inventory-delete-title"
		: "inventory-item-title"}
	aria-describedby={isConfirmingDeletion
		? "inventory-delete-description"
		: "inventory-item-subtitle"}
	bind:card
	onBackdropClick={close}>
	{#if isConfirmingDeletion}
		<div class="dialog-heading">
			<div>
				<h2 id="inventory-delete-title">Delete item?</h2>
				<p id="inventory-delete-description">
					{name} will be removed from @{username}'s inventory. Robux spent on
					it is not refunded, and this can't be undone.
				</p>
			</div>
			<button
				type="button"
				aria-label="Close delete confirmation"
				disabled={inventory.isDeletingItem}
				onclick={close}>
				<Icon svg={xIcon} size={14} aria-hidden="true" />
			</button>
		</div>
		{#if inventory.itemActionError}
			<p class="notice-error" role="alert">{inventory.itemActionError}</p>
		{/if}
		<div class="dialog-actions">
			<button
				class="button"
				type="button"
				data-cancel-deletion
				disabled={inventory.isDeletingItem}
				onclick={close}>Cancel</button>
			<button
				class="button-danger"
				type="button"
				aria-busy={inventory.isDeletingItem}
				disabled={inventory.isDeletingItem}
				onclick={() => void inventory.deleteItem(item)}
				>{inventory.isDeletingItem ? "Deleting…" : "Delete item"}</button>
		</div>
	{:else}
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
			<button type="button" aria-label="Close item details" onclick={close}>
				<Icon svg={xIcon} size={14} aria-hidden="true" />
			</button>
		</div>

		<dl class="facts">
			{#if item.creatorName}
				<div>
					<dt>
						<Icon
							svg={userCircleIcon}
							size={16}
							aria-hidden="true" />Creator
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
					<dt>
						<Icon svg={storefrontIcon} size={16} aria-hidden="true" />Price
					</dt>
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
							<Icon svg={tagIcon} size={16} aria-hidden="true" />Lowest
							resale
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
						<Icon svg={barcodeIcon} size={16} aria-hidden="true" />Serial
						number
					</dt>
					<dd>
						<span class="font-mono"
							>#{exactNumbers.format(item.serialNumber)}</span>
					</dd>
				</div>
			{/if}
			{#if market?.favoriteCount != null}
				<div>
					<dt>
						<Icon svg={heartIcon} size={16} aria-hidden="true" />Favorites
					</dt>
					<dd>
						<span class="font-mono"
							>{exactNumbers.format(market.favoriteCount)}</span>
					</dd>
				</div>
			{/if}
			{#if badge}
				<div>
					<dt>
						<Icon svg={trophyIcon} size={16} aria-hidden="true" />Awarded
					</dt>
					<dd>
						<span class="font-mono"
							>{exactNumbers.format(badge.awardedCount)}</span>
						<small
							>Won by {percent.format(badge.pastDayWinFraction)} of players
							yesterday</small>
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

		{#if inventory.itemActionError}
			<p class="mt-3 notice-error" role="alert">{inventory.itemActionError}</p>
		{/if}

		{#if canDelete || canFavorite || (badge && badge.awardingPlaceId > 0)}
			<div class="dialog-actions">
				{#if canDelete}
					<button
						class="mr-auto button"
						type="button"
						data-begin-deletion
						disabled={isBusy}
						onclick={() => void showDeletionConfirmation(true)}>
						<Icon svg={trashIcon} size={16} aria-hidden="true" />
						Delete…
					</button>
				{/if}
				{#if canFavorite}
					<button
						class="button"
						type="button"
						aria-pressed={isFavorite ?? false}
						aria-busy={isCheckingFavorite || inventory.isSavingFavorite}
						data-tooltip={isFavorite
							? "Remove from favorites"
							: "Add to favorites"}
						disabled={isBusy || isFavorite === undefined}
						onclick={() => void inventory.setFavorite(item, !isFavorite)}>
						{#if isFavorite}
							<Icon
								svg={heartFillIcon}
								class="animate-pop text-accent-strong"
								size={16}
								aria-hidden="true" />
							Favorited
						{:else}
							<Icon svg={heartIcon} size={16} aria-hidden="true" />
							Favorite
						{/if}
					</button>
				{/if}
				{#if badge && badge.awardingPlaceId > 0}
					<button
						class="button"
						type="button"
						onclick={() => onSearchPlace(badge.awardingPlaceId)}>
						<Icon svg={magnifyingGlassIcon} size={16} aria-hidden="true" />
						Find game in Games
					</button>
				{/if}
			</div>
		{/if}
	{/if}
</Dialog>
