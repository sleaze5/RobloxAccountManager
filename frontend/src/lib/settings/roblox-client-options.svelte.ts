import {
	accountBackend,
	type MultiInstanceSnapshot,
	type RobloxCleanerState,
} from "../backend/bridge"
import { errorMessage } from "../shared/errors"

type StatusTone = "neutral" | "success" | "warning"

// Client options are shared by the Client options menu in the launch panel
// and Settings > Roblox, so a change in one place shows in the other.
export class RobloxClientOptions {
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
		try {
			this.cleaner = await accountBackend.GetRobloxCleanerState()
			this.cleanerError = ""
		} catch {
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
		if (this.isSavingCleanBeforeLaunch || !this.cleaner?.supported) return
		this.isSavingCleanBeforeLaunch = true
		this.cleanerError = ""
		try {
			this.cleaner = await accountBackend.SetCleanBeforeLaunch(enabled)
		} catch (error) {
			this.cleanerError = errorMessage(
				error,
				"The cleaning setting could not be saved. Try again.",
			)
		} finally {
			this.isSavingCleanBeforeLaunch = false
		}
	}

	async setDeleteLogsWhenCleaning(enabled: boolean): Promise<void> {
		if (this.isSavingDeleteLogs || !this.cleaner?.supported) return
		this.isSavingDeleteLogs = true
		this.cleanerError = ""
		try {
			this.cleaner = await accountBackend.SetDeleteLogsWhenCleaning(enabled)
		} catch (error) {
			this.cleanerError = errorMessage(
				error,
				"The log cleaning setting could not be saved. Try again.",
			)
		} finally {
			this.isSavingDeleteLogs = false
		}
	}

	async setUseAdministratorRights(enabled: boolean): Promise<void> {
		if (this.isSavingAdministratorRights || !this.cleaner?.elevationSupported)
			return
		this.isSavingAdministratorRights = true
		this.cleanerError = ""
		try {
			this.cleaner = await accountBackend.SetUseAdministratorRights(enabled)
		} catch (error) {
			this.cleanerError = errorMessage(
				error,
				"The administrator rights setting could not be saved. Try again.",
			)
		} finally {
			this.isSavingAdministratorRights = false
		}
	}
}

export const robloxClientOptions = new RobloxClientOptions()
