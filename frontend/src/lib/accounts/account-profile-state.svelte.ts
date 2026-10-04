import { accountBackend } from "../backend/bridge"
import type { AccountProfileSnapshot } from "../backend/bridge"
import { detailsKeepMs, isFresh } from "../shared/freshness"
import type { AccountStore } from "./account-store.svelte"

const fieldsBySource = {
	account: ["ageBracket", "countryCode", "premium", "plus"],
	robux: ["robux"],
	pendingRobux: ["pendingRobux"],
	ageGroup: ["ageGroup", "ageVerification"],
	ageVerification: ["ageVerification"],
	twoStep: ["twoStepEnabled", "twoStepMethods"],
	about: ["description", "verifiedBadge"],
	friends: ["friendCount"],
	followers: ["followerCount"],
	following: ["followingCount"],
	primaryGroup: ["primaryGroup"],
} as const

export class AccountProfileState {
	snapshot = $state<AccountProfileSnapshot | null>(null)
	loading = $state(false)
	error = $state("")
	#active = true
	#fetchedAt = 0
	#request: ReturnType<typeof accountBackend.GetAccountProfile> | null = null
	#validationRequest: ReturnType<typeof accountBackend.ValidateAccount> | null = null
	readonly #accountId: number
	readonly #store: AccountStore

	constructor(accountId: number, store: AccountStore) {
		this.#accountId = accountId
		this.#store = store
	}

	#current(): boolean {
		return this.#active && this.#store.vault.unlocked
	}

	enter(): void {
		if (isFresh(this.#fetchedAt)) return
		if (Date.now() - this.#fetchedAt >= detailsKeepMs) this.snapshot = null
		void this.refresh()
	}

	async refresh(validateIdentity = false): Promise<void> {
		if (!this.#current() || this.loading) return
		this.loading = true
		this.error = ""
		try {
			const request = accountBackend.GetAccountProfile(this.#accountId)
			this.#request = request
			const [profile, identity] = await Promise.allSettled([
				request,
				validateIdentity ? this.#refreshIdentity() : null,
			])
			if (!this.#current()) return
			if (profile.status === "rejected") throw profile.reason
			const snapshot = profile.value
			if (snapshot.accountId !== this.#accountId) {
				throw new Error("The account changed while loading. Refresh to retry.")
			}
			this.snapshot = this.#mergeSnapshot(snapshot)
			this.#fetchedAt = Date.now()
			if (identity.status === "rejected") throw identity.reason
		} catch (error) {
			if (!this.#current()) return
			this.error =
				error instanceof Error && error.message.trim()
					? error.message
					: "Could not refresh profile details. Refresh to retry."
		} finally {
			this.#request = null
			this.#validationRequest = null
			if (this.#current()) this.loading = false
		}
	}

	async #refreshIdentity(): Promise<void> {
		this.#validationRequest = accountBackend.ValidateAccount(this.#accountId)
		const results = await Promise.allSettled([
			this.#validationRequest,
			this.#store.refreshAccountAvatar(this.#accountId),
			this.#store.refreshPresences(this.#accountId),
		])
		for (const result of results) {
			if (result.status === "rejected") throw result.reason
		}
	}

	#mergeSnapshot(next: AccountProfileSnapshot): AccountProfileSnapshot {
		const previous = this.snapshot
		if (previous) {
			for (const [source, fields] of Object.entries(fieldsBySource)) {
				if (next.unavailable?.includes(source)) {
					Object.assign(
						next,
						Object.fromEntries(
							fields.map((field) => [field, previous[field]]),
						),
					)
				}
			}
		}
		return next
	}

	dispose(): void {
		this.#active = false
		void this.#request?.cancel()
		void this.#validationRequest?.cancel()
		this.#request = null
		this.#validationRequest = null
		this.snapshot = null
		this.loading = false
		this.error = ""
	}
}
