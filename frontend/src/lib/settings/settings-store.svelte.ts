import { accountBackend, MotionPreference } from "../backend/bridge"
import type { GameRegion, PresenceScope, PresenceSettings } from "../backend/bridge"
import { setMotionPreference } from "../shared/motion"
import { timestampFormats } from "../shared/timestamp"
import type { NotificationCenter } from "../notifications/notification-center.svelte"

export type LoggingLevel = "trace" | "debug" | "info" | "warn" | "error"

export class SettingsStore {
	presence = $state<PresenceSettings>({
		accounts: { enabled: true, intervalSeconds: 120 },
		profile: { enabled: true, intervalSeconds: 60 },
	})
	busyPresence = $state(false)
	enabledLoggingLevels = $state<Record<LoggingLevel, boolean>>({
		debug: false,
		error: true,
		info: false,
		trace: false,
		warn: true,
	})
	timestampFormat = $state<string>(timestampFormats.shown)
	timestampHoverFormat = $state<string>(timestampFormats.tooltip)
	busyLevel = $state<LoggingLevel | "all" | null>(null)
	busyTimestampFormats = $state(false)
	motion = $state(MotionPreference.MotionSystem)
	busyMotion = $state(false)
	roValraEnabled = $state(true)
	// The server-browser region code used by the launch panel, or empty before one is chosen.
	roValraRegion = $state("")
	busyRoValra = $state(false)
	busyRoValraRegion = $state(false)
	// The RoValra region catalog, loaded at startup and refreshed on request.
	serverRegions = $state<GameRegion[] | null>(null)
	serverRegionsFailed = $state(false)
	loadingServerRegions = $state(false)
	#notifications: NotificationCenter | null = null
	initialized = $state(false)
	error = $state("")

	async initialize(notifications: NotificationCenter): Promise<void> {
		this.#notifications = notifications
		this.error = ""
		try {
			const settings = await accountBackend.GetAppSettings()
			this.presence = settings.presence
			this.enabledLoggingLevels = { ...settings.enabledLoggingLevels } as Record<
				LoggingLevel,
				boolean
			>
			this.timestampFormat = settings.timestampFormat
			this.timestampHoverFormat = settings.timestampHoverFormat
			this.motion = settings.motion
			setMotionPreference(settings.motion)
			this.roValraEnabled = settings.roValraEnabled
			this.roValraRegion = settings.roValraRegion
			if (settings.settingsRecovered) {
				notifications.show({
					id: "settings-recovered",
					title: "Settings reset",
					message:
						"settings.json was corrupted or could not be loaded. A new file was created with default values, and the old file was renamed to ./storage/settings.json.bak.",
					durationMs: 0,
				})
			}
		} catch (error) {
			this.error =
				error instanceof Error ? error.message : "Settings could not be loaded."
		} finally {
			this.initialized = true
		}
		if (this.roValraEnabled) void this.loadServerRegions()
	}

	get preferredRegionLabel(): string {
		return (
			this.serverRegions?.find((region) => region.code === this.roValraRegion)
				?.name ?? this.roValraRegion
		)
	}

