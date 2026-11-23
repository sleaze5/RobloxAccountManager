import { accountBackend, InventoryCategory } from "../backend/bridge"
import type {
	Avatar,
	CategorizedInventoryItem,
	SavedAccountsItemOwnership,
	GamePlace,
	InventoryItem,
	InventoryItemDetails,
	InventoryItemLook,
	ItemInstanceTree,
	RenderedModel,
} from "../backend/bridge"
import type { AccountStore } from "../accounts/account-store.svelte"
import { AccountScope } from "../accounts/account-scope"
import { readRobuxBalance } from "../accounts/robux-balance"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import type { CancellableRequest } from "../shared/cancellable-request"
import { copyFailureMessage, copyText } from "../shared/clipboard"
import { errorMessage } from "../shared/errors"
import { isFresh } from "../shared/freshness"
import { KeyedLoad } from "../shared/keyed-load.svelte"
import { assetCategories } from "./inventory-categories"

interface InventoryCategoryPage {
	items: InventoryItem[]
	nextCursor: string
	fetchedAt: number
}

// inventoryItemKey tells apart an asset and a bundle that share an ID.
export function inventoryItemKey(item: InventoryItem): string {
	return `${item.catalogItemType}:${item.itemId}`
}

// ownedCopyKey tells apart the copies of a collectible by their serial numbers.
// Copies without a serial number show the same details.
export function ownedCopyKey(item: InventoryItem): string {
	return `${inventoryItemKey(item)}:${item.serialNumber ?? ""}`
}

// categoryItemKey tells apart the items of different categories that share an
// ID, such as a badge and a game pass.
export function categoryItemKey(category: InventoryCategory, itemId: number): string {
	return `${category}:${itemId}`
}

export type InventoryDownloadKind = "file" | "model"

// ownershipCheckBatchSize is the most items that one ownership check takes.
const ownershipCheckBatchSize = 100

// InventoryState lists what one Roblox user owns, as the account sees it. Each
// category keeps its pages while the inventory stays open, so that switching
// back to a category shows it at once.
export class InventoryState {
	category = $state<InventoryCategory>(InventoryCategory.InventoryAccessories)
	pagesByCategory = $state.raw<
		Partial<Record<InventoryCategory, InventoryCategoryPage>>
	>({})
	loading = $state(false)
	loadingMore = $state(false)
	error = $state("")
	// Whether the account favorited each catalog item, by inventoryItemKey.
	favoriteByItemKey = $state.raw(new Map<string, boolean>())
	checkingFavoriteKey = $state("")
	isSavingFavorite = $state(false)
	isDeletingItem = $state(false)
	isBuyingItem = $state(false)
	isDownloadingItem = $state(false)
	isFindingGamePassPlace = $state(false)
	#gamePassPlaceRequest: ReturnType<typeof accountBackend.FindGamePassPlace> | null =
		null
	// The items that the account bought and that Roblox still delivers, by
	// categoryItemKey.
	pendingPurchaseKeys = $state.raw(new Set<string>())
	// The Robux of the account, read when a purchase confirmation opens.
	// undefined while it loads, and null when it failed.
	robuxBalance = $state<number | null | undefined>(undefined)
	itemActionError = $state("")
	wornAssetIds = $state.raw(new Set<number>())
	// The details and the ownership by the account of each opened item, by
	// categoryItemKey. null means that the request failed.
	detailsByItemKey = $state.raw(new Map<string, InventoryItemDetails | null>())
	isOwnedByItemKey = $state.raw(new Map<string, boolean | null>())
	// Which other saved accounts own each opened item, by categoryItemKey.
	// null means that the request failed.
	savedAccountsOwnershipByItemKey = $state.raw(
		new Map<string, SavedAccountsItemOwnership | null>(),
	)
	// The assets that each opened item includes or uses, by categoryItemKey.
	// null means that the request failed.
	relatedItemsByItemKey = $state.raw(
		new Map<string, CategorizedInventoryItem[] | null>(),
	)
	// How the 3D viewer shows the opened item, by the categoryItemKey of the
	// item.
	readonly itemLook: KeyedLoad<InventoryItemLook>
	// The 3D model that Roblox renders of the opened item, for an item that
	// the 3D viewer cannot build, by the categoryItemKey of the item.
	readonly itemModel: KeyedLoad<RenderedModel>
	// The full-body image of the account wearing the opened item, which Roblox
	// renders without saving it, by the categoryItemKey of the item.
	readonly tryOnImage: KeyedLoad<string>
	// The saved avatar of the account wearing the opened item, which the 3D
	// viewer builds, by the categoryItemKey of the item.
	readonly tryOnAvatar: KeyedLoad<Avatar>
	// The instances of the file of the opened item, or of each asset of a
	// bundle, which the explorer shows, by the categoryItemKey of the item.
	readonly instanceTree: KeyedLoad<ItemInstanceTree>
	// Why the explorer could not copy a value, shown in the explorer.
	explorerCopyError = $state("")
	explorerAssetError = $state("")
	readonly userId: number
	readonly #scope: AccountScope
	#request: ReturnType<typeof accountBackend.GetInventoryPage> | null = null
	readonly #loadingRelatedItemKeys = new Set<string>()
	readonly #checkingOwnershipKeys = new Set<string>()
	readonly #loadingSavedAccountsOwnershipKeys = new Set<string>()
	readonly #notifications: NotificationCenter

