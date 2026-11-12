import { accountBackend } from "../backend/bridge"
import type { AccountSession } from "../backend/bridge"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import type { AccountStore } from "./account-store.svelte"
import { AccountScope } from "./account-scope"
import { errorMessage } from "../shared/errors"
import { isFresh } from "../shared/freshness"

const keptPageName = "account-sessions"

export class AccountSessionsState {
	sessions = $state<AccountSession[] | null>(null)
	loading = $state(false)
	signingOut = $state(false)
	error = $state("")
	signOutError = $state("")
	#fetchedAt = 0
	readonly #scope: AccountScope
	readonly #store: AccountStore
	readonly #notifications: NotificationCenter

	constructor(
		accountId: number,
		store: AccountStore,
		notifications: NotificationCenter,
	) {
		this.#scope = new AccountScope(accountId, store)
		this.#store = store
		this.#notifications = notifications
		const kept = this.#scope.readKept<AccountSession[]>(keptPageName)
		if (kept) {
			this.sessions = kept.value
			this.#fetchedAt = kept.fetchedAt
		}
	}

	get isFresh(): boolean {
		return isFresh(this.#fetchedAt)
	}

	async refresh(): Promise<void> {
		if (!this.#scope.isCurrent || this.loading || this.signingOut) return
		this.loading = true
		this.error = ""
		try {
			const sessions = await this.#scope.track(
				accountBackend.GetAccountSessions(this.#scope.accountId),
			)
			if (!this.#scope.isCurrent) return
			this.sessions = sessions ?? []
			this.#fetchedAt = Date.now()
		} catch (error) {
			if (!this.#scope.isCurrent) return
			this.error = errorMessage(
				error,
				"Could not load sessions. Refresh to retry.",
			)
		} finally {
			if (this.#scope.isCurrent) this.loading = false
		}
	}

	async signOutOthers(): Promise<boolean> {
		if (
			!this.#scope.isCurrent ||
			this.loading ||
			this.signingOut ||
			this.#store.busy
		) {
			return false
		}
		this.signingOut = true
		this.signOutError = ""
		this.#store.busy = true
		try {
			const signOutResult = await accountBackend.SignOutOtherSessions(
				this.#scope.accountId,
			)
			this.#notifications.show({
				id: `sessions-sign-out-${this.#scope.accountId}`,
				tone: signOutResult.warning ? "warning" : "success",
				title: "Other sessions signed out",
				message:
					signOutResult.warning ||
					"Every other browser, device, and app was signed out. This app stays signed in.",
			})
			return true
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.signOutError = errorMessage(
					error,
					"Could not sign out other sessions. Refresh sessions to check.",
				)
			}
			return false
		} finally {
			this.#store.busy = false
			this.signingOut = false
			if (this.#scope.isCurrent) {
				void this.#store.refreshAccounts()
				void this.refresh()
			}
		}
	}

	dispose(): void {
		if (this.sessions) {
			this.#scope.keep(keptPageName, {
				value: this.sessions,
				fetchedAt: this.#fetchedAt,
			})
		}
		this.#scope.dispose()
		this.sessions = null
		this.loading = false
		this.error = ""
		this.signOutError = ""
	}
}
