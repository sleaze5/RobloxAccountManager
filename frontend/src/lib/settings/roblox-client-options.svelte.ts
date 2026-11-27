import {
	accountBackend,
	type MultiClientSnapshot,
	type RobloxCleanerState,
	type RobloxCleanReport,
} from "../backend/bridge"
import { errorMessage } from "../shared/errors"

type StatusTone = "neutral" | "success" | "warning"

type CleanerSavingFlag =
	| "isSavingCleanBeforeLaunch"
	| "isSavingDeleteLogs"
	| "isSavingAdministratorRights"

// The status of Multi-client changes while Roblox processes start and stop.
const multiClientPollIntervalMilliseconds = 500

// Client options are shared by the Client options menu in the launch panel
// and Settings > Roblox, so a change in one place shows in the other.
export function cleanReportSummary(report: RobloxCleanReport): string {
	const deleted = `Deleted ${report.deletedCount} ${report.deletedCount === 1 ? "item" : "items"}.`
	return report.skippedCount > 0
		? `${deleted} ${report.skippedCount} skipped.`
		: deleted
}

class RobloxClientOptions {
	multiClient = $state<MultiClientSnapshot | null>(null)
	cleaner = $state<RobloxCleanerState | null>(null)
	isLoadingMultiClient = $state(true)
	isSavingMultiClient = $state(false)
	isSavingCleanBeforeLaunch = $state(false)
	isSavingDeleteLogs = $state(false)
	isSavingAdministratorRights = $state(false)
	multiClientError = $state("")
	multiClientStatusError = $state("")
	cleanerError = $state("")
	#isCheckingMultiClient = false
	#multiClientRevision = 0
	#cleanerRevision = 0
	#cleanerQueue: Promise<void> = Promise.resolve()

	get multiClientStatus(): { text: string; tone: StatusTone } {
		const snapshot = this.multiClient
		if (this.isLoadingMultiClient) return { text: "Checking…", tone: "neutral" }
		if (this.isSavingMultiClient) return { text: "Saving…", tone: "neutral" }
		if (!snapshot) return { text: "Not checked", tone: "neutral" }
		if (!snapshot.supported) return { text: "Windows only", tone: "neutral" }
		if (!snapshot.enabled) return { text: "Off", tone: "neutral" }
		if (snapshot.ready) return { text: "Ready", tone: "success" }
		return { text: `Not ready: ${snapshot.message}`, tone: "warning" }
	}

	get isCleanBeforeLaunchActive(): boolean {
		return Boolean(
			this.cleaner?.cleanBeforeLaunch && !this.cleaner.isBlockedByMultiClient,
		)
	}

	async load(): Promise<void> {
		await Promise.all([this.refreshMultiClient(), this.refreshCleaner()])
	}

	// Returns the function that stops polling.
	pollMultiClientWhileEnabled(): () => void {
		const interval = setInterval(() => {
			if (this.multiClient?.enabled) void this.refreshMultiClient()
		}, multiClientPollIntervalMilliseconds)
		return () => clearInterval(interval)
	}

	async refreshMultiClient(): Promise<void> {
		if (this.#isCheckingMultiClient || this.isSavingMultiClient) return
		this.#isCheckingMultiClient = true
		this.isLoadingMultiClient = this.multiClient === null
		const requestRevision = this.#multiClientRevision
		try {
			const state = await accountBackend.GetMultiClientState()
			if (requestRevision !== this.#multiClientRevision) return
			this.multiClient = state
			this.multiClientStatusError = ""
		} catch {
			if (requestRevision !== this.#multiClientRevision) return
			this.multiClientStatusError =
				"Could not check the status of Multi-client. Try again."
		} finally {
			this.#isCheckingMultiClient = false
			if (requestRevision === this.#multiClientRevision) {
				this.isLoadingMultiClient = false
			}
		}
	}

	refreshCleaner(): Promise<void> {
		return this.#requestCleaner(
			() => accountBackend.GetRobloxCleanerState(),
			false,
			"Could not load the cleaning settings. Try again.",
		)
	}

	async setMultiClientEnabled(enabled: boolean): Promise<void> {
		if (
			this.isSavingMultiClient ||
			this.isLoadingMultiClient ||
			!this.multiClient?.supported
		) {
			return
		}
		const previous = this.multiClient
		this.#multiClientRevision++
		this.isSavingMultiClient = true
		this.multiClientError = ""
		try {
			this.multiClient = await accountBackend.SetMultiClientEnabled(enabled)
			this.multiClientStatusError = ""
		} catch (error) {
			this.multiClient = previous
			this.multiClientError = errorMessage(
				error,
				"Could not save the Multi-client setting. Try again.",
			)
		} finally {
			this.isSavingMultiClient = false
		}
		// Multi-client decides whether Roblox is cleaned before a launch.
		await this.refreshCleaner()
	}

	async setCleanBeforeLaunch(enabled: boolean): Promise<void> {
		if (!this.cleaner?.supported) return
		await this.#saveCleanerSetting(
			"isSavingCleanBeforeLaunch",
			() => accountBackend.SetCleanBeforeLaunch(enabled),
			"Could not save the setting. Try again.",
		)
	}

	async setDeleteLogsWhenCleaning(enabled: boolean): Promise<void> {
		if (!this.cleaner?.supported) return
		await this.#saveCleanerSetting(
			"isSavingDeleteLogs",
			() => accountBackend.SetDeleteLogsWhenCleaning(enabled),
			"Could not save the setting. Try again.",
		)
	}

	async setUseAdministratorRights(enabled: boolean): Promise<void> {
		if (!this.cleaner?.elevationSupported) return
		await this.#saveCleanerSetting(
			"isSavingAdministratorRights",
			() => accountBackend.SetUseAdministratorRights(enabled),
			"Could not save the setting. Try again.",
		)
	}

	async #saveCleanerSetting(
		savingFlag: CleanerSavingFlag,
		save: () => Promise<RobloxCleanerState>,
		fallbackMessage: string,
	): Promise<void> {
		if (this[savingFlag]) return
		this[savingFlag] = true
		await this.#requestCleaner(save, true, fallbackMessage)
		this[savingFlag] = false
	}

	// Cleaner requests run one at a time, in the order they were made, so a
	// refresh never reads the backend while a save is still changing it, and
	// two saves never finish out of order. A response applies only while no
	// newer save was requested.
	#requestCleaner(
		request: () => Promise<RobloxCleanerState>,
		isSave: boolean,
		failureMessage: string,
	): Promise<void> {
		if (isSave) this.#cleanerRevision++
		const revision = this.#cleanerRevision
		const run = async (): Promise<void> => {
			try {
				const state = await request()
				if (revision !== this.#cleanerRevision) return
				this.cleaner = state
				this.cleanerError = ""
			} catch (error) {
				if (revision === this.#cleanerRevision) {
					this.cleanerError = errorMessage(error, failureMessage)
				}
			}
		}
		const queued = this.#cleanerQueue.then(run)
		this.#cleanerQueue = queued
		return queued
	}
}

export const robloxClientOptions = new RobloxClientOptions()