	constructor(
		accountId: number,
		userId: number,
		store: AccountStore,
		notifications: NotificationCenter,
	) {
		this.#scope = new AccountScope(accountId, store)
		this.#notifications = notifications
		this.itemLook = new KeyedLoad(this.#scope)
		this.itemModel = new KeyedLoad(this.#scope)
		this.tryOnImage = new KeyedLoad(this.#scope)
		this.tryOnAvatar = new KeyedLoad(this.#scope)
		this.instanceTree = new KeyedLoad(this.#scope)
		this.userId = userId
	}

	get accountId(): number {
		return this.#scope.accountId
	}

	get page(): InventoryCategoryPage | null {
		return this.pagesByCategory[this.category] ?? null
	}

	show(category: InventoryCategory): void {
		if (category === this.category) return
		void this.#request?.cancel()
		this.#request = null
		this.loading = this.loadingMore = false
		this.error = ""
		this.category = category
		const page = this.page
		if (!page || !isFresh(page.fetchedAt)) void this.refresh()
	}

	async refresh(): Promise<void> {
		void this.loadWornAssets()
		await this.#load("")
	}

	async loadMore(): Promise<void> {
		const cursor = this.page?.nextCursor
		if (!cursor || this.loading || this.loadingMore) return
		await this.#load(cursor)
	}

	clearItemActionError(): void {
		this.itemActionError = ""
	}

	// loadItemDetails reads the details of an item, and whether the account
	// owns it when the item belongs to another user.
	loadItemDetails(
		item: InventoryItem,
		category: InventoryCategory,
		shouldCheckOwnership: boolean,
	): void {
		const key = categoryItemKey(category, item.itemId)
		if (!this.detailsByItemKey.has(key)) {
			void this.#loadItemFact(
				accountBackend.GetInventoryItemDetails(
					this.#scope.accountId,
					category,
					item.itemId,
				),
				(details) =>
					(this.detailsByItemKey = new Map(this.detailsByItemKey).set(
						key,
						details,
					)),
			)
		}
		if (shouldCheckOwnership && !this.isOwnedByItemKey.has(key)) {
			void this.#loadItemFact(
				accountBackend.IsInventoryItemOwned(
					this.#scope.accountId,
					category,
					item.itemId,
				),
				(isOwned) =>
					(this.isOwnedByItemKey = new Map(this.isOwnedByItemKey).set(
						key,
						isOwned,
					)),
			)
		}
	}

	// loadOwnership checks which of the items the account owns, so that their
	// tiles show it. An item whose check fails stays unknown, and its details
	// check it again.
	async loadOwnership(categorizedItems: CategorizedInventoryItem[]): Promise<void> {
		const references = new Map<
			string,
			{ category: InventoryCategory; itemId: number }
		>()
		for (const { category, item } of categorizedItems) {
			const key = categoryItemKey(category, item.itemId)
			if (
				!this.isOwnedByItemKey.has(key) &&
				!this.#checkingOwnershipKeys.has(key)
			) {
				references.set(key, { category, itemId: item.itemId })
			}
		}
		const keys = [...references.keys()]
		for (const key of keys) this.#checkingOwnershipKeys.add(key)
		try {
			const batches = Array.from(
					{ length: Math.ceil(keys.length / ownershipCheckBatchSize) },
					(_unused, index) =>
						keys.slice(
							index * ownershipCheckBatchSize,
							(index + 1) * ownershipCheckBatchSize,
						),
				),
				ownershipsByBatch = await Promise.all(
					batches.map((batchKeys) =>
						this.#scope.track(
							accountBackend.CheckInventoryItemsOwned(
								this.#scope.accountId,
								batchKeys.map((key) => references.get(key)!),
							),
						),
					),
				)
			if (!this.#scope.isCurrent) return
			const isOwnedByItemKey = new Map(this.isOwnedByItemKey)
			for (const ownership of ownershipsByBatch.flatMap(
				(ownerships) => ownerships ?? [],
			)) {
				isOwnedByItemKey.set(
					categoryItemKey(ownership.category, ownership.itemId),
					ownership.isOwned,
				)
			}
			this.isOwnedByItemKey = isOwnedByItemKey
		} catch {
			// The tiles stay without the owned mark, and the backend logs the
			// failure.
		} finally {
			for (const key of keys) this.#checkingOwnershipKeys.delete(key)
		}
	}

	// loadSavedAccountsOwnership reads which other saved accounts own the item,
	// unless every account answered before. A failed read loads again.
	async loadSavedAccountsOwnership(
		item: InventoryItem,
		category: InventoryCategory,
	): Promise<void> {
		const key = categoryItemKey(category, item.itemId)
		if (
			this.savedAccountsOwnershipByItemKey.get(key)?.failedAccountCount === 0 ||
			this.#loadingSavedAccountsOwnershipKeys.has(key)
		) {
			return
		}
		this.#loadingSavedAccountsOwnershipKeys.add(key)
		if (this.savedAccountsOwnershipByItemKey.has(key)) {
			const savedAccountsOwnershipByItemKey = new Map(
				this.savedAccountsOwnershipByItemKey,
			)
			savedAccountsOwnershipByItemKey.delete(key)
			this.savedAccountsOwnershipByItemKey = savedAccountsOwnershipByItemKey
		}
		try {
			await this.#loadItemFact(
				accountBackend.GetSavedAccountsItemOwnership(
					this.#scope.accountId,
					category,
					item.itemId,
				),
				(ownership) =>
					(this.savedAccountsOwnershipByItemKey = new Map(
						this.savedAccountsOwnershipByItemKey,
					).set(key, ownership)),
			)
		} finally {
			this.#loadingSavedAccountsOwnershipKeys.delete(key)
		}
	}

	async loadItemLook(
		item: InventoryItem,
		category: InventoryCategory,
	): Promise<void> {
		await this.itemLook.load(
			categoryItemKey(category, item.itemId),
			() =>
				accountBackend.GetInventoryItemLook(
					this.#scope.accountId,
					category,
					item.itemId,
				),
			"Could not load the 3D model. Try again.",
		)
	}

	async loadInstanceTree(
		item: InventoryItem,
		category: InventoryCategory,
	): Promise<void> {
		await this.instanceTree.load(
			categoryItemKey(category, item.itemId),
			() =>
				accountBackend.GetItemInstanceTree(
					this.#scope.accountId,
					category,
					item.itemId,
				),
			"Could not load the instances. Try again.",
		)
	}

	// copyExplorerText copies a value that the explorer shows, such as the
	// source of a script. label names it in sentence case, such as "Source".
	async copyExplorerText(text: string, label: string): Promise<boolean> {
		this.explorerCopyError = ""
		const isCopied = await copyText(
			this.#notifications,
			text,
			`${label} copied to clipboard.`,
		)
		if (!isCopied) this.explorerCopyError = copyFailureMessage(label)
		return isCopied
	}

	// readAssetItem reads one asset that the explorer shows, such as a mesh, as
	// an item. It returns null when it fails or the account changed.
	async readAssetItem(assetId: number): Promise<CategorizedInventoryItem | null> {
		const scope = this.#scope
		this.explorerAssetError = ""
		try {
			const assetItem = await scope.track(
				accountBackend.GetInventoryAssetItem(scope.accountId, assetId),
			)
			return scope.isCurrent ? assetItem : null
		} catch (error) {
			if (scope.isCurrent) {
				this.explorerAssetError = errorMessage(
					error,
					"Could not open the asset. Try again.",
				)
			}
			return null
		}
	}

	async loadItemModel(
		item: InventoryItem,
		category: InventoryCategory,
	): Promise<void> {
		await this.itemModel.load(
			categoryItemKey(category, item.itemId),
			() =>
				accountBackend.GetInventoryItemModel(
					this.#scope.accountId,
					category,
					item.itemId,
				),
			"Could not load the 3D model. Try again.",
		)
	}

	async loadTryOnAvatar(
		item: InventoryItem,
		category: InventoryCategory,
	): Promise<void> {
		await this.tryOnAvatar.load(
			categoryItemKey(category, item.itemId),
			() =>
				accountBackend.GetInventoryItemTryOnAvatar(
					this.#scope.accountId,
					category,
					item.itemId,
				),
			"Could not try on the item in 3D. Try again.",
		)
	}

	// tryOnItem renders the avatar of the account wearing the item.
	async tryOnItem(item: InventoryItem, category: InventoryCategory): Promise<void> {
		await this.tryOnImage.load(
			categoryItemKey(category, item.itemId),
			() =>
				accountBackend.TryOnInventoryItem(
					this.#scope.accountId,
					category,
					item.itemId,
				),
			"Could not try on the item. Try again.",
		)
	}

	// loadRelatedItems reads the assets that the item includes or uses, unless
	// they are loaded or loading. A failed read loads again.
	async loadRelatedItems(
		item: InventoryItem,
		category: InventoryCategory,
	): Promise<void> {
		const key = categoryItemKey(category, item.itemId)
		if (
			this.relatedItemsByItemKey.get(key) ||
			this.#loadingRelatedItemKeys.has(key)
		) {
			return
		}
		this.#loadingRelatedItemKeys.add(key)
		if (this.relatedItemsByItemKey.has(key)) {
			const relatedItemsByItemKey = new Map(this.relatedItemsByItemKey)
			relatedItemsByItemKey.delete(key)
			this.relatedItemsByItemKey = relatedItemsByItemKey
		}
		try {
			await this.#loadItemFact(
				accountBackend.GetInventoryRelatedItems(
					this.#scope.accountId,
					category,
					item.itemId,
				),
				(relatedItems) =>
					(this.relatedItemsByItemKey = new Map(
						this.relatedItemsByItemKey,
					).set(key, relatedItems)),
			)
		} finally {
			this.#loadingRelatedItemKeys.delete(key)
		}
	}

	async downloadItem(
		item: InventoryItem,
		category: InventoryCategory,
		downloadKind: InventoryDownloadKind,
	): Promise<void> {
		if (this.isDownloadingItem) return
		this.isDownloadingItem = true
		this.itemActionError = ""
		try {
			const savedPath = await this.#scope.track(
				(downloadKind === "model"
					? accountBackend.DownloadInventoryItemModel
					: accountBackend.DownloadInventoryItem)(
					this.#scope.accountId,
					category,
					item.itemId,
					item.name,
				),
			)
			if (!this.#scope.isCurrent || !savedPath) return
			this.#notifications.show({
				id: "inventory-download",
				tone: "success",
				title: "Item downloaded",
				message: savedPath,
			})
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.itemActionError = errorMessage(
					error,
					downloadKind === "model"
						? "Could not download the 3D model. Try again."
						: "Could not download the item. Try again.",
				)
			}
		} finally {
			this.isDownloadingItem = false
		}
	}

	// findGamePassPlace returns the pass's game, or null when it fails or is
	// cancelled.
	async findGamePassPlace(gamePassId: number): Promise<GamePlace | null> {
		if (this.#gamePassPlaceRequest) return null
		const scope = this.#scope,
			request = accountBackend.FindGamePassPlace(gamePassId)
		this.#gamePassPlaceRequest = request
		this.isFindingGamePassPlace = true
		this.itemActionError = ""
		try {
			const place = await scope.track(request)
			return this.#gamePassPlaceRequest === request && scope.isCurrent
				? place
				: null
		} catch (error) {
			if (this.#gamePassPlaceRequest === request && scope.isCurrent) {
				this.itemActionError = errorMessage(
					error,
					"Could not find this pass's game. Try again.",
				)
			}
			return null
		} finally {
			if (this.#gamePassPlaceRequest === request) {
				this.#gamePassPlaceRequest = null
				this.isFindingGamePassPlace = false
			}
		}
	}

	cancelFindingGamePassPlace(): void {
		void this.#gamePassPlaceRequest?.cancel()
		this.#gamePassPlaceRequest = null
		this.isFindingGamePassPlace = false
	}

	async checkFavorite(item: InventoryItem): Promise<void> {
		const key = inventoryItemKey(item),
			itemType = item.catalogItemType
		if (
			!itemType ||
			this.favoriteByItemKey.has(key) ||
			this.checkingFavoriteKey === key
		) {
			return
		}
		this.checkingFavoriteKey = key
		try {
			const isFavorite = await this.#scope.track(
				accountBackend.IsInventoryItemFavorite(
					this.#scope.accountId,
					itemType,
					item.itemId,
				),
			)
			if (!this.#scope.isCurrent) return
			this.favoriteByItemKey = new Map(this.favoriteByItemKey).set(
				key,
				isFavorite,
			)
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.itemActionError = errorMessage(
					error,
					"Could not check whether this item is a favorite.",
				)
			}
		} finally {
			if (this.checkingFavoriteKey === key) this.checkingFavoriteKey = ""
		}
	}

	async setFavorite(item: InventoryItem, isFavorite: boolean): Promise<void> {
		const key = inventoryItemKey(item),
			itemType = item.catalogItemType
		if (!itemType || this.isSavingFavorite) return
		this.isSavingFavorite = true
		this.itemActionError = ""
		try {
			await this.#scope.track(
				accountBackend.SetInventoryItemFavorite(
					this.#scope.accountId,
					itemType,
					item.itemId,
					isFavorite,
				),
			)
			if (!this.#scope.isCurrent) return
			this.favoriteByItemKey = new Map(this.favoriteByItemKey).set(
				key,
				isFavorite,
			)
			const countChange = isFavorite ? 1 : -1
			this.#changeItems(Object.values(InventoryCategory), (items) =>
				items.map((listedItem) =>
					inventoryItemKey(listedItem) === key &&
					listedItem.market?.favoriteCount != null
						? {
								...listedItem,
								market: {
									...listedItem.market,
									favoriteCount: Math.max(
										listedItem.market.favoriteCount + countChange,
										0,
									),
								},
							}
						: listedItem,
				),
			)
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.itemActionError = errorMessage(
					error,
					isFavorite
						? "Could not add the item to favorites. Try again."
						: "Could not remove the item from favorites. Try again.",
				)
			}
		} finally {
			this.isSavingFavorite = false
		}
	}

	async loadRobuxBalance(): Promise<void> {
		this.robuxBalance = undefined
		const robuxBalance = await readRobuxBalance(this.#scope.accountId, (request) =>
			this.#scope.track(request),
		)
		if (this.#scope.isCurrent) this.robuxBalance = robuxBalance
	}

	// buyItem buys an item for the account at the price that the user
	// confirmed, and reports whether Roblox sold it.
	async buyItem(
		item: InventoryItem,
		category: InventoryCategory,
		priceInRobux: number,
	): Promise<boolean> {
		if (this.isBuyingItem) return false
		const key = categoryItemKey(category, item.itemId)
		this.isBuyingItem = true
		this.itemActionError = ""
		try {
			const isPending = await this.#scope.track(
				accountBackend.BuyInventoryItem(
					this.#scope.accountId,
					category,
					item.itemId,
					priceInRobux,
				),
			)
			if (!this.#scope.isCurrent) return false
			this.isOwnedByItemKey = new Map(this.isOwnedByItemKey).set(key, true)
			if (isPending) {
				this.pendingPurchaseKeys = new Set(this.pendingPurchaseKeys).add(key)
			}
			return true
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.itemActionError = errorMessage(
					error,
					"Could not buy the item. Try again.",
				)
			}
			return false
		} finally {
			this.isBuyingItem = false
		}
	}

	// deleteItem removes an asset from the inventory of the account, and
	// reports whether Roblox deleted it.
	async deleteItem(item: InventoryItem): Promise<boolean> {
		if (this.isDeletingItem) return false
		this.isDeletingItem = true
		this.itemActionError = ""
		try {
			await this.#scope.track(
				accountBackend.DeleteInventoryAsset(this.#scope.accountId, item.itemId),
			)
			if (!this.#scope.isCurrent) return false
			this.#changeItems(assetCategories, (items) =>
				items.filter((listedItem) => listedItem.itemId !== item.itemId),
			)
			return true
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.itemActionError = errorMessage(
					error,
					"Could not delete the item. Try again.",
				)
			}
			return false
		} finally {
			this.isDeletingItem = false
		}
	}

	dispose(): void {
		this.#request = null
		this.#scope.dispose()
	}

	async loadWornAssets(): Promise<void> {
		try {
			const assetIds = await this.#scope.track(
				accountBackend.GetWornAssetIDs(this.#scope.accountId, this.userId),
			)
			if (this.#scope.isCurrent) this.wornAssetIds = new Set(assetIds ?? [])
		} catch {
			// The inventory stays usable without the worn marks, and the backend
			// logs the failure.
		}
	}

	// #loadItemFact stores null when the request fails, so that the dialog
	// shows the fact as unavailable. The backend logs the failure.
	async #loadItemFact<Fact>(
		request: CancellableRequest<Fact>,
		store: (fact: Fact | null) => void,
	): Promise<void> {
		let fact: Fact | null
		try {
			fact = await this.#scope.track(request)
		} catch {
			fact = null
		}
		if (this.#scope.isCurrent) store(fact)
	}

	#changeItems(
		categories: readonly InventoryCategory[],
		change: (items: InventoryItem[]) => InventoryItem[],
	): void {
		const pagesByCategory = { ...this.pagesByCategory }
		for (const category of categories) {
			const page = pagesByCategory[category]
			if (page) pagesByCategory[category] = { ...page, items: change(page.items) }
		}
		this.pagesByCategory = pagesByCategory
	}

	async #load(cursor: string): Promise<void> {
		if (!this.#scope.isCurrent) return
		void this.#request?.cancel()
		const category = this.category,
			isMore = cursor !== "",
			request = accountBackend.GetInventoryPage(
				this.#scope.accountId,
				this.userId,
				category,
				cursor,
			)
		this.#request = request
		this.loading = !isMore
		this.loadingMore = isMore
		this.error = ""
		try {
			const received = await this.#scope.track(request)
			if (request !== this.#request || !this.#scope.isCurrent) return
			const previous = this.pagesByCategory[category],
				items = received.items ?? []
			this.pagesByCategory = {
				...this.pagesByCategory,
				[category]: {
					items: isMore && previous ? [...previous.items, ...items] : items,
					nextCursor: received.nextCursor,
					fetchedAt: isMore && previous ? previous.fetchedAt : Date.now(),
				},
			}
		} catch (error) {
			if (request !== this.#request || !this.#scope.isCurrent) return
			this.error = errorMessage(
				error,
				isMore
					? "Could not load more items. Try again."
					: "Could not load the inventory. Try again.",
			)
		} finally {
			if (request === this.#request) {
				this.#request = null
				this.loading = this.loadingMore = false
			}
		}
	}
}
