import { Events } from "@wailsio/runtime"
import {
	accountBackend,
	RobloxCleanScope,
	type RobloxCleanReport,
} from "../backend/bridge"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import { errorMessage } from "../shared/errors"

export type CleanPhase =
	| "closing-processes"
	| "removing-applications"
	| "removing-registry-entries"
	| "deleting-user-content"
	| "deleting-caches"
	| "deleting-logs"

export type CleanPhaseProgress = {
	phase: CleanPhase
	state: "waiting" | "running" | "done"
	completedCount: number
	totalCount: number
	skippedCount: number
}

export type CleanActivity = {
	action: "closed" | "deleted" | "skipped"
	item: string
	skipReason?: "in-use" | "needs-administrator-rights" | "link" | "failed"
}

type CleanProgress = { phases: CleanPhaseProgress[]; activity?: CleanActivity }

export const cleanPhaseText: Record<
	CleanPhase,
	{ title: string; description: string; progressLabel: string }
> = {
	"closing-processes": {
		title: "Close Roblox",
		description:
			"Roblox Player, Roblox Studio, their launchers, and the crash handler",
		progressLabel: "Closing Roblox…",
	},
	"removing-applications": {
		title: "Remove Roblox and Studio",
		description: "Install folders, Start menu folders, and shortcuts",
		progressLabel: "Removing Roblox and Studio…",
	},
	"removing-registry-entries": {
		title: "Remove registry entries",
		description: "File associations, Roblox link handlers, and uninstall entries",
		progressLabel: "Removing registry entries…",
	},
	"deleting-user-content": {
		title: "Delete saved places and media",
		description: "Saved places, screenshots, and videos",
		progressLabel: "Deleting saved places and media…",
	},
	"deleting-caches": {
		title: "Delete caches and sign-in data",
		description: "Caches, cookies, the Studio sign-in, and temporary files",
		progressLabel: "Deleting caches and sign-in data…",
	},
	"deleting-logs": {
		title: "Delete logs",
		description: "Roblox logs, which Logs Explorer reads",
		progressLabel: "Deleting logs…",
	},
}

export const skipReasonText: Record<
	NonNullable<CleanActivity["skipReason"]>,
	string
> = {
	"in-use": "in use",
	"needs-administrator-rights": "needs administrator rights",
	"link": "is a link",
	"failed": "could not be deleted",
}

export function cleanReportSummary(report: RobloxCleanReport): string {
	const deleted = `Deleted ${report.deletedCount} ${report.deletedCount === 1 ? "item" : "items"}.`
	return report.skippedCount > 0
		? `${deleted} ${report.skippedCount} skipped.`
		: deleted
}

export function cleanDoneTitle(scope: RobloxCleanScope): string {
	return scope === RobloxCleanScope.ScopeUninstall
		? "Roblox uninstalled"
		: "Roblox data cleaned"
}

// RobloxCleaningRun holds the one cleaning that can run at a time, so that
// Settings and the clean dialog show the same run. The last run stays until
// the next one starts or it is dismissed.
class RobloxCleaningRun {
	scope = $state<RobloxCleanScope | null>(null)
	phases = $state<CleanPhaseProgress[]>([])
	activities = $state<CleanActivity[]>([])
	report = $state<RobloxCleanReport | null>(null)
	error = $state("")
	isRunning = $state(false)
	durationMilliseconds = $state(0)
	#isListening = false
	#viewerCount = 0

	get completedCount(): number {
		return this.phases.reduce((sum, phase) => sum + phase.completedCount, 0)
	}

	get totalCount(): number {
		return this.phases.reduce((sum, phase) => sum + phase.totalCount, 0)
	}

	get runningPhase(): CleanPhaseProgress | undefined {
		return this.phases.find((phase) => phase.state === "running")
	}

	// view marks a mounted view that shows the run. Without one, the result
	// of a finished run goes to a notification. Returns the function that ends
	// the view.
	view(): () => void {
		this.#viewerCount++
		return () => this.#viewerCount--
	}

	dismiss(): void {
		if (this.isRunning) return
		this.scope = null
		this.phases = []
		this.activities = []
		this.report = null
		this.error = ""
	}

	async start(
		scope: RobloxCleanScope,
		notifications: NotificationCenter,
	): Promise<void> {
		if (this.isRunning) return
		this.#listen()
		this.dismiss()
		this.scope = scope
		this.isRunning = true
		const startedAt = performance.now()
		try {
			const report = await accountBackend.CleanRoblox(scope)
			// A report can arrive before the last progress event, and a finished
			// run has finished every phase.
			this.phases = this.phases.map((phase) => ({
				...phase,
				state: "done",
				completedCount: phase.totalCount,
			}))
			this.report = report
			if (this.#viewerCount === 0) {
				notifications.show({
					id: "roblox-clean",
					tone: report.skippedCount > 0 ? "warning" : "success",
					title: cleanDoneTitle(scope),
					message: cleanReportSummary(report),
				})
			}
		} catch (cause) {
			this.error = errorMessage(cause, "Could not clean Roblox. Try again.")
			if (this.#viewerCount === 0) {
				notifications.show({
					id: "roblox-clean",
					tone: "error",
					title: "Could not finish cleaning Roblox",
					message: this.error,
				})
			}
		} finally {
			this.durationMilliseconds = performance.now() - startedAt
			this.isRunning = false
		}
	}

	#listen(): void {
		if (this.#isListening) return
		this.#isListening = true
		Events.On("roblox:clean-progress", (event) => {
			if (this.scope === null || this.report) return
			const progress = event.data as CleanProgress
			this.phases = progress.phases
			if (progress.activity) this.activities.push(progress.activity)
		})
	}
}

export const robloxCleaning = new RobloxCleaningRun()
