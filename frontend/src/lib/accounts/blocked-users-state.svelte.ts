import { accountBackend } from "../backend/bridge"
import type { BlockedUser } from "../backend/bridge"
import type { AccountStore } from "./account-store.svelte"
import { AccountScope } from "./account-scope"
import { errorMessage } from "../shared/errors"
import { isFresh } from "../shared/freshness"

const keptPageName = "blocked-users"

export class BlockedUsersState {
	users = $state<BlockedUser[] | null>(null)
	loading = $state(false)
	unblocking = $state<number | null>(null)
	error = $state("")
	#fetchedAt = 0
	readonly #scope: AccountScope

	constructor(accountId: number, store: AccountStore) {
		this.#scope = new AccountScope(accountId, store)
		const kept = this.#scope.readKept<BlockedUser[]>(keptPageName)
		if (kept) {
			this.users = kept.value
			this.#fetchedAt = kept.fetchedAt
		}
	}

	get isFresh(): boolean {
		return isFresh(this.#fetchedAt)
	}

	async refresh(): Promise<void> {
		if (!this.#scope.isCurrent || this.loading || this.unblocking !== null) return
		this.loading = true
		this.error = ""
		try {
			const users = await this.#scope.track(
				accountBackend.GetBlockedUsers(this.#scope.accountId),
			)
			if (!this.#scope.isCurrent) return
			this.users = users
			this.#fetchedAt = Date.now()
		} catch (error) {
			if (!this.#scope.isCurrent) return
			this.error = errorMessage(
				error,
				"Could not load blocked users. Refresh to retry.",
			)
		} finally {
			if (this.#scope.isCurrent) this.loading = false
		}
	}

	async unblock(user: BlockedUser): Promise<void> {
		if (!this.#scope.isCurrent || this.loading || this.unblocking !== null) return
		this.unblocking = user.userId
		this.error = ""
		try {
			await accountBackend.UnblockUser(this.#scope.accountId, user.userId)
			if (!this.#scope.isCurrent) return
			this.users = (this.users ?? []).filter(
				(blockedUser) => blockedUser.userId !== user.userId,
			)
		} catch (error) {
			if (!this.#scope.isCurrent) return
			this.error = errorMessage(
				error,
				"Could not unblock this user. Refresh to check.",
			)
		} finally {
			if (this.#scope.isCurrent) this.unblocking = null
		}
	}

	dispose(): void {
		if (this.users) {
			this.#scope.keep(keptPageName, {
				value: this.users,
				fetchedAt: this.#fetchedAt,
			})
		}
		this.#scope.dispose()
		this.users = null
		this.loading = false
		this.unblocking = null
		this.error = ""
	}
}
