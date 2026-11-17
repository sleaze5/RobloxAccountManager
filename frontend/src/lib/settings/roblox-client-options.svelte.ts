import {
	accountBackend,
	type MultiInstanceSnapshot,
	type RobloxCleanerState,
	type RobloxCleanReport,
} from "../backend/bridge"
import { errorMessage } from "../shared/errors"

type StatusTone = "neutral" | "success" | "warning"

type CleanerSavingFlag =
	| "isSavingCleanBeforeLaunch"
	| "isSavingDeleteLogs"
	| "isSavingAdministratorRights"

// The multi-instance status changes while Roblox processes start and stop.
const multiInstancePollIntervalMilliseconds = 500

// Client options are shared by the Client options menu in the launch panel
// and Settings > Roblox, so a change in one place shows in the other.
export function cleanReportSummary(report: RobloxCleanReport): string {
	const deleted = `Deleted ${report.deletedCount} ${report.deletedCount === 1 ? "item" : "items"}.`
	return report.skippedCount > 0
		? `${deleted} ${report.skippedCount} skipped.`
		: deleted
}

class RobloxClientOptions {
	multiInstance = $state<MultiInstanceSnapshot | null>(null)
	cleaner = $state<RobloxCleanerState | null>(null)
	isLoadingMultiInstance = $state(true)
	isSavingMultiInstance = $state(false)
	isSavingCleanBeforeLaunch = $state(false)
	isSavingDeleteLogs = $state(false)
	isSavingAdministratorRights = $state(false)
	multiInstanceError = $state("")
	multiInstanceStatusError = $state("")
	cleanerError = $state("")
	#isCheckingMultiInstance = false
	#multiInstanceRevision = 0
	#cleanerRevision = 0
	#cleanerQueue: Promise<void> = Promise.resolve()

	get multiInstanceStatus(): { text: string; tone: StatusTone } {
		const snapshot = this.multiInstance
		if (this.isLoadingMultiInstance) return { text: "Checking…", tone: "neutral" }
		if (this.isSavingMultiInstance) return { text: "Saving…", tone: "neutral" }
		if (!snapshot) return { text: "Not checked", tone: "neutral" }
		if (!snapshot.supported) return { text: "Windows only", tone: "neutral" }
		if (!snapshot.enabled) return { text: "Off", tone: "neutral" }
		if (snapshot.ready) return { text: "Ready", tone: "success" }
		return { text: `Not ready: ${snapshot.message}`, tone: "warning" }
	}

	get isCleanBeforeLaunchActive(): boolean {
		return Boolean(
			this.cleaner?.cleanBeforeLaunch && !this.cleaner.isBlockedByMultiInstance,
		)
	}

	async load(): Promise<void> {
		await Promise.all([this.refreshMultiInstance(), this.refreshCleaner()])
	}

	// Returns the function that stops polling.
	pollMultiInstanceWhileEnabled(): () => void {
		const interval = setInterval(() => {
			if (this.multiInstance?.enabled) void this.refreshMultiInstance()
		}, multiInstancePollIntervalMilliseconds)
		return () => clearInterval(interval)
	}

	async refreshMultiInstance(): Promise<void> {
		if (this.#isCheckingMultiInstance || this.isSavingMultiInstance) return
		this.#isCheckingMultiInstance = true
		this.isLoadingMultiInstance = this.multiInstance === null
		const requestRevision = this.#multiInstanceRevision
		try {
			const state = await accountBackend.GetMultiInstanceState()
			if (requestRevision !== this.#multiInstanceRevision) return
			this.multiInstance = state
			this.multiInstanceStatusError = ""
		} catch {
			if (requestRevision !== this.#multiInstanceRevision) return
			this.multiInstanceStatusError =
				"Could not check the multi-instance status. Try again."
		} finally {
			this.#isCheckingMultiInstance = false
			if (requestRevision === this.#multiInstanceRevision) {
				this.isLoadingMultiInstance = false
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

	async setMultiInstanceEnabled(enabled: boolean): Promise<void> {
		if (
			this.isSavingMultiInstance ||
			this.isLoadingMultiInstance ||
			!this.multiInstance?.supported
		) {
			return
		}
		const previous = this.multiInstance
		this.#multiInstanceRevision++
		this.isSavingMultiInstance = true
		this.multiInstanceError = ""
		try {
			this.multiInstance = await accountBackend.SetMultiInstanceEnabled(enabled)
			this.multiInstanceStatusError = ""
		} catch (error) {
			this.multiInstance = previous
			this.multiInstanceError = errorMessage(
				error,
				"Could not save the multi-instance setting. Try again.",
			)
		} finally {
			this.isSavingMultiInstance = false
		}
		// Multi-instance decides whether Roblox is cleaned before a launch.
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
