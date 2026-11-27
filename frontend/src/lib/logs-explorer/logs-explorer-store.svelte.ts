import { SvelteSet } from "svelte/reactivity"
import { accountBackend } from "../backend/bridge"
import type { LogsExplorerSession } from "../backend/bridge"
import { errorMessage } from "../shared/errors"
import { matchesSearch, searchTerms } from "../shared/search"
import { pageSidebarMinimumWidth } from "../layout/sidebar-widths"

export class LogsExplorerStore {
	sessions = $state<LogsExplorerSession[]>([])
	selectedFile = $state("")
	query = $state("")
	sidebarCollapsed = $state(false)
	sidebarWidth = $state(pageSidebarMinimumWidth)
	timelineNewestFirst = $state(true)
	initialized = $state(false)
	refreshing = $state(false)
	readonly refreshingFiles = new SvelteSet<string>()
	loaded = $state(false)
	error = $state("")

	#request = 0
	readonly #appliedRequests = new Map<string, number>()

	get filteredSessions(): LogsExplorerSession[] {
		const terms = searchTerms(this.query)
		if (terms.length === 0) return this.sessions
		return this.sessions.filter((session) =>
			matchesSearch(
				terms,
				[
					session.fileName,
					session.userId,
					session.version,
					session.channel,
					...(session.visits ?? []).flatMap((visit) => [
						visit.placeId,
						visit.serverId,
						visit.universeId,
						visit.datacenterId,
					]),
				].join(" "),
			),
		)
	}

	get selectedSession(): LogsExplorerSession | undefined {
		return this.filteredSessions.find(
			(session) => session.fileName === this.selectedFile,
		)
	}

	initialize(): void {
		if (this.initialized) return
		this.initialized = true
		void this.refreshAll()
	}

	async refreshAll(): Promise<void> {
		if (this.refreshing) return
		const request = ++this.#request
		this.refreshing = true
		this.error = ""
		try {
			const snapshot = await accountBackend.GetLogsExplorer()
			const current = new Map(
				this.sessions.map((session) => [session.fileName, session]),
			)
			this.sessions = (snapshot.sessions ?? []).map((session) => {
				if ((this.#appliedRequests.get(session.fileName) ?? 0) > request) {
					return current.get(session.fileName) ?? session
				}
				this.#appliedRequests.set(session.fileName, request)
				return session
			})
			this.loaded = true
			if (!this.selectedSession) {
				this.selectedFile = this.filteredSessions[0]?.fileName ?? ""
			}
		} catch (error) {
			this.error = errorMessage(
				error,
				"Could not load the logs. Refresh to try again.",
			)
		} finally {
			this.refreshing = false
		}
	}

	async refreshSession(fileName: string): Promise<void> {
		if (this.refreshingFiles.has(fileName)) return
		const request = ++this.#request
		this.refreshingFiles.add(fileName)
		this.error = ""
		try {
			const session = await accountBackend.RefreshLogsExplorerLog(fileName)
			if ((this.#appliedRequests.get(fileName) ?? 0) > request) return
			this.#appliedRequests.set(fileName, request)
			this.sessions = this.sessions.map((current) =>
				current.fileName === fileName ? session : current,
			)
		} catch (error) {
			this.error = errorMessage(
				error,
				"Could not refresh this launch session. Refresh all logs to try again.",
			)
		} finally {
			this.refreshingFiles.delete(fileName)
		}
	}
}
