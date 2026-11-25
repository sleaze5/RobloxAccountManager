<script lang="ts">
	import { getProfilePreviews } from "../shared/profile-preview"
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import arrowsLeftRightIcon from "@phosphor-icons/core/regular/arrows-left-right.svg?raw"
	import arrowsOutIcon from "@phosphor-icons/core/regular/arrows-out.svg?raw"
	import barcodeIcon from "@phosphor-icons/core/regular/barcode.svg?raw"
	import caretLeftIcon from "@phosphor-icons/core/regular/caret-left.svg?raw"
	import calendarDotsIcon from "@phosphor-icons/core/regular/calendar-dots.svg?raw"
	import checkCircleFillIcon from "@phosphor-icons/core/fill/check-circle-fill.svg?raw"
	import clockCounterClockwiseIcon from "@phosphor-icons/core/regular/clock-counter-clockwise.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import diamondIcon from "@phosphor-icons/core/regular/diamond.svg?raw"
	import prohibitIcon from "@phosphor-icons/core/regular/prohibit.svg?raw"
	import cubeIcon from "@phosphor-icons/core/regular/cube.svg?raw"
	import downloadSimpleIcon from "@phosphor-icons/core/regular/download-simple.svg?raw"
	import fileArrowDownIcon from "@phosphor-icons/core/regular/file-arrow-down.svg?raw"
	import fingerprintIcon from "@phosphor-icons/core/regular/fingerprint.svg?raw"
	import gameControllerIcon from "@phosphor-icons/core/regular/game-controller.svg?raw"
	import heartFillIcon from "@phosphor-icons/core/fill/heart-fill.svg?raw"
	import heartIcon from "@phosphor-icons/core/regular/heart.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import mapPinIcon from "@phosphor-icons/core/regular/map-pin.svg?raw"
	import packageIcon from "@phosphor-icons/core/regular/package.svg?raw"
	import shoppingCartIcon from "@phosphor-icons/core/regular/shopping-cart.svg?raw"
	import shapesIcon from "@phosphor-icons/core/regular/shapes.svg?raw"
	import storefrontIcon from "@phosphor-icons/core/regular/storefront.svg?raw"
	import tShirtIcon from "@phosphor-icons/core/regular/t-shirt.svg?raw"
	import timerIcon from "@phosphor-icons/core/regular/timer.svg?raw"
	import trashIcon from "@phosphor-icons/core/regular/trash.svg?raw"
	import treeStructureIcon from "@phosphor-icons/core/regular/tree-structure.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { onDestroy, tick, untrack } from "svelte"
	import {
		CatalogItemType,
		InventoryCategory,
		InventorySaleLocation,
		ItemAvailability,
	} from "../backend/bridge"
	import type { InventoryItem, CategorizedInventoryItem } from "../backend/bridge"
	import CreatorByline from "../shared/CreatorByline.svelte"
	import ProfileStats from "../shared/ProfileStats.svelte"
	import type { ProfileStat } from "../shared/ProfileStats.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import FrameUnavailable from "../shared/FrameUnavailable.svelte"
	import AnchoredActionMenu from "../shared/AnchoredActionMenu.svelte"
	import type { MenuAction } from "../shared/AnchoredActionMenu.svelte"
	import ExpandedImageDialog from "../shared/ExpandedImageDialog.svelte"
	import ItemExplorerDialog from "../explorer/ItemExplorerDialog.svelte"
	import Icon from "../shared/Icon.svelte"
	import ImageViewToggle from "../shared/ImageViewToggle.svelte"
	import type { ImageView } from "../shared/ImageViewToggle.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import AnimationControls from "../model-viewer/AnimationControls.svelte"
	import ModelFrame from "../model-viewer/ModelFrame.svelte"
	import type {
		AnimationPlayback,
		AnimationStart,
		ModelSubject,
	} from "../model-viewer/model-subject"
	import Skeleton from "../shared/Skeleton.svelte"
	import SkeletonTile from "../shared/SkeletonTile.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
	import { relativeTime } from "../shared/timestamp"
	import {
		assetCategories,
		inventoryCategoryOptionOf,
		inventoryTypeLabel,
	} from "./inventory-categories"
	import type { InventoryCategoryOption } from "./inventory-categories"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import InventoryItemTile from "./InventoryItemTile.svelte"
	import { formatRobux, salePriceLabel } from "./inventory-prices"
	import PurchaseAccountActions from "./PurchaseAccountActions.svelte"
	import PurchaseItemSummary from "./PurchaseItemSummary.svelte"
	import SavedAccountsOwnership from "./SavedAccountsOwnership.svelte"
	import { categoryItemKey, inventoryItemKey } from "./inventory-state.svelte"
	import type {
		InventoryDownloadKind,
		InventoryState,
	} from "./inventory-state.svelte"

	let {
		item: openedItem,
		category: openedCategory,
		inventory,
		username,
		accounts,
		isOwnInventory: isOpenedFromOwnInventory,
		isWorn: isOpenedItemWorn,
		onClose,
	}: {
		item: InventoryItem
		category: InventoryCategoryOption
		inventory: InventoryState
		username: string
		accounts: AccountStore
		isOwnInventory: boolean
		isWorn: boolean
		onClose: () => void
	} = $props()
	const previews = getProfilePreviews()
	const elementIdPrefix = $props.id()

	// The related items that the user opened inside the dialog, the latest
	// last. Back returns to the one before.
	let openedRelatedItems = $state.raw<
		{ item: InventoryItem; category: InventoryCategoryOption }[]
	>([])

	const account = $derived(accounts.selectedAccount),
		otherAccounts = $derived(
			accounts.accounts.filter((savedAccount) => savedAccount.id !== account.id),
		),
		item = $derived(openedRelatedItems.at(-1)?.item ?? openedItem),
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
		isExplorerOpen = $state(false),
		isTryingOn = $state(false),
		// Whether an animation or an emote plays on the plain R15 rig instead
		// of the saved avatar. The choice stays for the next item.
		isRigShown = $state(false),
		downloadButton = $state<HTMLButtonElement>(),
		isDownloadMenuOpen = $state(false),
		// The item whose model the 3D viewer could not build, which shows the
		// model that Roblox renders instead.
		failedLiveItemKey = $state(""),
		playbackByView = $state<
			Record<"dialog" | "expanded", AnimationPlayback | undefined>
		>({ dialog: undefined, expanded: undefined })

	const exactNumbers = new Intl.NumberFormat("en-US"),
		compactNumbers = new Intl.NumberFormat("en-US", {
			notation: "compact",
			maximumFractionDigits: 1,
		}),
		percent = new Intl.NumberFormat("en-US", {
			style: "percent",
			maximumFractionDigits: 1,
		}),
		saleLocationLabels: Record<InventorySaleLocation, string> = {
			[InventorySaleLocation.$zero]: "",
			[InventorySaleLocation.SaleInMarketplace]: "Marketplace only",
			[InventorySaleLocation.SaleInMarketplaceAndAllExperiences]:
				"Marketplace and games",
			[InventorySaleLocation.SaleInMarketplaceAndSomeExperiences]:
				"Marketplace and some games",
			[InventorySaleLocation.SaleInExperiences]: "Games only",
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
		// Whether the item is an asset, whose ID the worn items can hold.
		isAssetCategory = $derived(assetCategories.includes(category.id)),
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
		hasModelView = $derived(!!details?.hasModelView),
		hasRenderedModel = $derived(!!details?.hasRenderedModel),
		canTryOn = $derived(!!details?.canTryOn),
		canHaveRelatedItems = $derived(!!details?.canHaveRelatedItems),
		canDownload = $derived(!!details?.canDownload),
		canExplore = $derived(!!details?.canExplore),
		// Whether Roblox moderated or deleted the item.
		isRemoved = $derived(
			details?.availability === ItemAvailability.ItemModerated ||
				details?.availability === ItemAvailability.ItemDeleted,
		),
		// The file of the item, and the 3D model that Roblox renders of it with
		// its textures, which 3D programs open.
		downloadActions = $derived<MenuAction[]>([
			...(canDownload
				? [
						{
							label: "Download file",
							icon: fileArrowDownIcon,
							onSelect: () => download("file"),
						},
					]
				: []),
			...(hasRenderedModel
				? [
						{
							label: "Download 3D model",
							icon: cubeIcon,
							onSelect: () => download("model"),
						},
					]
				: []),
		]),
		// undefined while the related items load, and null when they failed.
		relatedItems = $derived(inventory.relatedItemsByItemKey.get(detailsKey)),
		// Try on stays on while the view switches between 2D and 3D.
		isTryOnShown = $derived(canTryOn && isTryingOn),
		// An item without a model of its own has a 3D view only on the avatar.
		hasThreeDimensionalView = $derived(hasModelView || isTryOnShown),
		shownView = $derived<ImageView>(hasThreeDimensionalView ? view : "2D"),
		isModelShown = $derived(shownView === "3D"),
		tryOnLabel = $derived(`@${account.username}'s avatar wearing ${name}`),
		tryOnImageUrl = $derived(inventory.tryOnImage.loadedFor(detailsKey) ?? ""),
		tryOnError = $derived(inventory.tryOnImage.errorFor(detailsKey)),
		tryOnAvatar = $derived(inventory.tryOnAvatar.loadedFor(detailsKey)),
		tryOnAvatarError = $derived(inventory.tryOnAvatar.errorFor(detailsKey)),
		tryOnSubject = $derived<ModelSubject | null>(
			tryOnAvatar ? { kind: "avatar", look: tryOnAvatar } : null,
		),
		itemLook = $derived(inventory.itemLook.loadedFor(detailsKey)),
		itemLookError = $derived(inventory.itemLook.errorFor(detailsKey)),
		// An item shows the model that Roblox renders when the 3D viewer does
		// not build its kind, such as models and gear, or could not build it.
		isItemModelRendered = $derived(
			hasRenderedModel &&
				(!!itemLookError ||
					failedLiveItemKey === detailsKey ||
					(!!itemLook && !itemLook.avatar && !itemLook.accessoryAssetId)),
		),
		itemModel = $derived(inventory.itemModel.loadedFor(detailsKey)),
		itemModelError = $derived(inventory.itemModel.errorFor(detailsKey)),
		// An animation or an emote plays on the avatar that shows it.
		itemStartAnimation = $derived<AnimationStart | undefined>(
			category.id === InventoryCategory.InventoryEmotes
				? { kind: "emote", emote: item }
				: category.id === InventoryCategory.InventoryAnimations
					? { kind: "animation", assetTypeName: item.typeName }
					: undefined,
		),
		itemSubject = $derived<ModelSubject | null>(
			isItemModelRendered
				? {
						kind: "rendered",
						model: itemModel,
						error: itemModelError,
						onRetry: () => void inventory.loadItemModel(item, category.id),
					}
				: itemLook?.avatar
					? {
							kind: "avatar",
							look:
								isRigShown && itemLook.rigAvatar
									? itemLook.rigAvatar
									: itemLook.avatar,
							startAnimation: itemStartAnimation,
						}
					: itemLook
						? { kind: "accessory", assetId: itemLook.accessoryAssetId }
						: null,
		),
		// Whether the 3D view shows an avatar, which has animations to play.
		isAvatarModelShown = $derived(
			isModelShown &&
				(isTryOnShown ? !tryOnAvatarError : itemSubject?.kind === "avatar"),
		),
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
			item.serialNumber !== null ||
				!!stockLabel ||
				(isCatalogItem && !!market) ||
				!!saleLocationLabel ||
				(!!market && market.offSaleDeadlineMs > 0),
		),
		// The headline numbers come first, and the sections below never
		// repeat them.
		itemStats = $derived<ProfileStat[]>([
			...(market?.favoriteCount == null
				? []
				: [
						{
							id: "favorites",
							value: compactNumbers.format(market.favoriteCount),
							label:
								market.favoriteCount === 1 ? "favorite" : "favorites",
							description: `${exactNumbers.format(market.favoriteCount)} ${market.favoriteCount === 1 ? "favorite" : "favorites"}`,
						},
					]),
			...(item.recentAveragePrice === null
				? []
				: [
						{
							id: "recentAveragePrice",
							value: formatRobux(item.recentAveragePrice, true),
							label: "average price",
							description: `${formatRobux(item.recentAveragePrice, false)} recent average price`,
						},
					]),
			...(market?.lowestResalePriceInRobux == null
				? []
				: [
						{
							id: "lowestResale",
							value: formatRobux(market.lowestResalePriceInRobux, true),
							label: "lowest resale",
							description: `${formatRobux(market.lowestResalePriceInRobux, false)} lowest resale price`,
						},
					]),
			...(badge
				? [
						{
							id: "wonYesterday",
							value: percent.format(badge.pastDayWinFraction),
							label: "won yesterday",
							description: `Won by ${percent.format(badge.pastDayWinFraction)} of players yesterday`,
						},
						{
							id: "awarded",
							value: compactNumbers.format(badge.awardedCount),
							label: "awarded",
							description: `Awarded ${exactNumbers.format(badge.awardedCount)} times`,
						},
					]
				: []),
		])

	$effect(() => {
		const shownItem = item,
			shownCategory = category.id
		if (canHaveRelatedItems) {
			untrack(() => void inventory.loadRelatedItems(shownItem, shownCategory))
		}
	})

	$effect(() => {
		const shownRelatedItems = relatedItems
		if (shownRelatedItems?.length) {
			untrack(() => void inventory.loadOwnership(shownRelatedItems))
		}
	})

	$effect(() => {
		const shownItem = item,
			shownCategory = category.id
		if (otherAccounts.length > 0) {
			untrack(
				() =>
					void inventory.loadSavedAccountsOwnership(shownItem, shownCategory),
			)
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
		isTryingOn = isExpanded = isExplorerOpen = false
		inventory.cancelFindingGamePassPlace()
		inventory.clearItemActionError()
		await tick()
		card?.querySelector<HTMLElement>(
			`#${elementIdPrefix}-inventory-item-title`,
		)?.scrollIntoView({
			block: "nearest",
		})
		card?.querySelector<HTMLElement>(
			nextOpenedRelatedItems.length > 0
				? "[data-item-back]"
				: "[data-close-item-details]",
		)?.focus()
	}

	function openRelatedItem(relatedItem: CategorizedInventoryItem): void {
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
				void inventory.loadTryOnAvatar(shownItem, shownCategory)
			if (isTryingOnInImage) void inventory.tryOnItem(shownItem, shownCategory)
			if (isItemModelShown) void inventory.loadItemLook(shownItem, shownCategory)
		})
	})

	$effect(() => {
		if (isModelShown && !isTryOnShown && isItemModelRendered) {
			const shownItem = item,
				shownCategory = category.id
			untrack(() => void inventory.loadItemModel(shownItem, shownCategory))
		}
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

	// A lookup that ends after the dialog closes must not navigate.
	onDestroy(() => inventory.cancelFindingGamePassPlace())

	async function showGamePass(): Promise<void> {
		const gamePassId = item.itemId,
			gamePassKey = detailsKey,
			place = await inventory.findGamePassPlace(gamePassId)
		if (place && detailsKey === gamePassKey)
			previews.show({ kind: "Place", placeId: place.placeId, gamePassId })
	}

	function download(downloadKind: InventoryDownloadKind): void {
		isDownloadMenuOpen = false
		void inventory.downloadItem(item, category.id, downloadKind)
	}

	function close(): void {
		if (inventory.isDeletingItem || inventory.isBuyingItem) return
		if (confirmation) void showConfirmation(null)
		else onClose()
	}

	notifyErrors("Item action not completed", () => inventory.itemActionError)

	notifyErrors("Avatar not loaded", () => tryOnAvatarError)
	notifyErrors("Item not tried on", () => tryOnError)
	notifyErrors("Preview not loaded", () => itemLookError)
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") {
			event.preventDefault()
			close()
		}
	}} />

