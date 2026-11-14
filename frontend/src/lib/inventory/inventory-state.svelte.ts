import { accountBackend, CatalogItemType, InventoryCategory } from "../backend/bridge"
import type { InventoryItem } from "../backend/bridge"
import type { AccountStore } from "../accounts/account-store.svelte"
import { AccountScope } from "../accounts/account-scope"
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
	itemActionError = $state("")
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
