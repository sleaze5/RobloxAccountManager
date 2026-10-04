import { accountBackend } from "../backend/bridge"
import type { AccountInfoSnapshot } from "../backend/bridge"
import type { AccountStore } from "./account-store.svelte"

export class AccountInfoState {
	snapshot = $state<AccountInfoSnapshot | null>(null)
	loading = $state(false)
	error = $state("")
	#active = true
	#request: ReturnType<typeof accountBackend.GetAccountInfo> | null = null
	readonly #accountId: number
	readonly #store: AccountStore

	constructor(accountId: number, store: AccountStore) {
		this.#accountId = accountId
		this.#store = store
	}

	#current(): boolean {
		return (
			this.#active &&
			this.#store.vault.unlocked &&
			this.#store.selectedAccount.id === this.#accountId
		)
	}

	async refresh(): Promise<void> {
		if (!this.#current() || this.loading) return
		this.loading = true
		this.error = ""
		try {
			this.#request = accountBackend.GetAccountInfo(this.#accountId)
			const snapshot = await this.#request
			if (!this.#current() || snapshot.accountId !== this.#accountId) return
			this.snapshot = snapshot
		} catch (error) {
			if (!this.#current()) return
			this.error =
				error instanceof Error && error.message.trim()
					? error.message
					: "Could not load account info. Refresh to retry."
		} finally {
			this.#request = null
			if (this.#current()) this.loading = false
		}
	}

	dispose(): void {
		this.#active = false
		void this.#request?.cancel()
		this.#request = null
		this.snapshot = null
		this.loading = false
		this.error = ""
	}
}
