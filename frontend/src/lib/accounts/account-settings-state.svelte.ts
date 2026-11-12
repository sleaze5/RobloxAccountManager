import { accountBackend } from "../backend/bridge"
import type {
	AccountSettingChange,
	AccountSettingKey,
	AccountSettingOption,
	AccountSettingsSnapshot,
	AccountSettingUpdateResult,
	AccountSettingView,
} from "../backend/bridge"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import { AccountScope } from "./account-scope"
import type { AccountStore } from "./account-store.svelte"
import { errorMessage } from "../shared/errors"
import { isFresh } from "../shared/freshness"

const keptPageName = "account-settings"

export class AccountSettingsState {
	snapshot = $state<AccountSettingsSnapshot | null>(null)
	loading = $state(false)
	saving = $state<AccountSettingKey | null>(null)
	error = $state("")
	refreshRequired = $state(false)
	feedback = $state<{
		key: AccountSettingKey
		message: string
		success: boolean
	} | null>(null)
	#generation = 0
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
		const kept = this.#scope.readKept<AccountSettingsSnapshot>(keptPageName)
		if (kept) {
			this.snapshot = kept.value
			this.#fetchedAt = kept.fetchedAt
		}
	}

	get isFresh(): boolean {
		return isFresh(this.#fetchedAt)
	}

	#current(generation: number): boolean {
		return this.#scope.isCurrent && generation === this.#generation
	}

	async refresh(): Promise<void> {
		if (!this.#scope.isCurrent || this.loading || this.saving !== null) return
		const generation = ++this.#generation
		this.loading = true
		this.error = ""
		this.feedback = null
		try {
			const snapshot = await this.#scope.track(
				accountBackend.GetAccountSettings(this.#scope.accountId),
			)
			if (
				!this.#current(generation) ||
				snapshot.accountId !== this.#scope.accountId
			) {
				return
			}
			this.snapshot = snapshot
			this.refreshRequired = false
			this.#fetchedAt = Date.now()
		} catch (error) {
			if (!this.#current(generation)) return
			this.error = errorMessage(
				error,
				"Could not load account settings. Try again.",
			)
			this.refreshRequired = true
		} finally {
			if (this.#current(generation)) this.loading = false
		}
	}

	async save(
		current: AccountSettingView,
		option: AccountSettingOption,
	): Promise<void> {
		if (
			!this.#scope.isCurrent ||
			!current.editable ||
			!option.enabled ||
			this.loading ||
			this.saving !== null ||
			this.refreshRequired
		) {
			return
		}
		// The kept settings stop being trusted once a save starts. dispose keeps
		// them again only after the save is confirmed, so a page that reopens
		// after an unconfirmed save loads the settings before allowing edits.
		this.#scope.forgetKept(keptPageName)
		const generation = ++this.#generation,
			change: AccountSettingChange = {
				key: current.key,
				stringValue: option.stringValue,
				boolValue: option.boolValue,
				expectedStringValue: current.stringValue,
				expectedBoolValue: current.boolValue,
			}
		this.saving = current.key
		this.feedback = null
		try {
			const updateResult = await accountBackend.UpdateAccountSetting(
				this.#scope.accountId,
				change,
			)
			this.#handleSaveSuccess(updateResult, current.key, generation)
		} catch (error) {
			this.#handleSaveError(error, current.key, generation)
		} finally {
			if (this.#current(generation)) this.saving = null
		}
	}

	#handleSaveSuccess(
		updateResult: AccountSettingUpdateResult,
		key: AccountSettingKey,
		generation: number,
	): void {
		const success =
			updateResult.outcome === "applied" || updateResult.outcome === "unchanged"
		if (!this.#current(generation)) {
			if (!success) this.#notifyFailure(key)
			return
		}
		if (updateResult.snapshot.accountId !== this.#scope.accountId) return
		this.snapshot = updateResult.snapshot
		this.refreshRequired = updateResult.outcome === "unconfirmed"
		this.feedback = { key, message: updateResult.message, success }
	}

	#handleSaveError(error: unknown, key: AccountSettingKey, generation: number): void {
		if (!this.#current(generation)) {
			this.#notifyFailure(key)
			return
		}
		this.refreshRequired = true
		this.feedback = {
			key,
			message: errorMessage(
				error,
				"Could not save this setting. Refresh settings to check.",
			),
			success: false,
		}
	}

	#notifyFailure(key: AccountSettingKey): void {
		const account = this.#store.accounts.find(
			(listedAccount) => listedAccount.id === this.#scope.accountId,
		)
		if (!this.#store.vault.unlocked || !account) return
		this.#notifications.show({
			id: `account-setting-${this.#scope.accountId}-${key}`,
			title: "Check account settings",
			tone: "warning",
			message: `A setting change for @${account.username} could not be confirmed. Open that account's settings and refresh before trying again.`,
		})
	}

	// A snapshot that needs a refresh is not kept, because a save left its
	// values unconfirmed.
	dispose(): void {
		if (this.snapshot && !this.refreshRequired && this.saving === null) {
			this.#scope.keep(keptPageName, {
				value: this.snapshot,
				fetchedAt: this.#fetchedAt,
			})
		}
		this.#scope.dispose()
		this.#generation++
		this.snapshot = null
		this.feedback = null
		this.error = ""
		this.loading = false
		this.saving = null
		this.refreshRequired = false
	}
}
