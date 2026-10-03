import { Events } from "@wailsio/runtime"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import { accountBackend, Lifecycle, Mode, RuntimeStatus } from "../backend/bridge"
import type {
	BrowserSnapshot,
	CandidateState,
	SaveResult,
	ShutdownEffects,
} from "../backend/bridge"

const emptySnapshot = {
	candidates: [],
	cleanup: [],
	revision: 0,
	runtime: {
		approximateBytes: 0,
		canCancel: false,
		downloadedBytes: 0,
		downloadTotalBytes: 0,
		downloadBytesPerSecond: 0,
		downloadEtaSeconds: null,
		installedBytes: 0,
		requiredVersion: "",
		status: RuntimeStatus.RuntimeMissing,
	},
	sessions: [],
} as BrowserSnapshot

export class BrowserStore {
	snapshot = $state<BrowserSnapshot>(emptySnapshot)
	loading = $state(false)
	error = $state("")
	#reload: Promise<void> | null = null
	#queued = false
	#notifications: NotificationCenter | null = null
	#cleanupNotified = false
	#detachmentNotified = new Set<string>()
	#runtimeDamageNotified = false
	#runtimeDownloadPending = false
	#runtimeDownloadReplacing = false
	#runtimeViewCount = 0

	get runtimeReady(): boolean {
		return this.snapshot.runtime.status === RuntimeStatus.RuntimeReady
	}

	get runtimeDownloadActive(): boolean {
		return (
			this.#runtimeDownloadPending ||
			this.snapshot.runtime.status === RuntimeStatus.RuntimeDownloading ||
			this.snapshot.runtime.status === RuntimeStatus.RuntimeInstalling
		)
	}

	get loginSessions() {
		return (this.snapshot.sessions ?? []).filter(
			(session) => session.mode === Mode.ModeLogin,
		)
	}

	get activeSessions() {
		return (this.snapshot.sessions ?? []).filter(
			(session) =>
				session.lifecycle !== Lifecycle.LifecycleClosed &&
				session.lifecycle !== Lifecycle.LifecycleFailed,
		)
	}

	get browserSessions() {
		return this.snapshot.sessions ?? []
	}

	get activeLoginSessions() {
		return this.loginSessions.filter(
			(session) =>
				session.lifecycle !== Lifecycle.LifecycleClosed &&
				session.lifecycle !== Lifecycle.LifecycleFailed,
		)
	}

	get candidates(): CandidateState[] {
		return this.snapshot.candidates ?? []
	}

	trackRuntimeView(): () => void {
		this.#runtimeViewCount++
		return () => {
			this.#runtimeViewCount--
		}
	}

	mount(notifications: NotificationCenter): () => void {
		this.#notifications = notifications
		const remove = Events.On("browser:state-changed", () => void this.refresh())
		const removeLaunchFailure = Events.On("browser:launch-failed", (event) => {
			const { sessionId, message } = event.data as {
				sessionId: string
				message: string
			}
			notifications.show({
				id: `browser-launch-${sessionId}`,
				title: "Game launch failed",
				message,
			})
		})
		void this.refresh()
		return () => {
			remove()
			removeLaunchFailure()
		}
	}

	async refresh(): Promise<void> {
		if (this.#reload) {
			this.#queued = true
			return this.#reload
		}
		this.#reload = this.#refreshQueued()
		try {
			await this.#reload
		} finally {
			this.#reload = null
		}
	}

