import { accountBackend } from "../backend/bridge"
import type { GameServerVisit, NearbyServerRegion } from "../backend/bridge"
import { errorMessage } from "../shared/errors"

export type RegionCheckStatus =
	| "waiting"
	| "checking"
	| "found"
	| "empty"
	| "skipped"
	| "failed"

export type ServerCheckStatus =
	| "checking"
	| "open"
	| "closed"
	| "found"
	| "skipped"
	| "failed"

export interface ServerCheck {
	jobId: string
	status: ServerCheckStatus
}

export interface RegionCheck {
	nearbyRegion: NearbyServerRegion
	status: RegionCheckStatus
	servers: ServerCheck[]
	listedJobIds: string[]
	hasSkippedServer: boolean
}

export type NearestServerSearchPhase =
	| "loading"
	| "checking"
	| "found"
	| "exhausted"
	| "failed"

const searchError = "The server search failed. Try again."

export class NearestServerSearch {
	regionChecks = $state<RegionCheck[]>([])
	phase = $state<NearestServerSearchPhase>("loading")
	activeRegionIndex = $state(0)
	foundJobId = $state("")
	foundServerVisit = $state<GameServerVisit | null>(null)
	error = $state("")

	#requests: { cancel: () => unknown }[] = []
	#visitRequest: { cancel: () => unknown } | null = null
	#version = 0
	#hasLoadedRegions = false

	constructor(
		private readonly placeId: number,
		readonly accountId: number,
	) {}

	async start(): Promise<void> {
		const version = this.#beginRequests()
		this.phase = "loading"
		this.error = ""
		this.foundJobId = ""
		this.regionChecks = []
		this.activeRegionIndex = 0
		this.#hasLoadedRegions = false
		const request = accountBackend.ListNearbyServerRegions(this.placeId)
		this.#requests = [request]
		try {
			const nearbyRegions = await request
			if (version !== this.#version) return
			this.regionChecks = (nearbyRegions ?? []).map((nearbyRegion) => ({
				nearbyRegion,
				status: "waiting",
				servers: [],
				listedJobIds: [],
				hasSkippedServer: false,
			}))
			this.#hasLoadedRegions = true
		} catch (error) {
			if (version !== this.#version) return
			this.error = errorMessage(error, searchError)
			this.phase = "failed"
			return
		}
		await this.#checkRegion(0, version)
	}

	async skipFoundServer(): Promise<void> {
		const check = this.regionChecks[this.activeRegionIndex],
			foundServer = check?.servers.find((server) => server.status === "found")
		if (this.phase !== "found" || !check || !foundServer) return
		foundServer.status = "skipped"
		check.hasSkippedServer = true
		this.foundJobId = ""
		if (this.#chooseOpenServer(check)) return
		await this.#checkRegion(this.activeRegionIndex, this.#beginRequests())
	}

	async retry(): Promise<void> {
		if (this.phase !== "failed") return
		if (this.#hasLoadedRegions) {
			await this.#checkRegion(this.activeRegionIndex, this.#beginRequests())
		} else {
			await this.start()
		}
	}

	dispose(): void {
		this.#beginRequests()
		void this.#visitRequest?.cancel()
		this.#visitRequest = null
	}

	#beginRequests(): number {
		for (const request of this.#requests) void request.cancel()
		this.#requests = []
		return ++this.#version
	}

	#chooseOpenServer(check: RegionCheck): boolean {
		const openServer = check.servers.find((server) => server.status === "open")
		if (!openServer) return false
		openServer.status = "found"
		check.status = "found"
		this.foundJobId = openServer.jobId
		this.phase = "found"
		void this.#findServerVisit(openServer.jobId)
		return true
	}

	async #findServerVisit(jobId: string): Promise<void> {
		void this.#visitRequest?.cancel()
		this.foundServerVisit = null
		const request = accountBackend.FindGameServerVisit(this.placeId, jobId)
		this.#visitRequest = request
		try {
			const visit = await request
			if (this.phase === "found" && this.foundJobId === jobId) {
				this.foundServerVisit = visit
			}
		} catch {
			// The visit badge is extra information, so a search whose Roblox
			// logs cannot be read only shows no badge.
		}
	}

	// Regions are checked one at a time, nearest first, so that the search
	// stops at the first region with an open server. The servers of a region
	// are checked together as one batch.
	async #checkRegion(index: number, version: number): Promise<void> {
		const check = this.regionChecks.at(index)
		if (!check) {
			this.phase = "exhausted"
			return
		}
		this.activeRegionIndex = index
		this.error = ""
		check.status = "checking"
		check.servers = []
		this.phase = "checking"
		const listRequest = accountBackend.ListRegionServers({
			placeId: this.placeId,
			region: check.nearbyRegion.region.code,
			excludedJobIds: [...check.listedJobIds],
		})
		this.#requests = [listRequest]
		let jobIds: string[]
		try {
			jobIds = (await listRequest) ?? []
		} catch (error) {
			if (version !== this.#version) return
			this.#failRegion(check, error)
			return
		}
		if (version !== this.#version) return
		if (jobIds.length === 0) {
			check.status = check.hasSkippedServer ? "skipped" : "empty"
			await this.#checkRegion(index + 1, version)
			return
		}
		// Without a selected account no server can be join-checked, so the
		// first listed server is used as it is.
		if (this.accountId === 0) {
			check.listedJobIds.push(jobIds[0])
			check.servers = [{ jobId: jobIds[0], status: "open" }]
			this.#chooseOpenServer(check)
			return
		}
		check.listedJobIds.push(...jobIds)
		check.servers = jobIds.map((jobId) => ({ jobId, status: "checking" }))
		const servers = check.servers,
			checkRequests = jobIds.map((jobId) =>
				accountBackend.CheckGameServerJoinable(
					this.placeId,
					this.accountId,
					jobId,
				),
			)
		this.#requests = checkRequests
		const errors = await Promise.all(
			checkRequests.map((request, serverIndex) =>
				request.then(
					(isJoinable) => {
						if (version === this.#version) {
							servers[serverIndex].status = isJoinable ? "open" : "closed"
						}
						return null
					},
					(error: unknown) => {
						if (version === this.#version)
							servers[serverIndex].status = "failed"
						return error
					},
				),
			),
		)
		if (version !== this.#version) return
		this.#requests = []
		if (this.#chooseOpenServer(check)) return
		if (servers.some((server) => server.status === "failed")) {
			// A failed server is listed again on retry instead of being skipped.
			check.listedJobIds = check.listedJobIds.filter(
				(jobId) =>
					!servers.some(
						(server) =>
							server.jobId === jobId && server.status === "failed",
					),
			)
			this.#failRegion(
				check,
				errors.find((error) => error !== null),
			)
			return
		}
		check.status = check.hasSkippedServer ? "skipped" : "empty"
		await this.#checkRegion(index + 1, version)
	}

	#failRegion(check: RegionCheck, error: unknown): void {
		check.status = "failed"
		this.error = errorMessage(error, searchError)
		this.phase = "failed"
	}
}
