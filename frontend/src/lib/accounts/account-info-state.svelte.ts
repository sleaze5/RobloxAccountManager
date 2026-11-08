import { accountBackend } from "../backend/bridge"
import type { AccountInfoSnapshot } from "../backend/bridge"
import type { AccountStore } from "./account-store.svelte"
import { AccountScope } from "./account-scope"
import { errorMessage } from "../shared/errors"

export class AccountInfoState {
	snapshot = $state<AccountInfoSnapshot | null>(null)
	loading = $state(false)
	error = $state("")
	readonly #scope: AccountScope

	constructor(accountId: number, store: AccountStore) {
		this.#scope = new AccountScope(accountId, store)
	}

	async refresh(): Promise<void> {
		if (!this.#scope.isCurrent || this.loading) return
		this.loading = true
		this.error = ""
		try {
			const snapshot = await this.#scope.track(
				accountBackend.GetAccountInfo(this.#scope.accountId),
			)
			if (
				!this.#scope.isCurrent ||
				snapshot.accountId !== this.#scope.accountId
			) {
				return
			}
			this.snapshot = snapshot
		} catch (error) {
			if (!this.#scope.isCurrent) return
			this.error = errorMessage(
				error,
				"Could not load account info. Refresh to retry.",
			)
		} finally {
			if (this.#scope.isCurrent) this.loading = false
		}
	}

	dispose(): void {
		this.#scope.dispose()
		this.snapshot = null
		this.loading = false
		this.error = ""
	}
}
