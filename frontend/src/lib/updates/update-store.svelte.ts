import { Events } from "@wailsio/runtime"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import { accountBackend, UpdateStatus } from "../backend/bridge"
import type { UpdateState } from "../backend/bridge"

const notificationId = "application-update"

export class UpdateStore {
	state = $state<UpdateState>({
		currentVersion: APP_VERSION,
		downloadedBytes: 0,
		status: UpdateStatus.StatusIdle,
		totalBytes: 0,
	})
	error = $state("")
	#notifications: NotificationCenter | null = null
	#reload: Promise<void> | null = null
	#queued = false

	mount(notifications: NotificationCenter): () => void {
		this.#notifications = notifications
		const remove = Events.On("update:state-changed", () => void this.refresh())
		void this.refresh()
		return remove
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
			const previous = this.state,
				next = await accountBackend.GetUpdateState()
			this.state = next
			this.#notify(previous, next)
		} catch {
			this.error = "The update status could not be loaded."
		}
		if (this.#queued) await this.#refreshQueued()
	}

	#notify(previous: UpdateState, next: UpdateState): void {
		if (previous.status === next.status) return
		const version = next.availableVersion ?? ""
		if (next.status === UpdateStatus.StatusAvailable && next.error) {
			this.#notifications?.show({
				actions: [
					{ emphasis: true, label: "Retry", onClick: () => this.install() },
				],
				id: notificationId,
				message: next.error,
				title: "Update failed",
			})
		} else if (next.status === UpdateStatus.StatusAvailable) {
			this.#notifications?.show({
				actions: [
					{
						emphasis: true,
						label: "Download",
						onClick: () => this.install(),
					},
				],
				id: notificationId,
				message: `Version ${version} is available.`,
				title: "Update available",
			})
		} else if (next.status === UpdateStatus.StatusReady) {
			this.#notifications?.show({
				actions: [
					{ emphasis: true, label: "Restart", onClick: () => this.restart() },
				],
				id: notificationId,
				message: `Restart to install version ${version}.`,
				title: "Update ready",
			})
		} else {
			this.#notifications?.dismiss(notificationId)
		}
	}

	async check(): Promise<void> {
		await this.#run(
			() => accountBackend.CheckForUpdate(),
			"Updates could not be checked. Try again.",
		)
	}

	async install(): Promise<void> {
		await this.#run(
			() => accountBackend.InstallUpdate(),
			"The update could not be downloaded. Try again.",
		)
	}

	async cancelDownload(): Promise<void> {
		await this.#run(
			() => accountBackend.CancelUpdateDownload(),
			"The download could not be canceled.",
		)
	}

	async restart(): Promise<void> {
		await this.#run(
			() => accountBackend.RestartToUpdate(),
			"The application could not restart to install the update. Try again.",
		)
	}

	async #run(action: () => Promise<void>, failure: string): Promise<void> {
		this.error = ""
		try {
			await action()
		} catch {
			this.error = failure
		}
		await this.refresh()
	}
}

export const updateStore = new UpdateStore()
