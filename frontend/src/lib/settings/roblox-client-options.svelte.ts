import {
	accountBackend,
	type MultiInstanceSnapshot,
	type RobloxCleanerState,
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

	get multiInstanceStatus(): { text: string; tone: StatusTone } {
		const snapshot = this.multiInstance
		if (this.isLoadingMultiInstance) return { text: "Checking…", tone: "neutral" }
		if (this.isSavingMultiInstance) return { text: "Saving…", tone: "neutral" }
		if (!snapshot) return { text: "Not checked", tone: "neutral" }
		if (!snapshot.supported)
			return { text: "Unavailable - Windows only", tone: "neutral" }
		if (!snapshot.enabled) return { text: "Off", tone: "neutral" }
		if (snapshot.ready) return { text: "Ready", tone: "success" }
		return { text: `Not ready - ${snapshot.message}`, tone: "warning" }
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
				"The multi-instance status could not be checked. Try again."
		} finally {
			this.#isCheckingMultiInstance = false
			if (requestRevision === this.#multiInstanceRevision) {
				this.isLoadingMultiInstance = false
			}
		}
	}

	async refreshCleaner(): Promise<void> {
		const requestRevision = this.#cleanerRevision
		try {
			const state = await accountBackend.GetRobloxCleanerState()
			if (requestRevision !== this.#cleanerRevision) return
			this.cleaner = state
			this.cleanerError = ""
		} catch {
			if (requestRevision !== this.#cleanerRevision) return
			this.cleanerError = "The cleaning settings could not be loaded. Try again."
		}
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
				"The multi-instance setting could not be saved. Try again.",
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
			"The cleaning setting could not be saved. Try again.",
		)
	}

	async setDeleteLogsWhenCleaning(enabled: boolean): Promise<void> {
		if (!this.cleaner?.supported) return
		await this.#saveCleanerSetting(
			"isSavingDeleteLogs",
			() => accountBackend.SetDeleteLogsWhenCleaning(enabled),
			"The log cleaning setting could not be saved. Try again.",
		)
	}

	async setUseAdministratorRights(enabled: boolean): Promise<void> {
		if (!this.cleaner?.elevationSupported) return
		await this.#saveCleanerSetting(
			"isSavingAdministratorRights",
			() => accountBackend.SetUseAdministratorRights(enabled),
			"The administrator rights setting could not be saved. Try again.",
		)
	}

	async #saveCleanerSetting(
		savingFlag: CleanerSavingFlag,
		save: () => Promise<RobloxCleanerState>,
		fallbackMessage: string,
	): Promise<void> {
		if (this[savingFlag]) return
		this.#cleanerRevision++
		this[savingFlag] = true
		this.cleanerError = ""
		try {
			this.cleaner = await save()
		} catch (error) {
			this.cleanerError = errorMessage(error, fallbackMessage)
		} finally {
			this[savingFlag] = false
		}
	}
}

export const robloxClientOptions = new RobloxClientOptions()
