<script lang="ts">
	import arrowsLeftRightIcon from "@phosphor-icons/core/regular/arrows-left-right.svg?raw"
	import barcodeIcon from "@phosphor-icons/core/regular/barcode.svg?raw"
	import calendarPlusIcon from "@phosphor-icons/core/regular/calendar-plus.svg?raw"
	import checkCircleFillIcon from "@phosphor-icons/core/fill/check-circle-fill.svg?raw"
	import circleDashedIcon from "@phosphor-icons/core/regular/circle-dashed.svg?raw"
	import clockClockwiseIcon from "@phosphor-icons/core/regular/clock-clockwise.svg?raw"
	import fingerprintIcon from "@phosphor-icons/core/regular/fingerprint.svg?raw"
	import gameControllerIcon from "@phosphor-icons/core/regular/game-controller.svg?raw"
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import heartFillIcon from "@phosphor-icons/core/fill/heart-fill.svg?raw"
	import heartIcon from "@phosphor-icons/core/regular/heart.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import mapPinIcon from "@phosphor-icons/core/regular/map-pin.svg?raw"
	import packageIcon from "@phosphor-icons/core/regular/package.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import stackIcon from "@phosphor-icons/core/regular/stack.svg?raw"
	import tShirtIcon from "@phosphor-icons/core/regular/t-shirt.svg?raw"
	import timerIcon from "@phosphor-icons/core/regular/timer.svg?raw"
	import trashIcon from "@phosphor-icons/core/regular/trash.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { tick } from "svelte"
	import {
		CatalogItemType,
		InventoryCategory,
		InventorySaleLocation,
	} from "../backend/bridge"
	import type { InventoryItem } from "../backend/bridge"
	import Dialog from "../shared/Dialog.svelte"
	import Icon from "../shared/Icon.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import { inventoryTypeLabel } from "./inventory-categories"
	import type { InventoryCategoryOption } from "./inventory-categories"
	import { formatRobux } from "./inventory-prices"
	import { categoryItemKey, inventoryItemKey } from "./inventory-state.svelte"
	import type { InventoryState } from "./inventory-state.svelte"

	let {
		item,
		category,
		inventory,
		username,
		isOwnInventory,
		isWorn,
		onSearchPlace,
		onClose,
	}: {
		item: InventoryItem
		category: InventoryCategoryOption
		inventory: InventoryState
		username: string
		isOwnInventory: boolean
		isWorn: boolean
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
		dates = new Intl.DateTimeFormat(undefined, {
			dateStyle: "medium",
			timeStyle: "short",
		}),
		saleLocationLabels: Record<InventorySaleLocation, string> = {
			[InventorySaleLocation.$zero]: "",
			[InventorySaleLocation.SaleInMarketplace]: "Marketplace only",
			[InventorySaleLocation.SaleInMarketplaceAndAllExperiences]:
				"Marketplace and experiences",
			[InventorySaleLocation.SaleInMarketplaceAndSomeExperiences]:
				"Marketplace and some experiences",
			[InventorySaleLocation.SaleInExperiences]: "Experiences only",
		},
		name = $derived(item.name || "Unnamed item"),
		typeLabel = $derived(inventoryTypeLabel(item.typeName)),
		market = $derived(item.market),
		badge = $derived(item.badge),
		isCatalogItem = $derived(!!item.catalogItemType),
		isBundle = $derived(item.catalogItemType === CatalogItemType.CatalogBundle),
		// Roblox renders badges and game passes only as small icons.
		isSmallIcon = $derived(
			category.id === InventoryCategory.InventoryBadges ||
				category.id === InventoryCategory.InventoryGamePasses,
		),
		detailsKey = $derived(categoryItemKey(category.id, item.itemId)),
		// undefined while the details load, and null when they failed.
		details = $derived(inventory.detailsByItemKey.get(detailsKey)),
		isOwnedByAccount = $derived(inventory.isOwnedByItemKey.get(detailsKey)),
		imageUrl = $derived(details?.largeImageUrl || item.imageUrl),
		key = $derived(inventoryItemKey(item)),
		canFavorite = $derived(isCatalogItem && !!market),
		isFavorite = $derived(inventory.favoriteByItemKey.get(key)),
		isCheckingFavorite = $derived(inventory.checkingFavoriteKey === key),
		// Roblox refuses to delete limited items and items that Roblox created.
		canDelete = $derived(
			isOwnInventory &&
				category.isDeletable &&
				!market?.isLimited &&
				item.creatorName !== "Roblox",
		),
		isBusy = $derived(inventory.isSavingFavorite || inventory.isDeletingItem),
		priceLabel = $derived(
			!market
				? ""
				: !market.isForSale
					? "Off sale"
					: market.priceInRobux === 0
						? "Free"
						: formatRobux(market.priceInRobux, false),
		),
		stockLabel = $derived(
			market?.totalQuantity == null
				? ""
				: market.remainingQuantity == null
					? `${exactNumbers.format(market.totalQuantity)} copies`
					: `${exactNumbers.format(market.remainingQuantity)} of ${exactNumbers.format(market.totalQuantity)} left`,
		),
		saleLocationLabel = $derived(
			market ? saleLocationLabels[market.saleLocation] : "",
		)

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

{#snippet detailValue(value: string | undefined, isMono: boolean)}
	{#if details === undefined}
		<Skeleton className="w-32" />
	{:else if details === null}
		<span class="text-fg-muted">Unavailable</span>
	{:else if value}
		<span class={[isMono && "font-mono select-text"]}>{value}</span>
	{:else}
		<span class="text-fg-muted">N/A</span>
	{/if}
{/snippet}

{#snippet stat(label: string, value: string, isMuted: boolean)}
	<div class="min-w-0">
		<dt class="text-2xs text-fg-muted">{label}</dt>
		<dd
			class={[
				"truncate font-mono text-sm font-semibold",
				isMuted ? "text-fg-muted" : "text-fg",
			]}>
			{value}
		</dd>
	</div>
{/snippet}

<Dialog
	class={isConfirmingDeletion ? "max-w-md text-left" : "max-w-2xl text-left"}
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
		<div class="flex items-start gap-4">
			<span
				class="grid size-64 shrink-0 place-items-center overflow-hidden rounded-lg border border-line-subtle bg-raised"
				aria-hidden="true">
				<span
					class={[
						"grid place-items-center",
						isSmallIcon ? "size-36" : "size-full",
					]}>
					<ProfileImage url={imageUrl} fallbackIcon={category.icon} />
				</span>
			</span>

			<div class="flex min-w-0 flex-1 flex-col gap-3 self-stretch">
				<div class="flex items-start justify-between gap-3">
					<div class="min-w-0">
						<h2 id="inventory-item-title" class="wrap-anywhere">{name}</h2>
						<p
							id="inventory-item-subtitle"
							class="mt-1 text-sm leading-normal text-fg-muted">
							{typeLabel || category.label}
						</p>
					</div>
					<button
						class="button-close"
						type="button"
						aria-label="Close item details"
						onclick={close}>
						<Icon svg={xIcon} size={14} aria-hidden="true" />
					</button>
				</div>

				{#if item.creatorName}
					<p class="flex min-w-0 items-center gap-1 text-xs text-fg-muted">
						By <span class="truncate font-medium text-fg"
							>{item.creatorName}</span
						>{#if item.creatorHasVerifiedBadge}<Icon
								svg={sealCheckIcon}
								class="shrink-0 text-accent-strong"
								size={14}
								aria-label="Verified" />{/if}
					</p>
				{/if}

				<ul class="flex min-h-6 flex-wrap gap-1.5" aria-label="Status">
					{#if isOwnInventory}
						<li
							class="inline-flex h-6 items-center gap-1 rounded-sm border border-line-subtle bg-fill px-2 text-xs font-medium text-fg">
							<Icon
								svg={checkCircleFillIcon}
								class="text-success"
								size={14}
								aria-hidden="true" />Owned
						</li>
					{:else if isOwnedByAccount === undefined}
						<li class="h-6 w-24 skeleton rounded-sm">
							<span class="sr-only">Checking whether you own this</span>
						</li>
					{:else if isOwnedByAccount}
						<li
							class="inline-flex h-6 animate-pop items-center gap-1 rounded-sm border border-line-subtle bg-fill px-2 text-xs font-medium text-fg">
							<Icon
								svg={checkCircleFillIcon}
								class="text-success"
								size={14}
								aria-hidden="true" />You own this
						</li>
					{:else if isOwnedByAccount === false}
						<li
							class="inline-flex h-6 items-center gap-1 rounded-sm border border-line-subtle bg-fill px-2 text-xs font-medium text-fg-muted">
							<Icon
								svg={circleDashedIcon}
								class="text-fg-faint"
								size={14}
								aria-hidden="true" />You don't own this
						</li>
					{/if}
					{#if isWorn}
						<li
							class="inline-flex h-6 items-center gap-1 rounded-sm border border-line-subtle bg-fill px-2 text-xs font-medium text-success">
							<Icon
								svg={tShirtIcon}
								size={14}
								aria-hidden="true" />Wearing
						</li>
					{/if}
					{#if market?.isLimited}
						<li
							class="inline-flex h-6 items-center rounded-sm border border-line-subtle bg-fill px-2 text-xs font-medium text-warning">
							Limited
						</li>
					{/if}
				</ul>

				{#if market || badge || item.recentAveragePrice !== null}
					<dl
						class="mt-auto grid grid-cols-3 gap-3 rounded-lg border border-line-subtle bg-fill px-3 py-2">
						{#if market}
							{@render stat("Price", priceLabel, !market.isForSale)}
							{#if market.lowestResalePriceInRobux !== null}
								{@render stat(
									"Lowest resale",
									formatRobux(market.lowestResalePriceInRobux, false),
									false,
								)}
							{/if}
						{/if}
						{#if item.recentAveragePrice !== null}
							{@render stat(
								"Recent average",
								formatRobux(item.recentAveragePrice, false),
								false,
							)}
						{/if}
						{#if market?.favoriteCount != null}
							{@render stat(
								"Favorites",
								exactNumbers.format(market.favoriteCount),
								false,
							)}
						{/if}
						{#if badge}
							{@render stat(
								"Awarded",
								exactNumbers.format(badge.awardedCount),
								false,
							)}
							{@render stat(
								"Won yesterday",
								percent.format(badge.pastDayWinFraction),
								false,
							)}
						{/if}
					</dl>
				{/if}
			</div>
		</div>

		<dl class="mt-3 facts">
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
			{#if stockLabel}
				<div>
					<dt>
						<Icon svg={packageIcon} size={16} aria-hidden="true" />Stock
					</dt>
					<dd><span class="font-mono">{stockLabel}</span></dd>
				</div>
			{/if}
			{#if isCatalogItem && market}
				<div data-tone={market.isLimited ? undefined : "muted"}>
					<dt>
						<Icon
							svg={arrowsLeftRightIcon}
							size={16}
							aria-hidden="true" />Tradable
					</dt>
					<dd><span>{market.isLimited ? "Yes" : "No"}</span></dd>
				</div>
			{/if}
			{#if saleLocationLabel}
				<div>
					<dt>
						<Icon svg={mapPinIcon} size={16} aria-hidden="true" />Sold in
					</dt>
					<dd><span>{saleLocationLabel}</span></dd>
				</div>
			{/if}
			{#if market && market.offSaleDeadlineMs > 0}
				<div data-tone="warning">
					<dt>
						<Icon svg={timerIcon} size={16} aria-hidden="true" />Leaves sale
					</dt>
					<dd><span>{dates.format(market.offSaleDeadlineMs)}</span></dd>
				</div>
			{/if}
			{#if badge}
				<div>
					<dt>
						<Icon
							svg={gameControllerIcon}
							size={16}
							aria-hidden="true" />Game
					</dt>
					<dd>{@render detailValue(details?.awardingGameName, false)}</dd>
				</div>
			{/if}
			{#if isBundle}
				<div>
					<dt>
						<Icon svg={stackIcon} size={16} aria-hidden="true" />Includes
					</dt>
					<dd>
						{#if details?.includedItemNames?.length}
							<ul class="flex flex-wrap gap-1">
								{#each details.includedItemNames as includedItemName, index (index)}
									<li class="tag-pill">
										<span>{includedItemName}</span>
									</li>
								{/each}
							</ul>
						{:else}
							{@render detailValue("", false)}
						{/if}
					</dd>
				</div>
			{/if}
			<div>
				<dt>
					<Icon svg={calendarPlusIcon} size={16} aria-hidden="true" />Created
				</dt>
				<dd>
					{@render detailValue(
						details?.createdAtMs ? dates.format(details.createdAtMs) : "",
						false,
					)}
				</dd>
			</div>
			{#if !isBundle}
				<div>
					<dt>
						<Icon
							svg={clockClockwiseIcon}
							size={16}
							aria-hidden="true" />Updated
					</dt>
					<dd>
						{@render detailValue(
							details?.updatedAtMs
								? dates.format(details.updatedAtMs)
								: "",
							false,
						)}
					</dd>
				</div>
			{/if}
			<div>
				<dt><Icon svg={hashIcon} size={16} aria-hidden="true" />ID</dt>
				<dd><span class="font-mono select-text">{item.itemId}</span></dd>
			</div>
			{#if isCatalogItem}
				<div>
					<dt>
						<Icon
							svg={fingerprintIcon}
							size={16}
							aria-hidden="true" />Collectible item ID
					</dt>
					<dd>{@render detailValue(details?.collectibleItemId, true)}</dd>
				</div>
				{#if !isBundle}
					<div>
						<dt>
							<Icon
								svg={fingerprintIcon}
								size={16}
								aria-hidden="true" />Collectible product ID
						</dt>
						<dd>
							{@render detailValue(details?.collectibleProductId, true)}
						</dd>
					</div>
				{/if}
			{/if}
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
