import type { AccountScope } from "../accounts/account-scope"
import type { CancellableRequest } from "./cancellable-request"
import { errorMessage } from "./errors"

// CursorPagedList reads a list that Roblox serves in pages, such as the
// members of a group, for one account. Each page adds to the items that are
// already loaded.
export class CursorPagedList<Page extends { nextCursor: string }, Item> {
	items = $state.raw<Item[]>([])
	nextCursor = $state("")
	// Whether the first page loaded, so that an empty list is known to be
	// empty.
	hasLoaded = $state(false)
	isLoading = $state(false)
	isLoadingMore = $state(false)
	error = $state("")
	readonly #scope: AccountScope
	readonly #readPage: (cursor: string) => CancellableRequest<Page>
	readonly #itemsOf: (page: Page) => Item[] | null
	readonly #loadError: string
	#request: CancellableRequest<Page> | null = null

	constructor(
		scope: AccountScope,
		readPage: (cursor: string) => CancellableRequest<Page>,
		itemsOf: (page: Page) => Item[] | null,
		loadError: string,
	) {
		this.#scope = scope
		this.#readPage = readPage
		this.#itemsOf = itemsOf
		this.#loadError = loadError
	}

	// refresh reads the first page again and keeps the loaded items visible
	// until it arrives.
	async refresh(): Promise<void> {
		await this.#load("")
	}

	async loadMore(): Promise<void> {
		if (!this.nextCursor || this.isLoading || this.isLoadingMore) return
		await this.#load(this.nextCursor)
	}

	// changeItems applies a change that the account made, such as a removed
	// member, without reading the list again.
	changeItems(change: (items: Item[]) => Item[]): void {
		this.items = change(this.items)
	}

	async #load(cursor: string): Promise<void> {
		if (!this.#scope.isCurrent) return
		void this.#request?.cancel()
		const isMore = cursor !== "",
			request = this.#readPage(cursor)
		this.#request = request
		this.isLoading = !isMore
		this.isLoadingMore = isMore
		this.error = ""
		try {
			const page = await this.#scope.track(request)
			if (request !== this.#request || !this.#scope.isCurrent) return
			const items = this.#itemsOf(page) ?? []
			this.items = isMore ? [...this.items, ...items] : items
			this.nextCursor = page.nextCursor
			this.hasLoaded = true
		} catch (error) {
			if (request !== this.#request || !this.#scope.isCurrent) return
			this.error = errorMessage(
				error,
				isMore ? "Could not load more. Try again." : this.#loadError,
			)
		} finally {
			if (request === this.#request) {
				this.#request = null
				this.isLoading = this.isLoadingMore = false
			}
		}
	}
}
