import { accountBackend } from "../backend/bridge"
import type { BlockedUser } from "../backend/bridge"
import type { AccountStore } from "./account-store.svelte"

export class BlockedUsersState {
	users = $state<BlockedUser[] | null>(null)
	loading = $state(false)
	unblocking = $state<number | null>(null)
	error = $state("")
	#active = true
	#request: ReturnType<typeof accountBackend.GetBlockedUsers> | null = null
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
		if (!this.#current() || this.loading || this.unblocking !== null) return
		this.loading = true
		this.error = ""
		try {
			this.#request = accountBackend.GetBlockedUsers(this.#accountId)
			const users = await this.#request
			if (!this.#current()) return
			this.users = users
		} catch (error) {
			if (!this.#current()) return
			this.error = message(
				error,
				"Could not load blocked users. Refresh to retry.",
			)
		} finally {
			this.#request = null
			if (this.#current()) this.loading = false
		}
	}

	async unblock(user: BlockedUser): Promise<void> {
		if (!this.#current() || this.loading || this.unblocking !== null) return
		this.unblocking = user.userId
		this.error = ""
		try {
			await accountBackend.UnblockUser(this.#accountId, user.userId)
			if (!this.#current()) return
			this.users = (this.users ?? []).filter(
				(item) => item.userId !== user.userId,
			)
		} catch (error) {
			if (!this.#current()) return
			this.error = message(
				error,
				"Could not unblock this user. Refresh to check.",
			)
		} finally {
			if (this.#current()) this.unblocking = null
		}
	}

	dispose(): void {
		this.#active = false
		void this.#request?.cancel()
		this.#request = null
		this.users = null
		this.loading = false
		this.unblocking = null
		this.error = ""
	}
}

function message(error: unknown, fallback: string): string {
	return error instanceof Error && error.message.trim() ? error.message : fallback
}
