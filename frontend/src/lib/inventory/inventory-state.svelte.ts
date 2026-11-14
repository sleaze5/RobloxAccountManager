import { Browser } from "@wailsio/runtime"
import { accountBackend, InventoryCategory } from "../backend/bridge"
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

	async openItem(item: InventoryItem): Promise<void> {
		try {
			await Browser.OpenURL(item.pageUrl)
		} catch (error) {
			this.error = errorMessage(error, "Could not open the item. Try again.")
		}
	}

	dispose(): void {
		this.#request = null
		this.#scope.dispose()
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
