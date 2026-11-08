import type { AccountStore } from "./account-store.svelte"

interface CancellableRequest<T> extends Promise<T> {
	cancel(): unknown
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

	dispose(): void {
		this.#isActive = false
		for (const request of this.#requests) void request.cancel()
		this.#requests.clear()
	}
}
