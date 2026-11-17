import { Events } from "@wailsio/runtime"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import { accountBackend, UpdateStatus } from "../backend/bridge"
import type { UpdateState } from "../backend/bridge"
import { coalesce } from "../shared/coalesce"

const notificationId = "application-update"

class UpdateStore {
	state = $state<UpdateState>({
		currentVersion: "",
		downloadedBytes: 0,
		status: UpdateStatus.StatusIdle,
		totalBytes: 0,
	})
	error = $state("")
	#notifications: NotificationCenter | null = null

	mount(notifications: NotificationCenter): () => void {
		this.#notifications = notifications
		const remove = Events.On("update:state-changed", () => void this.refresh())
		void this.refresh()
		return remove
	}

	readonly refresh = coalesce(async () => {
		try {
			const previous = this.state,
				next = await accountBackend.GetUpdateState()
			this.state = next
			this.#notify(previous, next)
		} catch {
			this.error = "Could not load the update status."
		}
	})

	#notify(previous: UpdateState, next: UpdateState): void {
		if (previous.status === next.status) return
		const version = next.availableVersion ?? ""
		if (next.status === UpdateStatus.StatusAvailable && next.error) {
			this.#notifications?.show({
				actions: [
					{
						emphasis: true,
						label: "Try again",
						onClick: () => this.install(),
					},
				],
				id: notificationId,
				message: next.error,
				title: "Update not completed",
				tone: "error",
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
			"Could not check the updates. Try again.",
		)
	}

	async install(): Promise<void> {
		await this.#run(
			() => accountBackend.InstallUpdate(),
			"Could not download the update. Try again.",
		)
	}

	async cancelDownload(): Promise<void> {
		await this.#run(
			() => accountBackend.CancelUpdateDownload(),
			"Could not cancel the download.",
		)
	}

	async restart(): Promise<void> {
		await this.#run(
			() => accountBackend.RestartToUpdate(),
			"Could not restart the app to install the update. Try again.",
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