{#snippet imageFrame(isExpandedView: boolean)}
	{#if isTryOnShown && isModelShown}
		{#if tryOnAvatarError}
			<FrameUnavailable
				icon={category.icon}
				message="Avatar unavailable"
				onRetry={() => void inventory.loadTryOnAvatar(item, category.id)} />
		{:else}
			<ModelFrame
				accountId={inventory.accountId}
				subject={tryOnSubject}
				label={tryOnLabel}
				bind:playback={
					playbackByView[isExpandedView ? "expanded" : "dialog"]
				} />
		{/if}
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
				message="Preview unavailable"
				onRetry={() => void inventory.tryOnItem(item, category.id)} />
		{:else}
			<span class="absolute inset-3 skeleton rounded-md" role="status">
				<span class="sr-only">Trying on the item</span>
			</span>
		{/if}
	{:else if isModelShown}
		{#if itemLookError && !hasRenderedModel}
			<FrameUnavailable
				icon={category.icon}
				message="Preview unavailable"
				onRetry={() => void inventory.loadItemLook(item, category.id)} />
		{:else}
			<ModelFrame
				accountId={inventory.accountId}
				subject={itemSubject}
				label={name}
				bind:playback={playbackByView[isExpandedView ? "expanded" : "dialog"]}
				onLiveFailure={hasRenderedModel
					? () => (failedLiveItemKey = detailsKey)
					: undefined} />
		{/if}
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
	{#if isModelShown && !isItemModelRendered && itemLook?.rigAvatar}
		<div class="segmented shrink-0" role="group" aria-label="Animation shown on">
			<button
				type="button"
				aria-pressed={!isRigShown}
				data-tooltip={`Play on @${account.username}'s saved avatar`}
				onclick={() => (isRigShown = false)}>Avatar</button>
			<button
				type="button"
				aria-pressed={isRigShown}
				data-tooltip="Play on a plain R15 rig. Roblox plays animations and emotes only on R15."
				onclick={() => (isRigShown = true)}>R15 rig</button>
		</div>
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
	{#if isExpandedView && isAvatarModelShown}
		<div class="min-w-0 flex-1">
			<AnimationControls
				playback={playbackByView.expanded}
				isWide
				hasAnimationPicker={!itemStartAnimation} />
		</div>
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
	class={confirmation ? "max-w-md text-left" : "max-w-4xl text-left"}
	role={confirmation ? "alertdialog" : "dialog"}
	aria-labelledby={confirmation
		? `${elementIdPrefix}-inventory-confirmation-title`
		: `${elementIdPrefix}-inventory-item-title`}
	aria-describedby={confirmation
		? `${elementIdPrefix}-inventory-confirmation-description`
		: undefined}
	bind:card
	onBackdropClick={close}>
	{#if confirmation}
		{@const isWorking = inventory.isDeletingItem || inventory.isBuyingItem}
		{@const price = market?.priceInRobux ?? 0}
		<div class="dialog-heading">
			<div>
				{#if confirmation === "deletion"}
					<h2 id={`${elementIdPrefix}-inventory-confirmation-title`}>
						Delete item?
					</h2>
					<p id={`${elementIdPrefix}-inventory-confirmation-description`}>
						{name} will be removed from @{username}'s inventory. Robux spent
						on it is not refunded, and this cannot be undone.
					</p>
				{:else if price === 0}
					<h2 id={`${elementIdPrefix}-inventory-confirmation-title`}>
						Get item?
					</h2>
					<p id={`${elementIdPrefix}-inventory-confirmation-description`}>
						This item is free.
					</p>
				{:else}
					<h2 id={`${elementIdPrefix}-inventory-confirmation-title`}>
						Buy item?
					</h2>
					<p id={`${elementIdPrefix}-inventory-confirmation-description`}>
						Roblox does not refund items.
					</p>
				{/if}
			</div>
			<button
				type="button"
				aria-label="Close dialog"
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
				<div class="flex w-96 shrink-0 flex-col gap-2">
					<div
						class={[
							"relative grid size-96 place-items-center overflow-hidden rounded-lg border border-line-subtle",
							isModelShown ? "bg-canvas" : "bg-raised",
						]}
						aria-hidden={isModelShown || isTryOnShown ? undefined : true}>
						{@render imageFrame(false)}
					</div>
					<div class="flex items-center gap-2">
						{@render imageControls(false)}
					</div>
					{#if isAvatarModelShown}
						<div class="animate-enter">
							<AnimationControls
								playback={playbackByView.dialog}
								hasAnimationPicker={!itemStartAnimation} />
						</div>
					{/if}
				</div>

				<div class="flex min-w-0 flex-1 flex-col gap-4">
					<div class="flex flex-col gap-1">
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
								id={`${elementIdPrefix}-inventory-item-title`}
								class="min-w-0 flex-1 leading-7 tracking-tight wrap-anywhere">
								{name}
							</h2>
							<div class="flex shrink-0 items-center gap-1">
								{#if canBuy && market}
									<button
										class="button-primary"
										type="button"
										data-begin-purchase
										disabled={isBusy ||
											isOwnedByAccount === undefined}
										onclick={() =>
											void showConfirmation("purchase")}>
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
											void inventory.setFavorite(
												item,
												!isFavorite,
											)}>
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
										aria-label="Show game"
										data-tooltip="Show game"
										onclick={() =>
											previews.show({
												kind: "Place",
												placeId: badge.awardingPlaceId,
											})}>
										<Icon
											svg={magnifyingGlassIcon}
											size={16}
											aria-hidden="true" />
									</button>
								{/if}
								{#if category.id === InventoryCategory.InventoryGamePasses}
									<button
										class="button-icon"
										type="button"
										aria-label="Show game"
										aria-busy={inventory.isFindingGamePassPlace}
										data-tooltip="Show game"
										disabled={inventory.isFindingGamePassPlace}
										onclick={() => void showGamePass()}>
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
								{#if canExplore}
									<button
										class="button-icon"
										type="button"
										aria-label="Open explorer"
										aria-haspopup="dialog"
										data-tooltip="Explore the instances in the file"
										data-open={isExplorerOpen || undefined}
										onclick={() => (isExplorerOpen = true)}>
										<Icon
											svg={treeStructureIcon}
											size={16}
											aria-hidden="true" />
									</button>
								{/if}
								{#if downloadActions.length > 0}
									{@const onlyDownloadAction =
										downloadActions.length === 1
											? downloadActions[0]
											: undefined}
									<button
										bind:this={downloadButton}
										class="button-icon"
										type="button"
										aria-label={onlyDownloadAction?.label ??
											"Download"}
										aria-haspopup={onlyDownloadAction
											? undefined
											: "menu"}
										aria-expanded={onlyDownloadAction
											? undefined
											: isDownloadMenuOpen}
										aria-controls={isDownloadMenuOpen
											? `${elementIdPrefix}-inventory-download-menu`
											: undefined}
										aria-busy={inventory.isDownloadingItem}
										data-open={isDownloadMenuOpen || undefined}
										data-tooltip={onlyDownloadAction?.label ??
											"Download"}
										disabled={inventory.isDownloadingItem}
										onclick={() => {
											if (onlyDownloadAction)
												onlyDownloadAction.onSelect()
											else
												isDownloadMenuOpen = !isDownloadMenuOpen
										}}>
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
										onclick={() =>
											void showConfirmation("deletion")}>
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
									aria-label="Close dialog"
									onclick={close}>
									<Icon svg={xIcon} size={14} aria-hidden="true" />
								</button>
							</div>
						</div>
						<div class={metaClass}>
							<span class="select-text">ID: {item.itemId}</span>
							<span class="text-fg-faint" aria-hidden="true">·</span>
							<span>{typeLabel}</span>
						</div>
						{#if item.creatorId && (item.creatorType === "User" || item.creatorType === "Group")}
							<CreatorByline
								kind={item.creatorType}
								creatorId={item.creatorId}
								name={item.creatorName}
								hasVerifiedBadge={item.creatorHasVerifiedBadge} />
						{:else if item.creatorName}
							<div class={metaClass}>
								<span>by {item.creatorName}</span>
							</div>
						{/if}
						{#if market}
							<p
								class={[
									"inline-flex items-center gap-1 text-xs font-medium tabular-nums",
									market.isForSale ? "text-fg" : "text-fg-muted",
								]}>
								<Icon
									svg={storefrontIcon}
									size={14}
									aria-hidden="true" />
								{priceLabel}
							</p>
						{/if}
						<ProfileStats label="Item numbers" stats={itemStats} />

						<ul
							class="mt-2 flex min-h-6 flex-wrap gap-1"
							aria-label="Status">
							{#if isRemoved}
								<li class="tag-pill-warning animate-pop">
									<Icon
										svg={prohibitIcon}
										size={12}
										aria-hidden="true" />
									<span>Deleted</span>
								</li>
							{/if}
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
									<Icon
										svg={tShirtIcon}
										size={12}
										aria-hidden="true" />
									<span>Worn</span>
								</li>
							{/if}
							{#if market?.isLimited}
								<li class="tag-pill-success">
									<Icon
										svg={diamondIcon}
										size={12}
										aria-hidden="true" />
									<span>Limited</span>
								</li>
							{/if}
						</ul>

						{#if details?.availability === ItemAvailability.ItemModerated}
							<p class="text-xs text-fg-muted">
								Roblox moderated this item and deleted its content, so
								its image, file, and 3D model are gone.
							</p>
						{:else if details?.availability === ItemAvailability.ItemDeleted}
							<p class="text-xs text-fg-muted">
								Roblox deleted this item. Only the name and type from
								the list remain.
							</p>
						{/if}

						{#if isPurchasePending}
							<p class="text-xs text-fg-muted" role="status">
								Bought. It can take a minute to show up in the
								inventory.
							</p>
						{/if}
					</div>

					{#if item.description}
						<section
							class="flex flex-col gap-3"
							aria-labelledby={`${elementIdPrefix}-inventory-item-description`}>
							<h3
								id={`${elementIdPrefix}-inventory-item-description`}
								class={sectionHeadingClass}>
								Description
							</h3>
							<p
								class="max-h-40 text-block overflow-y-auto"
								data-variant="prose">
								{item.description}
							</p>
						</section>
					{/if}
					{#if hasMarketSection}
						<section
							class="flex flex-col gap-3"
							aria-labelledby={`${elementIdPrefix}-inventory-item-market`}>
							<h3
								id={`${elementIdPrefix}-inventory-item-market`}
								class={sectionHeadingClass}>
								Market
							</h3>
							<dl class="facts">
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
									{@render fact(
										mapPinIcon,
										"Sold in",
										saleLocationLabel,
									)}
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
											<Timestamp
												value={market.offSaleDeadlineMs} />
											<small
												>{relativeTime(
													market.offSaleDeadlineMs,
												)}</small>
										</dd>
									</div>
								{/if}
							</dl>
						</section>
					{/if}

					<section
						class="flex flex-col gap-3"
						aria-labelledby={`${elementIdPrefix}-inventory-item-details`}>
						<h3
							id={`${elementIdPrefix}-inventory-item-details`}
							class={sectionHeadingClass}>
							Details
						</h3>
						<dl class="facts">
							{@render fact(shapesIcon, "Type", typeLabel)}
							{#if badge}
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
				</div>
			</div>

			{#if otherAccounts.length > 0}
				<SavedAccountsOwnership
					{otherAccounts}
					ownership={inventory.savedAccountsOwnershipByItemKey.get(
						detailsKey,
					)}
					canBeWorn={isAssetCategory} />
			{/if}

			{#if canHaveRelatedItems}
				<section
					class="flex flex-col gap-3"
					aria-labelledby={`${elementIdPrefix}-inventory-item-related`}>
					<h3
						id={`${elementIdPrefix}-inventory-item-related`}
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
							>Loading {isBundle
								? "bundle items"
								: "dependencies"}…</span>
					{:else if relatedItems === null}
						<div class="flex items-center gap-2 text-xs text-fg-muted">
							<span class="min-w-0 flex-1"
								>{isBundle ? "Bundle items" : "Dependencies"} unavailable</span>
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
										isOwned={inventory.isOwnedByItemKey.get(
											categoryItemKey(
												relatedItem.category,
												relatedItem.item.itemId,
											),
										) === true}
										actionLabel="Open details"
										onOpen={() => openRelatedItem(relatedItem)} />
								</li>
							{/each}
						</ul>
					{/if}
				</section>
			{/if}
		</div>
	{/if}
</Dialog>

{#if isDownloadMenuOpen && downloadButton && downloadActions.length > 1}
	<AnchoredActionMenu
		id={`${elementIdPrefix}-inventory-download-menu`}
		label="Download actions"
		anchor={downloadButton}
		actions={downloadActions}
		onClose={() => (isDownloadMenuOpen = false)} />
{/if}

{#if isExplorerOpen && !confirmation}
	<ItemExplorerDialog
		{item}
		{category}
		{inventory}
		onOpenAssetItem={openRelatedItem}
		onClose={() => (isExplorerOpen = false)} />
{/if}

{#if isExpanded && !confirmation}
	<ExpandedImageDialog label={name} onClose={() => (isExpanded = false)}>
		{#snippet frame()}
			{@render imageFrame(true)}
		{/snippet}
		{#snippet controls()}
			{@render imageControls(true)}
		{/snippet}
	</ExpandedImageDialog>
{/if}
