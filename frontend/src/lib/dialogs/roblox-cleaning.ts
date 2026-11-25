// The progress that the backend sends with the "roblox:clean-progress" event.
type CleanPhase =
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

export type CleanProgress = { phases: CleanPhaseProgress[]; activity?: CleanActivity }

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
