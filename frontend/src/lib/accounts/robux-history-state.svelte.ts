import { accountBackend, RobuxTransactionType } from "../backend/bridge"
import type { RobuxTransaction } from "../backend/bridge"
import type { AccountStore } from "./account-store.svelte"
import { AccountScope } from "./account-scope"
import { errorMessage } from "../shared/errors"
import { isFresh } from "../shared/freshness"

const keptPageName = "robux-history"

export class RobuxHistoryState {
	transactionType = $state<RobuxTransactionType>(RobuxTransactionType.RobuxPurchases)
	transactions = $state<RobuxTransaction[] | null>(null)
	nextCursor = $state("")
	loading = $state(false)
	loadingMore = $state(false)
	error = $state("")
	#requestCount = 0
	#fetchedAt = 0
	readonly #scope: AccountScope

	constructor(accountId: number, store: AccountStore) {
		this.#scope = new AccountScope(accountId, store)
		const kept = this.#scope.readKept<{
			transactionType: RobuxTransactionType
			transactions: RobuxTransaction[]
			nextCursor: string
		}>(keptPageName)
		if (kept) {
			this.transactionType = kept.value.transactionType
			this.transactions = kept.value.transactions
			this.nextCursor = kept.value.nextCursor
			this.#fetchedAt = kept.fetchedAt
		}
	}

	get isFresh(): boolean {
		return isFresh(this.#fetchedAt)
	}

	selectType(transactionType: RobuxTransactionType): void {
		if (transactionType === this.transactionType) return
		this.transactionType = transactionType
		this.transactions = null
		this.nextCursor = ""
		this.#fetchedAt = 0
		this.loading = false
		this.loadingMore = false
		void this.refresh()
	}

	async refresh(): Promise<void> {
		if (!this.#scope.isCurrent || this.loading) return
		await this.#load(false)
	}

	async loadMore(): Promise<void> {
		if (
			!this.#scope.isCurrent ||
			this.loading ||
			this.loadingMore ||
			!this.nextCursor
		) {
			return
		}
		await this.#load(true)
	}

	async #load(isMore: boolean): Promise<void> {
		const requestCount = ++this.#requestCount,
			transactionType = this.transactionType,
			isCurrentRequest = () =>
				this.#scope.isCurrent && requestCount === this.#requestCount
		if (isMore) this.loadingMore = true
		else this.loading = true
		this.error = ""
		try {
			const page = await this.#scope.track(
				accountBackend.GetRobuxTransactions(
					this.#scope.accountId,
					transactionType,
					isMore ? this.nextCursor : "",
				),
			)
			if (!isCurrentRequest()) return
			const received = page.transactions ?? []
			this.transactions = isMore
				? [...(this.transactions ?? []), ...received]
				: received
			this.nextCursor = page.nextCursor
			if (!isMore) this.#fetchedAt = Date.now()
		} catch (error) {
			if (!isCurrentRequest()) return
			this.error = errorMessage(
				error,
				"Could not load the Robux history. Refresh to retry.",
			)
		} finally {
			if (isCurrentRequest()) {
				this.loading = false
				this.loadingMore = false
			}
		}
	}

	dispose(): void {
		if (this.transactions) {
			this.#scope.keep(keptPageName, {
				value: {
					transactionType: this.transactionType,
					transactions: this.transactions,
					nextCursor: this.nextCursor,
				},
				fetchedAt: this.#fetchedAt,
			})
		}
		this.#scope.dispose()
		this.#requestCount++
		this.transactions = null
		this.loading = false
		this.loadingMore = false
		this.error = ""
	}
}
