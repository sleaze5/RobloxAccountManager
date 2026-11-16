import {
	accountBackend,
	AccountProfileSection,
	InventoryCategory,
} from "../backend/bridge"
import type {
	InventoryItem,
	InventoryItemDetails,
	RelatedInventoryItem,
} from "../backend/bridge"
import type { AccountStore } from "../accounts/account-store.svelte"
import { AccountScope } from "../accounts/account-scope"
import type { CancellableRequest } from "../shared/cancellable-request"
import { errorMessage } from "../shared/errors"
import { isFresh } from "../shared/freshness"
import { RenderedModelLoad } from "../shared/rendered-model-load.svelte"
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
	// The assets that each opened item includes or uses, by categoryItemKey.
	// null means that the request failed.
	relatedItemsByItemKey = $state.raw(new Map<string, RelatedInventoryItem[] | null>())
	// The 3D model of the opened item. Its key is the categoryItemKey of the
	// item.
	readonly itemModel: RenderedModelLoad
	// The avatar of the account wearing the opened item, which Roblox renders
	// without saving it. tryOnKey is the categoryItemKey of the item.
	tryOnKey = $state("")
	tryOnImageUrl = $state("")
	isLoadingTryOn = $state(false)
	tryOnError = $state("")
	// The same as a 3D model. Its key is the categoryItemKey of the item.
	readonly tryOnModel: RenderedModelLoad
	readonly userId: number
	readonly #scope: AccountScope
	#request: ReturnType<typeof accountBackend.GetInventoryPage> | null = null
	#tryOnRequest: ReturnType<typeof accountBackend.TryOnInventoryItem> | null = null
	readonly #loadingRelatedItemKeys = new Set<string>()

	constructor(accountId: number, userId: number, store: AccountStore) {
		this.#scope = new AccountScope(accountId, store)
		this.itemModel = new RenderedModelLoad(this.#scope)
		this.tryOnModel = new RenderedModelLoad(this.#scope)
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

	async loadTryOnModel(
		item: InventoryItem,
		category: InventoryCategory,
	): Promise<void> {
		await this.tryOnModel.load(
			categoryItemKey(category, item.itemId),
			() =>
				accountBackend.GetInventoryItemTryOnModel(
					this.#scope.accountId,
					category,
					item.itemId,
				),
			"Could not try on the item in 3D. Try again.",
		)
	}

	// tryOnItem renders the avatar of the account wearing the item, unless it
	// is already shown or rendering. A failed render runs again on the next
	// call.
	async tryOnItem(item: InventoryItem, category: InventoryCategory): Promise<void> {
		const key = categoryItemKey(category, item.itemId)
		if (!this.#scope.isCurrent) return
		if (key === this.tryOnKey && (this.tryOnImageUrl || this.isLoadingTryOn)) return
		void this.#tryOnRequest?.cancel()
		const request = accountBackend.TryOnInventoryItem(
			this.#scope.accountId,
			category,
			item.itemId,
		)
		this.#tryOnRequest = request
		this.tryOnKey = key
		this.tryOnImageUrl = this.tryOnError = ""
		this.isLoadingTryOn = true
		try {
			const tryOnImageUrl = await this.#scope.track(request)
			if (request !== this.#tryOnRequest || !this.#scope.isCurrent) return
			this.tryOnImageUrl = tryOnImageUrl
		} catch (error) {
			if (request !== this.#tryOnRequest || !this.#scope.isCurrent) return
			this.tryOnError = errorMessage(
				error,
				"Could not try on the item. Try again.",
			)
		} finally {
			if (request === this.#tryOnRequest) {
				this.#tryOnRequest = null
				this.isLoadingTryOn = false
			}
		}
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
		this.#request = this.#tryOnRequest = null
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
