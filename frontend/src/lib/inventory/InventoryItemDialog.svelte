<script lang="ts">
	import arrowsLeftRightIcon from "@phosphor-icons/core/regular/arrows-left-right.svg?raw"
	import barcodeIcon from "@phosphor-icons/core/regular/barcode.svg?raw"
	import calendarDotsIcon from "@phosphor-icons/core/regular/calendar-dots.svg?raw"
	import checkCircleFillIcon from "@phosphor-icons/core/fill/check-circle-fill.svg?raw"
	import clockCounterClockwiseIcon from "@phosphor-icons/core/regular/clock-counter-clockwise.svg?raw"
	import diamondIcon from "@phosphor-icons/core/regular/diamond.svg?raw"
	import fingerprintIcon from "@phosphor-icons/core/regular/fingerprint.svg?raw"
	import gameControllerIcon from "@phosphor-icons/core/regular/game-controller.svg?raw"
	import heartFillIcon from "@phosphor-icons/core/fill/heart-fill.svg?raw"
	import heartIcon from "@phosphor-icons/core/regular/heart.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import mapPinIcon from "@phosphor-icons/core/regular/map-pin.svg?raw"
	import packageIcon from "@phosphor-icons/core/regular/package.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import shoppingCartIcon from "@phosphor-icons/core/regular/shopping-cart.svg?raw"
	import shapesIcon from "@phosphor-icons/core/regular/shapes.svg?raw"
	import stackIcon from "@phosphor-icons/core/regular/stack.svg?raw"
	import storefrontIcon from "@phosphor-icons/core/regular/storefront.svg?raw"
	import tShirtIcon from "@phosphor-icons/core/regular/t-shirt.svg?raw"
	import tagIcon from "@phosphor-icons/core/regular/tag.svg?raw"
	import timerIcon from "@phosphor-icons/core/regular/timer.svg?raw"
	import trashIcon from "@phosphor-icons/core/regular/trash.svg?raw"
	import trendUpIcon from "@phosphor-icons/core/regular/trend-up.svg?raw"
	import trophyIcon from "@phosphor-icons/core/regular/trophy.svg?raw"
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
	import Timestamp from "../shared/Timestamp.svelte"
	import { relativeTime } from "../shared/timestamp"
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
		accountUsername,
		isOwnInventory,
		isWorn,
		onSearchPlace,
		onClose,
	}: {
		item: InventoryItem
		category: InventoryCategoryOption
		inventory: InventoryState
		username: string
		// accountUsername names the account that buys and favorites items.
		accountUsername: string
		isOwnInventory: boolean
		isWorn: boolean
		onSearchPlace: (placeId: number) => void
		onClose: () => void
	} = $props()

	let confirmation = $state<"deletion" | "purchase" | null>(null),
		card = $state<HTMLDivElement | undefined>(undefined)

	const exactNumbers = new Intl.NumberFormat("en-US"),
		percent = new Intl.NumberFormat("en-US", {
			style: "percent",
			maximumFractionDigits: 1,
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
		metaClass =
			"flex min-w-0 flex-wrap items-center gap-1 font-mono text-2xs wrap-anywhere text-fg-muted",
		sectionHeadingClass = "text-xs font-semibold text-fg-muted",
		name = $derived(item.name || "Unnamed item"),
		typeLabel = $derived(inventoryTypeLabel(item.typeName) || category.label),
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
		isBusy = $derived(
			inventory.isSavingFavorite ||
				inventory.isDeletingItem ||
				inventory.isBuyingItem,
		),
		// Roblox sells new copies of catalog items and game passes. The button
		// stays disabled while the ownership check runs, and leaves once the
		// account owns the item.
		canBuy = $derived(
			!isOwnInventory &&
				isOwnedByAccount !== true &&
				!!market?.isForSale &&
				(isCatalogItem ||
					category.id === InventoryCategory.InventoryGamePasses) &&
				market.saleLocation !== InventorySaleLocation.SaleInExperiences,
		),
		isPurchasePending = $derived(inventory.pendingPurchaseKeys.has(detailsKey)),
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
		),
		hasMarketSection = $derived(
			!!market || item.recentAveragePrice !== null || item.serialNumber !== null,
		),
		// null while the balance loads or when it is unavailable.
		balanceAfterPurchase = $derived(
			typeof inventory.robuxBalance === "number" && market
				? inventory.robuxBalance - market.priceInRobux
				: null,
		),
		isShortOfRobux = $derived(
			balanceAfterPurchase !== null && balanceAfterPurchase < 0,
		)

	// The dialog keeps its card while it switches between the details and the
	// deletion confirmation, so focus moves to the control that replaced the
	// one the user pressed.
	async function showConfirmation(
		shownConfirmation: "deletion" | "purchase" | null,
	): Promise<void> {
		const previousConfirmation = confirmation
		inventory.clearItemActionError()
		confirmation = shownConfirmation
		if (shownConfirmation === "purchase") void inventory.loadRobuxBalance()
		await tick()
		// A purchase removes its button, so focus falls back to the close button.
		const focusTarget =
			card?.querySelector<HTMLButtonElement>(
				shownConfirmation
					? "[data-cancel-confirmation]"
					: `[data-begin-${previousConfirmation}]`,
			) ?? card?.querySelector<HTMLButtonElement>("[data-close-item-details]")
		focusTarget?.focus()
	}

	async function buy(): Promise<void> {
		if (!market) return
		if (await inventory.buyItem(item, market.priceInRobux)) {
			await showConfirmation(null)
		}
	}

	function close(): void {
		if (inventory.isDeletingItem || inventory.isBuyingItem) return
		if (confirmation) void showConfirmation(null)
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

{#snippet fact(icon: string, label: string, value: string, isMono = false)}
	<div>
		<dt><Icon svg={icon} size={16} aria-hidden="true" />{label}</dt>
		<dd><span class={[isMono && "font-mono select-text"]}>{value}</span></dd>
	</div>
{/snippet}

{#snippet missingDetail()}
	{#if details === undefined}
		<Skeleton className="w-28" />
	{:else if details === null}
		<span class="text-fg-muted">Unavailable</span>
	{:else}
		<span class="text-fg-muted">N/A</span>
	{/if}
{/snippet}

{#snippet detailFact(icon: string, label: string, value: string | undefined)}
	<div>
		<dt><Icon svg={icon} size={16} aria-hidden="true" />{label}</dt>
		<dd>
			{#if value}
				<span class="font-mono select-text">{value}</span>
			{:else}
				{@render missingDetail()}
			{/if}
		</dd>
	</div>
{/snippet}

{#snippet dateFact(icon: string, label: string, valueMs: number | undefined)}
	<div>
		<dt><Icon svg={icon} size={16} aria-hidden="true" />{label}</dt>
		<dd>
			{#if valueMs}
				<Timestamp value={valueMs} />
				<small>{relativeTime(valueMs)}</small>
			{:else}
				{@render missingDetail()}
			{/if}
		</dd>
	</div>
{/snippet}

<Dialog
	class={confirmation ? "max-w-md text-left" : "max-w-2xl text-left"}
	role={confirmation ? "alertdialog" : "dialog"}
	aria-labelledby={confirmation
		? "inventory-confirmation-title"
		: "inventory-item-title"}
	aria-describedby={confirmation ? "inventory-confirmation-description" : undefined}
	bind:card
	onBackdropClick={close}>
	{#if confirmation}
		{@const isWorking = inventory.isDeletingItem || inventory.isBuyingItem}
		{@const price = market?.priceInRobux ?? 0}
		<div class="dialog-heading">
			<div>
				{#if confirmation === "deletion"}
					<h2 id="inventory-confirmation-title">Delete item?</h2>
					<p id="inventory-confirmation-description">
						{name} will be removed from @{username}'s inventory. Robux spent
						on it is not refunded, and this can't be undone.
					</p>
				{:else if price === 0}
					<h2 id="inventory-confirmation-title">Get item?</h2>
					<p id="inventory-confirmation-description">
						@{accountUsername} gets {name} for free.
					</p>
				{:else}
					<h2 id="inventory-confirmation-title">Buy item?</h2>
					<p id="inventory-confirmation-description">
						@{accountUsername} buys {name}. Roblox doesn't refund Robux
						spent on items.
					</p>
				{/if}
			</div>
			<button
				type="button"
				aria-label="Close confirmation"
				disabled={isWorking}
				onclick={close}>
				<Icon svg={xIcon} size={14} aria-hidden="true" />
			</button>
		</div>
		{#if confirmation === "purchase" && price > 0}
			<div
				class="flex items-center gap-3 rounded-lg border border-line-subtle bg-fill p-2">
				<span
					class="grid size-12 shrink-0 place-items-center overflow-hidden rounded-md bg-raised"
					aria-hidden="true">
					<ProfileImage url={imageUrl} fallbackIcon={category.icon} />
				</span>
				<div class="min-w-0">
					<p class="truncate text-sm font-medium text-fg">{name}</p>
					<p class="truncate text-xs text-fg-muted">{typeLabel}</p>
				</div>
			</div>
			<dl class="mt-3 flex flex-col gap-1.5 px-1 text-xs">
				<div class="flex items-baseline justify-between gap-3">
					<dt class="text-fg-muted">Price</dt>
					<dd class="font-mono text-fg">{formatRobux(price, false)}</dd>
				</div>
				<div class="flex items-baseline justify-between gap-3">
					<dt class="min-w-0 truncate text-fg-muted">
						@{accountUsername}'s Robux
					</dt>
					<dd class="font-mono text-fg">
						{#if inventory.robuxBalance === undefined}
							<Skeleton
								className="w-16"
								label="Loading the Robux balance" />
						{:else if inventory.robuxBalance === null}
							<span class="font-sans text-fg-muted">Unavailable</span>
						{:else}
							{formatRobux(inventory.robuxBalance, false)}
						{/if}
					</dd>
				</div>
				<div
					class="flex items-baseline justify-between gap-3 border-t border-line-subtle pt-1.5">
					<dt class="font-medium text-fg">After purchase</dt>
					<dd
						class={[
							"font-mono font-semibold",
							isShortOfRobux ? "text-danger-muted" : "text-fg",
						]}>
						{#if inventory.robuxBalance === undefined}
							<Skeleton
								className="w-16"
								label="Loading the Robux balance" />
						{:else if balanceAfterPurchase === null}
							<span class="font-sans font-normal text-fg-muted"
								>Unavailable</span>
						{:else}
							{formatRobux(balanceAfterPurchase, false)}
						{/if}
					</dd>
				</div>
			</dl>
			{#if isShortOfRobux && balanceAfterPurchase !== null}
				<p class="mt-3 notice-error" role="alert">
					@{accountUsername} needs {formatRobux(-balanceAfterPurchase, false)}
					more to buy this item.
				</p>
			{/if}
		{/if}
		{#if inventory.itemActionError}
			<p class="mt-3 notice-error" role="alert">{inventory.itemActionError}</p>
		{/if}
		<div class="dialog-actions">
			<button
				class="button"
				type="button"
				data-cancel-confirmation
				disabled={isWorking}
				onclick={close}>Cancel</button>
			{#if confirmation === "deletion"}
				<button
					class="button-danger"
					type="button"
					aria-busy={inventory.isDeletingItem}
					disabled={isWorking}
					onclick={() => void inventory.deleteItem(item)}
					>{inventory.isDeletingItem ? "Deleting…" : "Delete item"}</button>
			{:else}
				<button
					class="button-primary"
					type="button"
					aria-busy={inventory.isBuyingItem}
					disabled={isWorking || isShortOfRobux}
					onclick={() => void buy()}
					>{inventory.isBuyingItem
						? "Buying…"
						: price === 0
							? "Get item"
							: `Buy for ${formatRobux(price, false)}`}</button>
			{/if}
		</div>
	{:else}
		<div class="flex flex-col gap-4">
			<div class="flex items-start gap-4">
				<span
					class="grid size-48 shrink-0 place-items-center overflow-hidden rounded-lg border border-line-subtle bg-raised"
					aria-hidden="true">
					<span
						class={[
							"grid place-items-center",
							isSmallIcon ? "size-36" : "size-full",
						]}>
						<ProfileImage url={imageUrl} fallbackIcon={category.icon} />
					</span>
				</span>

				<div class="flex min-w-0 flex-1 flex-col gap-1">
					<div class="flex items-start gap-3">
						<h2
							id="inventory-item-title"
							class="min-w-0 flex-1 leading-7 tracking-tight wrap-anywhere">
							{name}
						</h2>
						<div class="flex shrink-0 items-center gap-1">
							{#if canBuy && market}
								<button
									class="button-primary"
									type="button"
									data-begin-purchase
									disabled={isBusy || isOwnedByAccount === undefined}
									onclick={() => void showConfirmation("purchase")}>
									<Icon
										svg={shoppingCartIcon}
										size={16}
										aria-hidden="true" />
									{market.priceInRobux === 0 ? "Get" : "Buy"}
								</button>
							{/if}
							{#if canFavorite}
								<button
									class="button-icon"
									type="button"
									aria-label="Favorite"
									aria-pressed={isFavorite ?? false}
									aria-busy={isCheckingFavorite ||
										inventory.isSavingFavorite}
									data-tooltip={isFavorite
										? "Remove from favorites"
										: "Add to favorites"}
									disabled={isBusy || isFavorite === undefined}
									onclick={() =>
										void inventory.setFavorite(item, !isFavorite)}>
									{#if isFavorite}
										<Icon
											svg={heartFillIcon}
											class="animate-pop text-accent-strong"
											size={16}
											aria-hidden="true" />
									{:else}
										<Icon
											svg={heartIcon}
											size={16}
											aria-hidden="true" />
									{/if}
								</button>
							{/if}
							{#if badge && badge.awardingPlaceId > 0}
								<button
									class="button-icon"
									type="button"
									aria-label="Find game in Games"
									data-tooltip="Find game in Games"
									onclick={() =>
										onSearchPlace(badge.awardingPlaceId)}>
									<Icon
										svg={magnifyingGlassIcon}
										size={16}
										aria-hidden="true" />
								</button>
							{/if}
							<button
								class="ml-1 button-close"
								type="button"
								data-close-item-details
								aria-label="Close item details"
								onclick={close}>
								<Icon svg={xIcon} size={14} aria-hidden="true" />
							</button>
						</div>
					</div>
					<div class={metaClass}>
						<span class="select-text">ID: {item.itemId}</span>
						<span class="text-fg-faint" aria-hidden="true">-</span>
						<span>{typeLabel}</span>
					</div>
					{#if item.creatorName}
						<div class={metaClass}>
							<span class="inline-flex items-center gap-1"
								>by {item.creatorName}{#if item.creatorHasVerifiedBadge}<Icon
										svg={sealCheckIcon}
										size={14}
										aria-label="Verified creator" />{/if}</span>
							{#if item.creatorId}
								<span class="text-fg-faint" aria-hidden="true">-</span>
								<span class="select-text"
									>{item.creatorType} ID: {item.creatorId}</span>
							{/if}
						</div>
					{/if}
					{#if market}
						<p
							class={[
								"inline-flex items-center gap-1 text-xs font-medium tabular-nums",
								market.isForSale ? "text-fg" : "text-fg-muted",
							]}>
							<Icon svg={storefrontIcon} size={14} aria-hidden="true" />
							{priceLabel}
						</p>
					{:else if badge}
						<p
							class="inline-flex items-center gap-1 text-xs font-medium text-fg tabular-nums">
							<Icon svg={trophyIcon} size={14} aria-hidden="true" />
							Won by {percent.format(badge.pastDayWinFraction)} of players yesterday
						</p>
					{/if}

					<ul class="mt-2 flex min-h-6 flex-wrap gap-1" aria-label="Status">
						{#if isOwnInventory}
							<li class="tag-pill-success">
								<Icon
									svg={checkCircleFillIcon}
									size={12}
									aria-hidden="true" />
								<span>Owned</span>
							</li>
						{:else if isOwnedByAccount === undefined}
							<li class="h-6 w-24 skeleton rounded-sm">
								<span class="sr-only"
									>Checking whether you own this</span>
							</li>
						{:else if isOwnedByAccount}
							<li class="tag-pill-success animate-pop">
								<Icon
									svg={checkCircleFillIcon}
									size={12}
									aria-hidden="true" />
								<span>You own this</span>
							</li>
						{/if}
						{#if isWorn}
							<li class="tag-pill-success">
								<Icon svg={tShirtIcon} size={12} aria-hidden="true" />
								<span>Wearing</span>
							</li>
						{/if}
						{#if market?.isLimited}
							<li class="tag-pill-warning">
								<Icon svg={diamondIcon} size={12} aria-hidden="true" />
								<span>Limited</span>
							</li>
						{/if}
					</ul>

					{#if isPurchasePending}
						<p class="text-xs text-fg-muted" role="status">
							Bought. It can take a minute to show up in the inventory.
						</p>
					{/if}
				</div>
			</div>

			{#if hasMarketSection}
				<section
					class="flex flex-col gap-3"
					aria-labelledby="inventory-item-market">
					<h3 id="inventory-item-market" class={sectionHeadingClass}>
						Market
					</h3>
					<dl class="facts">
						{#if market?.lowestResalePriceInRobux != null}
							{@render fact(
								tagIcon,
								"Lowest resale",
								formatRobux(market.lowestResalePriceInRobux, false),
							)}
						{/if}
						{#if item.recentAveragePrice !== null}
							{@render fact(
								trendUpIcon,
								"Recent average price",
								formatRobux(item.recentAveragePrice, false),
							)}
						{/if}
						{#if item.serialNumber !== null}
							{@render fact(
								barcodeIcon,
								"Serial number",
								`#${exactNumbers.format(item.serialNumber)}`,
							)}
						{/if}
						{#if stockLabel}
							{@render fact(packageIcon, "Stock", stockLabel)}
						{/if}
						{#if isCatalogItem && market}
							{@render fact(
								arrowsLeftRightIcon,
								"Tradable",
								market.isLimited ? "Yes" : "No",
							)}
						{/if}
						{#if saleLocationLabel}
							{@render fact(mapPinIcon, "Sold in", saleLocationLabel)}
						{/if}
						{#if market && market.offSaleDeadlineMs > 0}
							<div data-tone="warning">
								<dt>
									<Icon
										svg={timerIcon}
										size={16}
										aria-hidden="true" />Leaves sale
								</dt>
								<dd>
									<Timestamp value={market.offSaleDeadlineMs} />
									<small
										>{relativeTime(
											market.offSaleDeadlineMs,
										)}</small>
								</dd>
							</div>
						{/if}
						{#if market?.favoriteCount != null}
							{@render fact(
								heartIcon,
								"Favorites",
								exactNumbers.format(market.favoriteCount),
							)}
						{/if}
					</dl>
				</section>
			{/if}

			<section
				class="flex flex-col gap-3"
				aria-labelledby="inventory-item-details">
				<h3 id="inventory-item-details" class={sectionHeadingClass}>Details</h3>
				<dl class="facts">
					{@render fact(shapesIcon, "Type", typeLabel)}
					{#if badge}
						{@render fact(
							trophyIcon,
							"Awarded",
							exactNumbers.format(badge.awardedCount),
						)}
						<div>
							<dt>
								<Icon
									svg={gameControllerIcon}
									size={16}
									aria-hidden="true" />Game
							</dt>
							<dd>
								{#if details?.awardingGameName}
									<span>{details.awardingGameName}</span>
								{:else}
									{@render missingDetail()}
								{/if}
							</dd>
						</div>
					{/if}
					{#if isBundle}
						<div>
							<dt>
								<Icon
									svg={stackIcon}
									size={16}
									aria-hidden="true" />Includes
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
									{@render missingDetail()}
								{/if}
							</dd>
						</div>
					{/if}
					{#if !isBundle}
						{@render dateFact(
							clockCounterClockwiseIcon,
							"Updated",
							details?.updatedAtMs,
						)}
					{/if}
					{@render dateFact(
						calendarDotsIcon,
						"Created",
						details?.createdAtMs,
					)}
					{#if isCatalogItem}
						{@render detailFact(
							fingerprintIcon,
							"Collectible item ID",
							details?.collectibleItemId,
						)}
						{#if !isBundle}
							{@render detailFact(
								fingerprintIcon,
								"Collectible product ID",
								details?.collectibleProductId,
							)}
						{/if}
					{/if}
				</dl>
			</section>

			{#if item.description}
				<section
					class="flex flex-col gap-3"
					aria-labelledby="inventory-item-description">
					<h3 id="inventory-item-description" class={sectionHeadingClass}>
						Description
					</h3>
					<p class="max-h-40 text-block overflow-y-auto" data-variant="prose">
						{item.description}
					</p>
				</section>
			{/if}
		</div>

		{#if inventory.itemActionError}
			<p class="mt-3 notice-error" role="alert">{inventory.itemActionError}</p>
		{/if}

		{#if canDelete}
			<div class="dialog-actions">
				<button
					class="mr-auto button"
					type="button"
					data-begin-deletion
					disabled={isBusy}
					onclick={() => void showConfirmation("deletion")}>
					<Icon svg={trashIcon} size={16} aria-hidden="true" />
					Delete…
				</button>
			</div>
		{/if}
	{/if}
</Dialog>
