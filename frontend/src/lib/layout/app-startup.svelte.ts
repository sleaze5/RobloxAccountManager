import { accountBackend } from "../backend/bridge"
import type { AppBuild, AppLocation } from "../backend/bridge"
import { notifyLoadFailure } from "../notifications/error-notifications.svelte"
import type { NotificationCenter } from "../notifications/notification-center.svelte"

// AppStartup reads what the app checks before the vault opens: the build,
// which warns about a development build, and the data folder, which the user
// confirms on the first run and which reports a move from the last run.
export class AppStartup {
	build = $state<AppBuild | null>(null)
	developmentWarning = $state<"loading" | "open" | "closed">("loading")
	appLocation = $state<AppLocation | null>(null)
	readonly #notifications: NotificationCenter

	constructor(notifications: NotificationCenter) {
		this.#notifications = notifications
	}

	load(): void {
		void this.#loadBuild()
		void this.#loadAppLocation()
	}

	closeDevelopmentWarning(): void {
		this.developmentWarning = "closed"
	}

	confirmAppLocation(): void {
		if (this.appLocation)
			this.appLocation = { ...this.appLocation, initialized: true }
	}

	async #loadBuild(): Promise<void> {
		try {
			this.build = await accountBackend.GetBuild()
		} catch (error) {
			notifyLoadFailure(
				this.#notifications,
				"build-load",
				"Build details unavailable",
				error,
			)
		} finally {
			this.developmentWarning =
				this.build && !this.build.release ? "open" : "closed"
		}
	}

	async #loadAppLocation(): Promise<void> {
		let location: AppLocation
		try {
			location = await accountBackend.GetAppLocation()
		} catch (error) {
			notifyLoadFailure(
				this.#notifications,
				"location-load",
				"Data folder unavailable",
				error,
			)
			return
		}
		this.appLocation = location
		if (location.moveError) {
			this.#notifications.show({
				title: location.moved ? "Old data not removed" : "Data not moved",
				tone: location.moved ? "warning" : "error",
				message: location.moveError,
				durationMs: 0,
			})
		} else if (location.moved) {
			this.#notifications.show({
				title: "Data moved",
				tone: "success",
				message: "The vault, backups, and settings are in the new folder.",
			})
		}
	}
}
