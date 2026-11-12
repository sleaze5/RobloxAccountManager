import type { AccountStore } from "./account-store.svelte"

interface CancellableRequest<T> extends Promise<T> {
	cancel(): unknown
}

export interface KeptPageData<Value> {
	value: Value
	fetchedAt: number
}

export class AccountScope {
	readonly accountId: number
	readonly #store: AccountStore
	readonly #requests = new Set<CancellableRequest<unknown>>()
	#isActive = true

	constructor(accountId: number, store: AccountStore) {
		this.accountId = accountId
		this.#store = store
	}

	get isCurrent(): boolean {
		return (
			this.#isActive &&
			this.#store.vault.unlocked &&
			this.#store.selectedAccount.id === this.accountId
		)
	}

	async track<T>(request: CancellableRequest<T>): Promise<T> {
		this.#requests.add(request)
		try {
			return await request
		} finally {
			this.#requests.delete(request)
		}
	}

	// readKept returns what the named page showed for the account when it last
	// closed, so that the page opens with it while it refreshes.
	readKept<Value>(pageName: string): KeptPageData<Value> | null {
		return this.#store.readKeptPageData(
			this.accountId,
			pageName,
		) as KeptPageData<Value> | null
	}

	// keep stores what the named page shows when it closes.
	keep<Value>(pageName: string, kept: KeptPageData<Value>): void {
		this.#store.keepPageData(this.accountId, pageName, kept)
	}

	dispose(): void {
		this.#isActive = false
		for (const request of this.#requests) void request.cancel()
		this.#requests.clear()
	}
}
