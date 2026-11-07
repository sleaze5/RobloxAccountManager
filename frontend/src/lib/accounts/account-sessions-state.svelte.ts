import { accountBackend } from "../backend/bridge"
import type { AccountSession } from "../backend/bridge"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import type { AccountStore } from "./account-store.svelte"
import { errorMessage } from "../shared/errors"

export class AccountSessionsState {
	sessions = $state<AccountSession[] | null>(null)
	loading = $state(false)
	signingOut = $state(false)
	error = $state("")
	signOutError = $state("")
	#active = true
	#request: ReturnType<typeof accountBackend.GetAccountSessions> | null = null
	readonly #accountId: number
	readonly #store: AccountStore
	readonly #notifications: NotificationCenter

	constructor(
		accountId: number,
		store: AccountStore,
		notifications: NotificationCenter,
	) {
		this.#accountId = accountId
		this.#store = store
		this.#notifications = notifications
	}

	#current(): boolean {
		return (
			this.#active &&
			this.#store.vault.unlocked &&
			this.#store.selectedAccount.id === this.#accountId
		)
	}

	async refresh(): Promise<void> {
		if (!this.#current() || this.loading || this.signingOut) return
		this.loading = true
		this.error = ""
		try {
			this.#request = accountBackend.GetAccountSessions(this.#accountId)
			const sessions = await this.#request
			if (!this.#current()) return
			this.sessions = sessions ?? []
		} catch (error) {
			if (!this.#current()) return
			this.error = errorMessage(
				error,
				"Could not load sessions. Refresh to retry.",
			)
		} finally {
			this.#request = null
			if (this.#current()) this.loading = false
		}
	}

	async signOutOthers(): Promise<boolean> {
		if (!this.#current() || this.loading || this.signingOut || this.#store.busy) {
			return false
		}
		this.signingOut = true
		this.signOutError = ""
		this.#store.busy = true
		try {
			const result = await accountBackend.SignOutOtherSessions(this.#accountId)
			this.#notifications.show({
				id: `sessions-sign-out-${this.#accountId}`,
				tone: result.warning ? "warning" : "success",
				title: "Other sessions signed out",
				message:
					result.warning ||
					"Every other browser, device, and app was signed out. This app stays signed in.",
			})
			return true
		} catch (error) {
			if (this.#current()) {
				this.signOutError = errorMessage(
					error,
					"Could not sign out other sessions. Refresh sessions to check.",
				)
			}
			return false
		} finally {
			this.#store.busy = false
			this.signingOut = false
			if (this.#current()) {
				void this.#store.refreshAccounts()
				void this.refresh()
			}
		}
	}

	dispose(): void {
		this.#active = false
		void this.#request?.cancel()
		this.#request = null
		this.sessions = null
		this.loading = false
		this.error = ""
		this.signOutError = ""
	}
}
