import {
	accountBackend,
	AccountProfileSection,
	CatalogItemType,
	InventoryCategory,
} from "../backend/bridge"
import type { InventoryItem, InventoryItemDetails } from "../backend/bridge"
import type { AccountStore } from "../accounts/account-store.svelte"
import { AccountScope } from "../accounts/account-scope"
import type { CancellableRequest } from "../accounts/account-scope"
import { errorMessage } from "../shared/errors"
import { isFresh } from "../shared/freshness"

interface InventoryCategoryPage {
	items: InventoryItem[]
	nextCursor: string
	fetchedAt: number
}

// inventoryItemKey tells apart an asset and a bundle that share an ID.
export function inventoryItemKey(item: InventoryItem): string {
	return `${item.catalogItemType}:${item.itemId}`
}

// categoryItemKey tells apart the items of different categories that share an
// ID, such as a badge and a game pass.
export function categoryItemKey(category: InventoryCategory, itemId: number): string {
	return `${category}:${itemId}`
}

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
	readonly userId: number
	readonly #scope: AccountScope
	#request: ReturnType<typeof accountBackend.GetInventoryPage> | null = null

	constructor(accountId: number, userId: number, store: AccountStore) {
		this.#scope = new AccountScope(accountId, store)
		this.userId = userId
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
		void this.#loadWornAssets()
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

	// loadItemDetails reads the details of an item in the current category, and
	// whether the account owns it when the inventory belongs to another user.
	loadItemDetails(item: InventoryItem, shouldCheckOwnership: boolean): void {
		const category = this.category,
			key = categoryItemKey(category, item.itemId)
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
			this.#changeItems((items) =>
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
		let robuxBalance: number | null
		try {
			const profile = await this.#scope.track(
				accountBackend.GetAccountProfile(
					this.#scope.accountId,
					AccountProfileSection.ProfileSectionRobux,
				),
			)
			robuxBalance = profile.robux
		} catch {
			// The confirmation shows the balance as unavailable, and the backend
			// logs the failure.
			robuxBalance = null
		}
		if (this.#scope.isCurrent) this.robuxBalance = robuxBalance
	}

	// buyItem buys an item of the current category for the account at the
	// price that the user confirmed, and reports whether Roblox sold it.
	async buyItem(item: InventoryItem, priceInRobux: number): Promise<boolean> {
		if (this.isBuyingItem) return false
		const category = this.category,
			key = categoryItemKey(category, item.itemId)
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
			this.#changeItems((items) =>
				items.filter(
					(listedItem) =>
						listedItem.itemId !== item.itemId ||
						listedItem.catalogItemType === CatalogItemType.CatalogBundle,
				),
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

	async #loadWornAssets(): Promise<void> {
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

	#changeItems(change: (items: InventoryItem[]) => InventoryItem[]): void {
		const pagesByCategory: typeof this.pagesByCategory = {}
		for (const category of Object.values(InventoryCategory)) {
			const page = this.pagesByCategory[category]
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
