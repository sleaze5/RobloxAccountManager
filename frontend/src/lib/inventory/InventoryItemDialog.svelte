<script lang="ts">
	import arrowsLeftRightIcon from "@phosphor-icons/core/regular/arrows-left-right.svg?raw"
	import arrowsOutIcon from "@phosphor-icons/core/regular/arrows-out.svg?raw"
	import barcodeIcon from "@phosphor-icons/core/regular/barcode.svg?raw"
	import caretLeftIcon from "@phosphor-icons/core/regular/caret-left.svg?raw"
	import calendarDotsIcon from "@phosphor-icons/core/regular/calendar-dots.svg?raw"
	import checkCircleFillIcon from "@phosphor-icons/core/fill/check-circle-fill.svg?raw"
	import clockCounterClockwiseIcon from "@phosphor-icons/core/regular/clock-counter-clockwise.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import diamondIcon from "@phosphor-icons/core/regular/diamond.svg?raw"
	import downloadSimpleIcon from "@phosphor-icons/core/regular/download-simple.svg?raw"
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
	import storefrontIcon from "@phosphor-icons/core/regular/storefront.svg?raw"
	import tShirtIcon from "@phosphor-icons/core/regular/t-shirt.svg?raw"
	import tagIcon from "@phosphor-icons/core/regular/tag.svg?raw"
	import timerIcon from "@phosphor-icons/core/regular/timer.svg?raw"
	import trashIcon from "@phosphor-icons/core/regular/trash.svg?raw"
	import trendUpIcon from "@phosphor-icons/core/regular/trend-up.svg?raw"
	import trophyIcon from "@phosphor-icons/core/regular/trophy.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { tick, untrack } from "svelte"
	import {
		CatalogItemType,
		InventoryCategory,
		InventorySaleLocation,
	} from "../backend/bridge"
	import type {
		GamePlace,
		InventoryItem,
		RelatedInventoryItem,
	} from "../backend/bridge"
	import Dialog from "../shared/Dialog.svelte"
	import FrameUnavailable from "../shared/FrameUnavailable.svelte"
	import ExpandedImageDialog from "../shared/ExpandedImageDialog.svelte"
	import Icon from "../shared/Icon.svelte"
	import ImageViewToggle from "../shared/ImageViewToggle.svelte"
	import type { ImageView } from "../shared/ImageViewToggle.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import RenderedModelView from "../shared/RenderedModelView.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import SkeletonTile from "../shared/SkeletonTile.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
	import { relativeTime } from "../shared/timestamp"
	import {
		inventoryCategoryOptionOf,
		inventoryTypeLabel,
	} from "./inventory-categories"
	import type { InventoryCategoryOption } from "./inventory-categories"
	import type { Account } from "../accounts/account-model"
	import InventoryItemTile from "./InventoryItemTile.svelte"
	import { formatRobux, salePriceLabel } from "./inventory-prices"
	import PurchaseAccountActions from "./PurchaseAccountActions.svelte"
	import PurchaseItemSummary from "./PurchaseItemSummary.svelte"
	import { categoryItemKey, inventoryItemKey } from "./inventory-state.svelte"
	import type { InventoryState } from "./inventory-state.svelte"

	let {
		item: openedItem,
		category: openedCategory,
		inventory,
		username,
		account,
		isOwnInventory: isOpenedFromOwnInventory,
		isWorn: isOpenedItemWorn,
		onSearchPlace,
		onShowGamePass,
		onClose,
	}: {
		item: InventoryItem
		category: InventoryCategoryOption
		inventory: InventoryState
		username: string
		// The account that buys and favorites items.
		account: Account
		isOwnInventory: boolean
		isWorn: boolean
		onSearchPlace: (placeId: number) => void
		// Opens the pass in Games. Pages without game passes omit it.
		onShowGamePass?: (place: GamePlace, gamePassId: number) => void
		onClose: () => void
	} = $props()

	// The related items that the user opened inside the dialog, the latest
	// last. Back returns to the one before.
	let openedRelatedItems = $state.raw<
		{ item: InventoryItem; category: InventoryCategoryOption }[]
	>([])

	const item = $derived(openedRelatedItems.at(-1)?.item ?? openedItem),
		category = $derived(openedRelatedItems.at(-1)?.category ?? openedCategory),
		previousItemName = $derived(
			(openedRelatedItems.at(-2)?.item ?? openedItem).name || "Unnamed item",
		),
		// A related item belongs to no inventory that the page lists.
		isOwnInventory = $derived(
			isOpenedFromOwnInventory && openedRelatedItems.length === 0,
		),
		isWorn = $derived(
			openedRelatedItems.length === 0
				? isOpenedItemWorn
				: inventory.wornAssetIds.has(item.itemId),
		)

	let confirmation = $state<"deletion" | "purchase" | null>(null),
		card = $state<HTMLDivElement | undefined>(undefined),
		view = $state<ImageView>("2D"),
		isExpanded = $state(false),
		isTryingOn = $state(false)

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
		subtypeLabel = $derived(inventoryTypeLabel(item.typeName)),
		// The category names the kind of item and the type narrows it, as in
		// "Accessories; Shoulder".
		typeLabel = $derived(
			subtypeLabel && subtypeLabel !== category.label
				? [category.label, subtypeLabel].join("; ")
				: category.label,
		),
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
		hasModel = $derived(!!details?.hasModel),
		canTryOn = $derived(!!details?.canTryOn),
		canHaveRelatedItems = $derived(!!details?.canHaveRelatedItems),
		canDownload = $derived(!!details?.canDownload),
		// undefined while the related items load, and null when they failed.
		relatedItems = $derived(inventory.relatedItemsByItemKey.get(detailsKey)),
		// Try on stays on while the view switches between 2D and 3D.
		isTryOnShown = $derived(canTryOn && isTryingOn),
		// An item without a model of its own has a 3D view only on the avatar.
		hasThreeDimensionalView = $derived(hasModel || isTryOnShown),
		shownView = $derived<ImageView>(hasThreeDimensionalView ? view : "2D"),
		isModelShown = $derived(shownView === "3D"),
		tryOnLabel = $derived(`@${account.username}'s avatar wearing ${name}`),
		tryOnImageUrl = $derived(inventory.tryOnImage.loadedFor(detailsKey) ?? ""),
		tryOnError = $derived(inventory.tryOnImage.errorFor(detailsKey)),
		tryOnModel = $derived(inventory.tryOnModel.loadedFor(detailsKey)),
		tryOnModelError = $derived(inventory.tryOnModel.errorFor(detailsKey)),
		// The model of another item stays loaded until this item's model loads.
		itemModel = $derived(inventory.itemModel.loadedFor(detailsKey)),
		itemModelError = $derived(inventory.itemModel.errorFor(detailsKey)),
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
		priceLabel = $derived(market ? salePriceLabel(market, false) : ""),
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
		)

	$effect(() => {
		const shownItem = item,
			shownCategory = category.id
		if (canHaveRelatedItems) {
			untrack(() => void inventory.loadRelatedItems(shownItem, shownCategory))
		}
	})

	// showOpenedRelatedItems shows the last of the opened related items, or the
	// item that the dialog opened with, from its top.
	async function showOpenedRelatedItems(
		nextOpenedRelatedItems: {
			item: InventoryItem
			category: InventoryCategoryOption
		}[],
	): Promise<void> {
		openedRelatedItems = nextOpenedRelatedItems
		view = "2D"
		isTryingOn = isExpanded = false
		inventory.clearItemActionError()
		await tick()
		card?.querySelector<HTMLElement>("#inventory-item-title")?.scrollIntoView({
			block: "nearest",
		})
		card?.querySelector<HTMLElement>(
			nextOpenedRelatedItems.length > 0
				? "[data-item-back]"
				: "[data-close-item-details]",
		)?.focus()
	}

	function openRelatedItem(relatedItem: RelatedInventoryItem): void {
		const relatedCategory = inventoryCategoryOptionOf(relatedItem.category)
		inventory.loadItemDetails(relatedItem.item, relatedCategory.id, true)
		void inventory.checkFavorite(relatedItem.item)
		void showOpenedRelatedItems([
			...openedRelatedItems,
			{ item: relatedItem.item, category: relatedCategory },
		])
	}

	// What the frame shows loads when it opens: the avatar wearing the item
	// while Try on is on, or the item, in the shown view.
	$effect(() => {
		const shownItem = item,
			shownCategory = category.id,
			isTryingOnInModel = isTryOnShown && isModelShown,
			isTryingOnInImage = isTryOnShown && !isModelShown,
			isItemModelShown = !isTryOnShown && isModelShown
		untrack(() => {
			if (isTryingOnInModel)
				void inventory.loadTryOnModel(shownItem, shownCategory)
			if (isTryingOnInImage) void inventory.tryOnItem(shownItem, shownCategory)
			if (isItemModelShown) void inventory.loadItemModel(shownItem, shownCategory)
		})
	})

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
		if (await inventory.buyItem(item, category.id, market.priceInRobux)) {
			await showConfirmation(null)
		}
	}

	async function showGamePass(
		show: (place: GamePlace, gamePassId: number) => void,
	): Promise<void> {
		const gamePassId = item.itemId,
			place = await inventory.findGamePassPlace(gamePassId)
		if (place && item.itemId === gamePassId) show(place, gamePassId)
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

{#snippet imageFrame()}
	{#if isTryOnShown && isModelShown}
		<RenderedModelView
			model={tryOnModel}
			error={tryOnModelError}
			label={tryOnLabel}
			onRetry={() => void inventory.loadTryOnModel(item, category.id)} />
	{:else if isTryOnShown}
		{#if tryOnImageUrl}
			{#key tryOnImageUrl}
				<img
					class="relative size-full animate-pop object-contain"
					src={tryOnImageUrl}
					alt={tryOnLabel}
					decoding="async"
					draggable="false"
					referrerpolicy="no-referrer" />
			{/key}
		{:else if tryOnError}
			<FrameUnavailable
				icon={category.icon}
				message={tryOnError}
				onRetry={() => void inventory.tryOnItem(item, category.id)} />
		{:else}
			<span class="absolute inset-3 skeleton rounded-md" role="status">
				<span class="sr-only">Trying on the item</span>
			</span>
		{/if}
	{:else if isModelShown}
		<RenderedModelView
			model={itemModel}
			error={itemModelError}
			label={name}
			onRetry={() => void inventory.loadItemModel(item, category.id)} />
	{:else}
		<span
			class={["grid place-items-center", isSmallIcon ? "size-36" : "size-full"]}>
			<ProfileImage url={imageUrl} fallbackIcon={category.icon} />
		</span>
	{/if}
{/snippet}

{#snippet imageControls(isExpandedView: boolean)}
	{#if hasThreeDimensionalView}
		<ImageViewToggle bind:view />
	{/if}
	{#if canTryOn}
		<button
			class="button"
			type="button"
			data-tooltip={isTryOnShown
				? "Show the item without the avatar"
				: `Show the item on @${account.username}'s saved avatar`}
			onclick={() => (isTryingOn = !isTryingOn)}
			>{isTryOnShown ? "Take off" : "Try on"}</button>
	{/if}
	{#if !isExpandedView}
		<button
			class="button-icon"
			type="button"
			aria-label="Expand item image"
			data-tooltip="Expand"
			onclick={() => (isExpanded = true)}>
			<Icon svg={arrowsOutIcon} size={16} aria-hidden="true" />
		</button>
	{/if}
{/snippet}

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
	class={confirmation ? "max-w-md text-left" : "max-w-3xl text-left"}
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
					<p id="inventory-confirmation-description">This item is free.</p>
				{:else}
					<h2 id="inventory-confirmation-title">Buy item?</h2>
					<p id="inventory-confirmation-description">
						Roblox doesn't refund items.
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
		{#if confirmation === "purchase"}
			<PurchaseItemSummary
				{name}
				detail={typeLabel}
				{imageUrl}
				fallbackIcon={category.icon}
				priceInRobux={price} />
		{/if}
		{#if inventory.itemActionError}
			<p class="mt-3 notice-error" role="alert">{inventory.itemActionError}</p>
		{/if}
		{#if confirmation === "purchase"}
			<PurchaseAccountActions
				{account}
				priceInRobux={price}
				robuxBalance={inventory.robuxBalance}
				isBuying={inventory.isBuyingItem}
				onCancel={close}
				onBuy={() => void buy()} />
		{:else}
			<div class="dialog-actions">
				<button
					class="button"
					type="button"
					data-cancel-confirmation
					disabled={isWorking}
					onclick={close}>Cancel</button>
				<button
					class="button-danger"
					type="button"
					aria-busy={inventory.isDeletingItem}
					disabled={isWorking}
					onclick={() => void inventory.deleteItem(item)}
					>{inventory.isDeletingItem ? "Deleting…" : "Delete item"}</button>
			</div>
		{/if}
	{:else}
		<div class="flex flex-col gap-4">
			<div class="flex items-start gap-4">
				<div class="flex w-72 shrink-0 flex-col gap-2">
					<div
						class="relative grid size-72 place-items-center overflow-hidden rounded-lg border border-line-subtle bg-raised"
						aria-hidden={isModelShown || isTryOnShown ? undefined : true}>
						{@render imageFrame()}
					</div>
					<div class="flex items-center gap-2">
						{@render imageControls(false)}
					</div>
				</div>

				<div class="flex min-w-0 flex-1 flex-col gap-1">
					<div class="flex items-start gap-3">
						{#if openedRelatedItems.length > 0}
							<button
								class="-ml-1 button-ghost"
								type="button"
								data-item-back
								aria-label={`Back to ${previousItemName}`}
								data-tooltip={`Back to ${previousItemName}`}
								onclick={() =>
									void showOpenedRelatedItems(
										openedRelatedItems.slice(0, -1),
									)}>
								<Icon
									svg={caretLeftIcon}
									data-nudge="left"
									size={16}
									aria-hidden="true" />
							</button>
						{/if}
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
							{#if onShowGamePass && category.id === InventoryCategory.InventoryGamePasses}
								<button
									class="button-icon"
									type="button"
									aria-label="Show in Games"
									aria-busy={inventory.isFindingGamePassPlace}
									data-tooltip="Show in Games"
									disabled={inventory.isFindingGamePassPlace}
									onclick={() => void showGamePass(onShowGamePass)}>
									<Icon
										svg={inventory.isFindingGamePassPlace
											? circleNotchIcon
											: magnifyingGlassIcon}
										class={inventory.isFindingGamePassPlace
											? "spinner"
											: undefined}
										size={16}
										aria-hidden="true" />
								</button>
							{/if}
							{#if canDownload}
								<button
									class="button-icon"
									type="button"
									aria-label="Download"
									aria-busy={inventory.isDownloadingItem}
									data-tooltip="Download"
									disabled={inventory.isDownloadingItem}
									onclick={() =>
										void inventory.downloadItem(item, category.id)}>
									<Icon
										svg={inventory.isDownloadingItem
											? circleNotchIcon
											: downloadSimpleIcon}
										class={inventory.isDownloadingItem
											? "spinner"
											: undefined}
										size={16}
										aria-hidden="true" />
								</button>
							{/if}
							{#if canDelete}
								<button
									class="button"
									type="button"
									data-begin-deletion
									disabled={isBusy}
									onclick={() => void showConfirmation("deletion")}>
									<Icon
										svg={trashIcon}
										size={16}
										aria-hidden="true" />
									Delete
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

			{#if canHaveRelatedItems}
				<section
					class="flex flex-col gap-3"
					aria-labelledby="inventory-item-related">
					<h3
						id="inventory-item-related"
						class={["flex items-center gap-2", sectionHeadingClass]}>
						{isBundle ? "Includes" : "Dependencies"}
						{#if relatedItems?.length}
							<span class="font-mono text-2xs font-normal text-fg-faint"
								>{relatedItems.length}</span>
						{/if}
					</h3>
					{#if relatedItems === undefined}
						<ul class="grid grid-cols-5 gap-2" aria-busy="true">
							{#each Array.from({ length: 5 }, (_unused, index) => index) as index (index)}
								<SkeletonTile />
							{/each}
						</ul>
						<span class="sr-only" role="status"
							>Loading the {isBundle
								? "items in the bundle"
								: "dependencies"}</span>
					{:else if relatedItems === null}
						<div class="flex items-center gap-2 notice-error" role="alert">
							<span class="min-w-0 flex-1"
								>The {isBundle
									? "items in this bundle"
									: "dependencies"} couldn't be loaded.</span>
							<button
								class="button"
								type="button"
								onclick={() =>
									void inventory.loadRelatedItems(item, category.id)}
								>Try again</button>
						</div>
					{:else if relatedItems.length === 0}
						<p class="text-xs text-fg-muted">
							{isBundle
								? "This bundle lists no items."
								: "This item uses no other assets that this account can see."}
						</p>
					{:else}
						<ul class="grid animate-enter grid-cols-5 gap-2">
							{#each relatedItems as relatedItem (relatedItem.item.itemId)}
								<li class="min-w-0">
									<InventoryItemTile
										item={relatedItem.item}
										category={inventoryCategoryOptionOf(
											relatedItem.category,
										)}
										isWorn={inventory.wornAssetIds.has(
											relatedItem.item.itemId,
										)}
										actionLabel="Open details"
										onOpen={() => openRelatedItem(relatedItem)} />
								</li>
							{/each}
						</ul>
					{/if}
				</section>
			{/if}
		</div>

		{#if inventory.itemActionError}
			<p class="mt-3 notice-error" role="alert">{inventory.itemActionError}</p>
		{/if}
	{/if}
</Dialog>

{#if isExpanded && !confirmation}
	<ExpandedImageDialog label={name} onClose={() => (isExpanded = false)}>
		{#snippet frame()}
			{@render imageFrame()}
		{/snippet}
		{#snippet controls()}
			{@render imageControls(true)}
		{/snippet}
	</ExpandedImageDialog>
{/if}