	async #refreshQueued(): Promise<void> {
		this.#queued = false
		try {
			const next = await accountBackend.GetBrowserState()
			this.snapshot = next
			this.#updateNotifications(next)
		} catch (error) {
			this.error = errorMessage(error)
		}
		if (this.#queued) {
			await this.#refreshQueued()
		}
	}

	#updateNotifications(next: BrowserSnapshot): void {
		this.#updateRuntimeDownloadNotification(next)
		this.#updateDetachmentNotifications(next)
		this.#updateRuntimeNotification(next)
		this.#updateCleanupNotification(next)
	}

	#updateRuntimeDownloadNotification(next: BrowserSnapshot): void {
		const active =
			next.runtime.status === RuntimeStatus.RuntimeDownloading ||
			next.runtime.status === RuntimeStatus.RuntimeInstalling
		if (active) {
			this.#runtimeDownloadPending = true
			return
		}
		if (!this.#runtimeDownloadPending) return
		const replacing = this.#runtimeDownloadReplacing
		this.#runtimeDownloadPending = false
		this.#runtimeDownloadReplacing = false
		if (this.#runtimeViewCount > 0) return
		if (next.runtime.status === RuntimeStatus.RuntimeReady && !next.runtime.error) {
			this.#notifications?.show({
				id: "browser-runtime-download",
				message: "Chrome for Testing is installed and ready to use.",
				title: "Browser ready",
			})
		} else if (
			next.runtime.error &&
			next.runtime.status !== RuntimeStatus.RuntimeDamaged
		) {
			this.#notifications?.show({
				actions: [
					{
						label: "Retry",
						onClick: () => this.download(replacing),
					},
				],
				id: "browser-runtime-download",
				message: next.runtime.error,
				title: "Browser download failed",
			})
		}
	}

	#updateDetachmentNotifications(next: BrowserSnapshot): void {
		for (const session of next.sessions ?? []) {
			if (
				session.mode === Mode.ModeSaved &&
				session.lifecycle !== Lifecycle.LifecycleFailed &&
				!session.attached &&
				session.error &&
				!this.#detachmentNotified.has(session.id)
			) {
				this.#detachmentNotified.add(session.id)
				this.#notifications?.show({
					actions: [
						{
							label: "Close browser",
							onClick: () => this.close(session.id),
						},
					],
					id: `browser-detached-${session.id}`,
					message: session.error,
					title: "Account browser detached",
				})
			}
		}
	}

	#updateRuntimeNotification(next: BrowserSnapshot): void {
		if (
			next.runtime.status === RuntimeStatus.RuntimeDamaged &&
			!this.#runtimeDamageNotified
		) {
			this.#runtimeDamageNotified = true
			this.#notifications?.show({
				actions: [{ label: "Redownload", onClick: () => this.download(true) }],
				id: "browser-runtime-damaged",
				message: "The installed browser is invalid and must be redownloaded.",
				title: "Browser needs repair",
			})
		} else if (next.runtime.status !== RuntimeStatus.RuntimeDamaged) {
			this.#runtimeDamageNotified = false
			this.#notifications?.dismiss("browser-runtime-damaged")
		}
	}

	#updateCleanupNotification(next: BrowserSnapshot): void {
		if ((next.cleanup ?? []).length > 0 && !this.#cleanupNotified) {
			this.#cleanupNotified = true
			this.#notifications?.show({
				actions: [
					{ label: "Retry cleanup", onClick: () => this.retryCleanup() },
				],
				id: "browser-cleanup",
				message: "A temporary browser profile could not be removed.",
				title: "Browser cleanup needed",
			})
		} else if ((next.cleanup ?? []).length === 0) {
			this.#cleanupNotified = false
			this.#notifications?.dismiss("browser-cleanup")
		}
	}

	async startLogin(): Promise<boolean> {
		this.loading = true
		this.error = ""
		try {
			await accountBackend.StartLoginBrowser()
			await this.refresh()
			return true
		} catch (error) {
			this.error = errorMessage(error)
			return false
		} finally {
			this.loading = false
		}
	}

	async openAccount(accountId: number): Promise<void> {
		this.error = ""
		try {
			await accountBackend.OpenAccountBrowser(accountId)
			await this.refresh()
		} catch (error) {
			this.#notifications?.show({
				id: `browser-account-${accountId}`,
				message: errorMessage(error),
				title: "Browser could not open",
			})
		}
	}

	hasAccountSession(accountId: number): boolean {
		return (this.snapshot.sessions ?? []).some(
			(session) =>
				session.mode === Mode.ModeSaved &&
				session.accountId === accountId &&
				session.lifecycle !== Lifecycle.LifecycleClosed &&
				session.lifecycle !== Lifecycle.LifecycleFailed,
		)
	}

	accountSessionId(accountId: number): string {
		return (
			(this.snapshot.sessions ?? []).find(
				(session) =>
					session.mode === Mode.ModeSaved &&
					session.accountId === accountId &&
					session.lifecycle !== Lifecycle.LifecycleFailed,
			)?.id ?? ""
		)
	}

	async focus(id: string): Promise<void> {
		try {
			await accountBackend.FocusBrowserSession(id)
		} catch (error) {
			this.#notifications?.show({
				id: `browser-focus-${id}`,
				message: errorMessage(error),
				title: "Browser could not be focused",
			})
		}
	}

	async close(id: string): Promise<void> {
		try {
			await accountBackend.CloseBrowserSession(id)
			await this.refresh()
		} catch (error) {
			this.error = errorMessage(error)
		}
	}

	async closeAllLogin(): Promise<void> {
		await accountBackend.CloseAllLoginBrowsers()
		await this.refresh()
	}

	async save(selected: string[]): Promise<SaveResult | null> {
		this.loading = true
		try {
			const result = await accountBackend.SaveBrowserCandidates(selected)
			await this.refresh()
			return result
		} catch (error) {
			this.error = errorMessage(error)
			return null
		} finally {
			this.loading = false
		}
	}

	async discard(): Promise<void> {
		await accountBackend.DiscardBrowserCandidates()
		await this.refresh()
	}

	async download(redownload = false): Promise<void> {
		this.error = ""
		this.#runtimeDownloadPending = true
		this.#runtimeDownloadReplacing = redownload
		this.#notifications?.dismiss("browser-runtime-download")
		try {
			if (redownload) await accountBackend.RedownloadBrowserRuntime()
			else await accountBackend.DownloadBrowserRuntime()
			await this.refresh()
		} catch (error) {
			this.#runtimeDownloadPending = false
			this.#runtimeDownloadReplacing = false
			this.error = errorMessage(error)
		}
	}

	async cancelDownload(): Promise<void> {
		await accountBackend.CancelBrowserRuntimeDownload()
	}

	async removeRuntime(): Promise<void> {
		try {
			await accountBackend.RemoveBrowserRuntime()
			await this.refresh()
		} catch (error) {
			this.error = errorMessage(error)
		}
	}

	async effects(): Promise<ShutdownEffects> {
		return accountBackend.GetBrowserShutdownEffects()
	}

	async retryCleanup(): Promise<void> {
		await accountBackend.RetryBrowserCleanup()
		await this.refresh()
	}
}

export const browserStore = new BrowserStore()

function errorMessage(error: unknown): string {
	return error instanceof Error ? error.message : "The browser operation failed."
}
