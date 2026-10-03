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
import type { AccountStore } from "./account-store.svelte"

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
	#active = true
	#generation = 0
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

	#current(generation: number): boolean {
		return (
			this.#active &&
			generation === this.#generation &&
			this.#store.vault.unlocked &&
			this.#store.selectedAccount.id === this.#accountId
		)
	}

	async refresh(): Promise<void> {
		if (!this.#active || this.loading || this.saving !== null) return
		const generation = ++this.#generation
		this.loading = true
		this.error = ""
		this.feedback = null
		try {
			const snapshot = await accountBackend.GetAccountSettings(this.#accountId)
			if (!this.#current(generation) || snapshot.accountId !== this.#accountId) {
				return
			}
			this.snapshot = snapshot
			this.refreshRequired = false
		} catch (error) {
			if (!this.#current(generation)) return
			this.error =
				error instanceof Error && error.message.trim()
					? error.message
					: "Could not load account settings. Try again."
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
			!this.#active ||
			!current.editable ||
			!option.enabled ||
			this.loading ||
			this.saving !== null ||
			this.refreshRequired
		) {
			return
		}
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
			const result = await accountBackend.UpdateAccountSetting(
				this.#accountId,
				change,
			)
			this.#handleSaveSuccess(result, current.key, generation)
		} catch (error) {
			this.#handleSaveError(error, current.key, generation)
		} finally {
			if (this.#current(generation)) this.saving = null
		}
	}

	#handleSaveSuccess(
		result: AccountSettingUpdateResult,
		key: AccountSettingKey,
		generation: number,
	): void {
		const success = result.outcome === "applied" || result.outcome === "unchanged"
		if (!this.#current(generation)) {
			if (!success) this.#notifyFailure(key)
			return
		}
		if (result.snapshot.accountId !== this.#accountId) return
		this.snapshot = result.snapshot
		this.refreshRequired = result.outcome === "unconfirmed"
		this.feedback = { key, message: result.message, success }
	}

	#handleSaveError(error: unknown, key: AccountSettingKey, generation: number): void {
		if (!this.#current(generation)) {
			this.#notifyFailure(key)
			return
		}
		this.refreshRequired = true
		this.feedback = {
			key,
			message:
				error instanceof Error && error.message.trim()
					? error.message
					: "Could not save this setting. Refresh settings to check.",
			success: false,
		}
	}

	#notifyFailure(key: AccountSettingKey): void {
		const account = this.#store.accounts.find((item) => item.id === this.#accountId)
		if (!this.#store.vault.unlocked || !account) return
		this.#notifications.show({
			id: `account-setting-${this.#accountId}-${key}`,
			title: "Check account settings",
			message: `A setting change for @${account.username} could not be confirmed. Open that account's settings and refresh before trying again.`,
		})
	}

	dispose(): void {
		this.#active = false
		this.#generation++
		this.snapshot = null
		this.feedback = null
		this.error = ""
		this.loading = false
		this.saving = null
		this.refreshRequired = false
	}
}
