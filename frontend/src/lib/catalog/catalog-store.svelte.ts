import {
	accountBackend,
	CatalogCategory,
	CatalogCreatorType,
	CatalogSalesType,
	CatalogSortOrder,
	CatalogSortPeriod,
} from "../backend/bridge"
import type { CatalogSearch, CategorizedInventoryItem } from "../backend/bridge"
import type { AccountStore } from "../accounts/account-store.svelte"
import { errorMessage } from "../shared/errors"
import { isFresh } from "../shared/freshness"

export function defaultCatalogSearch(): CatalogSearch {
	return {
		keyword: "",
		category: CatalogCategory.CatalogAll,
		creatorName: "",
		creatorType: CatalogCreatorType.CatalogCreatorUser,
		minPriceInRobux: null,
		maxPriceInRobux: null,
		sortOrder: CatalogSortOrder.CatalogSortByRelevance,
		sortPeriod: CatalogSortPeriod.CatalogAllTime,
		salesType: CatalogSalesType.CatalogAnySalesType,
		shouldIncludeHiddenItems: false,
	}
}

// countActiveCatalogFilters counts the filters of the Filters panel that
// narrow the results, so that its button can show the count.
export function countActiveCatalogFilters(search: CatalogSearch): number {
	return [
		search.creatorName !== "",
		search.minPriceInRobux !== null || search.maxPriceInRobux !== null,
		search.salesType !== CatalogSalesType.CatalogAnySalesType,
		search.shouldIncludeHiddenItems,
	].filter(Boolean).length
}

// CatalogStore searches the Marketplace for the selected account. The search
// and its results stay while the app runs, so that the page opens where the
// user left it.
export class CatalogStore {
	sidebarCollapsed = $state(false)
	sidebarWidth = $state(248)
	// keywordInput is the text in the search field, which applies on submit.
	keywordInput = $state("")
	search = $state.raw<CatalogSearch>(defaultCatalogSearch())
	items = $state.raw<CategorizedInventoryItem[]>([])
	nextCursor = $state("")
	hasResults = $state(false)
	loading = $state(false)
	loadingMore = $state(false)
	error = $state("")
	// resultsVersion changes with each new search, so that the page scrolls
	// back to the top.
	resultsVersion = $state(0)
	#fetchedAt = 0
	#request: ReturnType<typeof accountBackend.SearchCatalog> | null = null
	readonly #accounts: AccountStore

	constructor(accounts: AccountStore) {
		this.#accounts = accounts
	}

	// refreshWhenStale searches again when the results are missing or older
	// than a minute, such as when the page opens, unless a search runs.
	refreshWhenStale(): void {
		if (this.loading) return
		if (!this.hasResults) void this.#load("", true)
		else if (!isFresh(this.#fetchedAt)) void this.refresh()
	}

	applySearch(change: Partial<CatalogSearch>): void {
		this.search = { ...this.search, ...change }
		void this.#load("", true)
	}

	submitKeyword(): void {
		const keyword = this.keywordInput.trim()
		this.keywordInput = keyword
		this.applySearch({ keyword })
	}

	clearFilters(): void {
		this.keywordInput = ""
		this.search = { ...defaultCatalogSearch(), category: this.search.category }
		void this.#load("", true)
	}

	// showCreations lists every item of a user or a group, including the
	// items that the Marketplace hides, newest first.
	showCreations(creatorName: string, creatorType: CatalogCreatorType): void {
		this.keywordInput = ""
		this.search = {
			...defaultCatalogSearch(),
			creatorName,
			creatorType,
			sortOrder: CatalogSortOrder.CatalogSortByRecentlyPublished,
			shouldIncludeHiddenItems: true,
		}
		void this.#load("", true)
	}

	// refresh searches again and keeps the results visible until the new ones
	// arrive.
	async refresh(): Promise<void> {
		await this.#load("", false)
	}

	async loadMore(): Promise<void> {
		if (!this.nextCursor || this.loading || this.loadingMore) return
		await this.#load(this.nextCursor, false)
	}

	reset(): void {
		void this.#request?.cancel()
		this.#request = null
		this.keywordInput = ""
		this.search = defaultCatalogSearch()
		this.items = []
		this.nextCursor = this.error = ""
		this.hasResults = this.loading = this.loadingMore = false
		this.#fetchedAt = 0
	}

	async #load(cursor: string, isNewSearch: boolean): Promise<void> {
		const accountId = this.#accounts.selectedAccount.id
		if (accountId === 0 || !this.#accounts.vault.unlocked) return
		void this.#request?.cancel()
		const isMore = cursor !== "",
			search = this.search,
			request = accountBackend.SearchCatalog(accountId, search, cursor)
		this.#request = request
		this.loading = !isMore
		this.loadingMore = isMore
		this.error = ""
		if (isNewSearch) {
			this.hasResults = false
			this.resultsVersion++
		}
		try {
			const page = await request
			if (request !== this.#request) return
			const items = page.items ?? []
			this.items = isMore ? [...this.items, ...items] : items
			this.nextCursor = page.nextCursor
			this.hasResults = true
			if (!isMore) this.#fetchedAt = Date.now()
		} catch (error) {
			if (request !== this.#request) return
			this.error = errorMessage(
				error,
				isMore
					? "Could not load more items. Try again."
					: "Could not search the Marketplace. Try again.",
			)
		} finally {
			if (request === this.#request) {
				this.#request = null
				this.loading = this.loadingMore = false
			}
		}
	}
}
