import { accountBackend } from "../backend/bridge"
import type { NearbyServerRegion } from "../backend/bridge"
import { errorMessage } from "../shared/errors"

export type RegionCheckStatus =
	| "waiting"
	| "checking"
	| "found"
	| "empty"
	| "skipped"
	| "failed"

export interface RegionCheck {
	nearbyRegion: NearbyServerRegion
	status: RegionCheckStatus
	skippedJobIds: string[]
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
	error = $state("")

	#request: { cancel: () => unknown } | null = null
	#version = 0
	#hasLoadedRegions = false

	constructor(
		private readonly placeId: number,
		private readonly accountId: number,
	) {}

	get checkedRegionCount(): number {
		return this.regionChecks.filter(
			(check) => check.status !== "waiting" && check.status !== "checking",
		).length
	}

	async start(): Promise<void> {
		const version = ++this.#version
		this.phase = "loading"
		this.error = ""
		this.foundJobId = ""
		this.regionChecks = []
		this.activeRegionIndex = 0
		this.#hasLoadedRegions = false
		const request = accountBackend.ListNearbyServerRegions(this.placeId)
		this.#request = request
		try {
			const nearbyRegions = await request
			if (version !== this.#version) return
			this.regionChecks = (nearbyRegions ?? []).map((nearbyRegion) => ({
				nearbyRegion,
				status: "waiting",
				skippedJobIds: [],
			}))
			this.#hasLoadedRegions = true
		} catch (error) {
			if (version !== this.#version) return
			this.error = errorMessage(error, searchError)
			this.phase = "failed"
			return
		}
		await this.#checkRegionsFrom(0)
	}

	async skipFoundServer(): Promise<void> {
		const check = this.regionChecks[this.activeRegionIndex]
		if (this.phase !== "found" || !check) return
		check.skippedJobIds.push(this.foundJobId)
		this.foundJobId = ""
		await this.#checkRegionsFrom(this.activeRegionIndex)
	}

	async retry(): Promise<void> {
		if (this.phase !== "failed") return
		if (this.#hasLoadedRegions) {
			await this.#checkRegionsFrom(this.activeRegionIndex)
		} else {
			await this.start()
		}
	}

	dispose(): void {
		this.#version++
		void this.#request?.cancel()
		this.#request = null
	}

	async #checkRegionsFrom(startIndex: number): Promise<void> {
		this.error = ""
		await this.#checkRegion(startIndex, ++this.#version)
	}

	// Regions are checked one at a time, nearest first, so that the search
	// stops at the first region with an open server.
	async #checkRegion(index: number, version: number): Promise<void> {
		const check = this.regionChecks.at(index)
		if (!check) {
			this.#request = null
			this.phase = "exhausted"
			return
		}
		this.activeRegionIndex = index
		check.status = "checking"
		this.phase = "checking"
		const request = accountBackend.FindRegionServer({
			placeId: this.placeId,
			accountId: this.accountId,
			region: check.nearbyRegion.region.code,
			skippedJobIds: [...check.skippedJobIds],
		})
		this.#request = request
		let jobId: string
		try {
			jobId = await request
		} catch (error) {
			if (version !== this.#version) return
			check.status = "failed"
			this.error = errorMessage(error, searchError)
			this.phase = "failed"
			return
		}
		if (version !== this.#version) return
		if (jobId) {
			check.status = "found"
			this.foundJobId = jobId
			this.phase = "found"
			return
		}
		check.status = check.skippedJobIds.length > 0 ? "skipped" : "empty"
		await this.#checkRegion(index + 1, version)
	}
}