	// loadServerRegions refreshes the catalog without replacing a saved region that is no longer listed.
	async loadServerRegions(refresh = false): Promise<void> {
		if (
			!this.roValraEnabled ||
			this.loadingServerRegions ||
			(!refresh && this.serverRegions)
		)
			return
		this.loadingServerRegions = true
		this.serverRegionsFailed = false
		if (refresh) this.error = ""
		try {
			const regions = (await accountBackend.ListServerRegions(refresh)) ?? []
			this.serverRegions = regions
			this.serverRegionsFailed = false
			const code = this.roValraRegion
			if (code && !regions.some((region) => region.code === code)) {
				this.#notifications?.show({
					id: "rovalra-region-missing",
					title: "Preferred region unavailable",
					message:
						"Your saved region is no longer listed by RoValra. Choose another one in Settings under Integrations.",
					durationMs: 0,
				})
			} else {
				this.#notifications?.dismiss("rovalra-region-missing")
			}
		} catch (error) {
			this.serverRegionsFailed = true
			if (refresh) {
				this.error =
					error instanceof Error
						? error.message
						: "Regions could not be refreshed. Try again."
			}
		} finally {
			this.loadingServerRegions = false
		}
	}

	async setPresenceUpdates(
		scope: PresenceScope,
		enabled: boolean,
		intervalSeconds: number,
	): Promise<void> {
		if (this.busyPresence) return
		this.busyPresence = true
		this.error = ""
		try {
			await accountBackend.SetPresenceUpdates(scope, enabled, intervalSeconds)
			this.presence = { ...this.presence, [scope]: { enabled, intervalSeconds } }
		} catch (error) {
			this.error =
				error instanceof Error
					? error.message
					: "Presence settings could not be saved."
		} finally {
			this.busyPresence = false
		}
	}

	async setRoValraEnabled(enabled: boolean): Promise<void> {
		if (this.busyRoValra) return
		this.busyRoValra = true
		this.error = ""
		try {
			await accountBackend.SetRoValraEnabled(enabled)
			this.roValraEnabled = enabled
			if (enabled) void this.loadServerRegions()
		} catch (error) {
			this.error =
				error instanceof Error
					? error.message
					: "The RoValra setting could not be saved."
		} finally {
			this.busyRoValra = false
		}
	}

	async setRoValraRegion(code: string): Promise<void> {
		if (this.busyRoValraRegion || this.loadingServerRegions) return
		this.busyRoValraRegion = true
		this.error = ""
		try {
			await accountBackend.SetRoValraRegion(code)
			this.roValraRegion = code
			this.#notifications?.dismiss("rovalra-region-missing")
		} catch (error) {
			this.error =
				error instanceof Error
					? error.message
					: "The preferred region could not be saved."
		} finally {
			this.busyRoValraRegion = false
		}
	}

	isLoggingLevelEnabled(level: LoggingLevel): boolean {
		return this.enabledLoggingLevels[level]
	}

	areAllLoggingLevelsEnabled(): boolean {
		return Object.values(this.enabledLoggingLevels).every(Boolean)
	}

	async setLoggingLevelEnabled(level: LoggingLevel, enabled: boolean): Promise<void> {
		if (this.busyLevel !== null) {
			return
		}
		const previous = this.enabledLoggingLevels
		this.busyLevel = level
		this.error = ""
		this.enabledLoggingLevels = { ...previous, [level]: enabled }
		try {
			await accountBackend.SetLoggingLevelEnabled(level, enabled)
		} catch (error) {
			this.enabledLoggingLevels = previous
			this.error =
				error instanceof Error
					? error.message
					: "The logging setting could not be saved."
		} finally {
			this.busyLevel = null
		}
	}

	async setAllLoggingLevelsEnabled(enabled: boolean): Promise<void> {
		if (this.busyLevel !== null) {
			return
		}
		const previous = this.enabledLoggingLevels
		this.busyLevel = "all"
		this.error = ""
		this.enabledLoggingLevels = Object.fromEntries(
			Object.keys(previous).map((level) => [level, enabled]),
		) as Record<LoggingLevel, boolean>
		try {
			await accountBackend.SetAllLoggingLevelsEnabled(enabled)
		} catch (error) {
			this.enabledLoggingLevels = previous
			this.error =
				error instanceof Error
					? error.message
					: "The logging settings could not be saved."
		} finally {
			this.busyLevel = null
		}
	}

	async setMotion(motion: MotionPreference): Promise<void> {
		if (this.busyMotion || motion === this.motion) return
		this.busyMotion = true
		this.error = ""
		try {
			await accountBackend.SetMotion(motion)
			this.motion = motion
			setMotionPreference(motion)
		} catch (error) {
			this.error =
				error instanceof Error
					? error.message
					: "The motion setting could not be saved."
		} finally {
			this.busyMotion = false
		}
	}

	async setTimestampFormats(
		timestampFormat: string,
		timestampHoverFormat: string,
	): Promise<boolean> {
		if (this.busyTimestampFormats) {
			return false
		}
		this.busyTimestampFormats = true
		this.error = ""
		try {
			await accountBackend.SetTimestampFormats(
				timestampFormat,
				timestampHoverFormat,
			)
			this.timestampFormat = timestampFormat
			this.timestampHoverFormat = timestampHoverFormat
			return true
		} catch (error) {
			this.error =
				error instanceof Error
					? error.message
					: "The timestamp formats could not be saved."
			return false
		} finally {
			this.busyTimestampFormats = false
		}
	}
}

export const appSettings = new SettingsStore()
