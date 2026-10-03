import {
	accountBackend,
	GameServerOrder,
	GameServerRecordOrder,
} from "../backend/bridge"
import type {
	GameServer,
	GameServerPage,
	GameServerRecord,
	GameServerRecordPage,
	GameServerRegion,
	GameServerStats,
} from "../backend/bridge"

export type ServerSource = "roblox" | "rovalra"
export type ServerPageSize = 10 | 25 | 50 | 100
export type RecordPageSize = 10 | 50 | 100

// Servers open, fill, and close quickly, so pages are reused only briefly.
const serverPageFreshMs = 30_000,
	maxCachedServerPages = 50

type PageKey = { placeId: number } & Record<string, string | number | boolean>
type CancellableRequest<T> = Promise<T> & { cancel(): unknown }

// ServerPageCache keeps recently loaded server pages so paging back and reopening a place is instant.
export class ServerPageCache<Page> {
	readonly #pages = new Map<
		string,
		{ placeId: number; page: Page; fetchedAt: number }
	>()

	get(key: PageKey): Page | null {
		const id = JSON.stringify(key),
			entry = this.#pages.get(id)
		if (!entry) return null
		if (Date.now() - entry.fetchedAt < serverPageFreshMs) return entry.page
		this.#pages.delete(id)
		return null
	}

	set(key: PageKey, page: Page): void {
		const now = Date.now()
		for (const [id, entry] of this.#pages) {
			if (now - entry.fetchedAt >= serverPageFreshMs) this.#pages.delete(id)
		}
		const id = JSON.stringify(key)
		this.#pages.delete(id)
		this.#pages.set(id, { placeId: key.placeId, page, fetchedAt: now })
		// Maps iterate in insertion order, so the first entry is the oldest.
		while (this.#pages.size > maxCachedServerPages) {
			const oldest = this.#pages.keys().next().value
			if (oldest === undefined) break
			this.#pages.delete(oldest)
		}
	}

	invalidate(placeId: number): void {
		for (const [id, entry] of this.#pages) {
			if (entry.placeId === placeId) this.#pages.delete(id)
		}
	}

	clear(): void {
		this.#pages.clear()
	}
}

// ServerPager pages through the servers of one place, remembering the cursors visited so far.
abstract class ServerPager<Cursor extends string | number, Page> {
	page = $state<Page | null>(null)
	loading = $state(false)
	error = $state("")
	#cursors: Cursor[]
	#pageNumber = $state(1)
	#request: CancellableRequest<Page> | null = null

	constructor(
		readonly placeId: number,
		private readonly firstCursor: Cursor,
		private readonly cache: ServerPageCache<Page>,
	) {
		this.#cursors = [firstCursor]
	}

	protected abstract key(cursor: Cursor): PageKey
	protected abstract fetch(key: PageKey): CancellableRequest<Page>
	protected abstract nextCursor(page: Page): Cursor | null

	get pageNumber(): number {
		return this.#pageNumber
	}

	get hasNext(): boolean {
		return this.page !== null && this.nextCursor(this.page) !== null
	}

	next(): void {
		const cursor = this.page && this.nextCursor(this.page)
		if (cursor === null || this.loading) return
		this.#cursors.push(cursor)
		void this.load()
	}

	previous(): void {
		if (this.#cursors.length < 2 || this.loading) return
		this.#cursors.pop()
		void this.load()
	}

	// first returns to the first page with the current options.
	first(): void {
		this.#cursors = [this.firstCursor]
		void this.load()
	}

	// refresh drops every cached page of this place and reloads the current page.
	refresh(): void {
		this.cache.invalidate(this.placeId)
		void this.load()
	}

	async load(): Promise<void> {
		void this.#request?.cancel()
		this.#request = null
		this.#pageNumber = this.#cursors.length
		const key = this.key(this.#cursors.at(-1) ?? this.firstCursor),
			cached = this.cache.get(key)
		this.error = ""
		if (cached) {
			this.page = cached
			this.loading = false
			return
		}
		this.loading = true
		const request = this.fetch(key)
		this.#request = request
		try {
			const page = await request
			if (this.#request !== request) return
			this.cache.set(key, page)
			this.page = page
		} catch (error) {
			if (this.#request === request) {
				this.error =
					error instanceof Error && error.message.trim()
						? error.message
						: "Servers could not be loaded. Try again."
			}
		} finally {
			if (this.#request === request) {
				this.#request = null
				this.loading = false
			}
		}
	}

	dispose(): void {
		void this.#request?.cancel()
		this.#request = null
	}
}

// GameServers pages through the public servers Roblox lists for a place.
export class GameServers extends ServerPager<string, GameServerPage> {
	order = $state(GameServerOrder.ServerOrderRecommended)
	excludeFull = $state(false)
	limit = $state<ServerPageSize>(50)
	// maxPing filters the loaded page only, because Roblox cannot filter by ping.
	// Servers without a reported ping never pass a ping limit.
	maxPing = $state(0)

	constructor(
		placeId: number,
		cache: ServerPageCache<GameServerPage>,
		// records reports whether pages include RoValra records, which belong in the cache key.
		private readonly records: () => boolean,
	) {
		super(placeId, "", cache)
	}

	get servers(): GameServer[] {
		const servers = this.page?.servers ?? []
		return this.maxPing
			? servers.filter(
					(server) => server.pingMs !== null && server.pingMs <= this.maxPing,
				)
			: servers
	}

	setOptions(options: {
		order?: GameServerOrder
		excludeFull?: boolean
		limit?: ServerPageSize
	}): void {
		this.order = options.order ?? this.order
		this.excludeFull = options.excludeFull ?? this.excludeFull
		this.limit = options.limit ?? this.limit
		this.first()
	}

	protected key(cursor: string): PageKey {
		return {
			placeId: this.placeId,
			cursor,
			order: this.order,
			excludeFull: this.excludeFull,
			limit: this.limit,
			records: this.records(),
		}
	}

	protected fetch(key: PageKey): CancellableRequest<GameServerPage> {
		return accountBackend.ListGameServers({
			placeId: this.placeId,
			cursor: key.cursor as string,
			order: this.order,
			excludeFull: this.excludeFull,
			limit: this.limit,
		})
	}

	protected nextCursor(page: GameServerPage): string | null {
		return page.nextCursor || null
	}
}

// RecordedServers pages through the servers RoValra has recorded for a place.
export class RecordedServers extends ServerPager<number, GameServerRecordPage> {
	order = $state(GameServerRecordOrder.RecordOrderNewest)
	// region is a GameServerRegion code, or empty for every region.
	region = $state("")
	limit = $state<RecordPageSize>(50)

	constructor(placeId: number, cache: ServerPageCache<GameServerRecordPage>) {
		super(placeId, 0, cache)
	}

	setOptions(options: {
		order?: GameServerRecordOrder
		region?: string
		limit?: RecordPageSize
	}): void {
		this.region = options.region ?? this.region
		// RoValra lists a region's servers newest first only.
		this.order = this.region
			? GameServerRecordOrder.RecordOrderNewest
			: (options.order ?? this.order)
		this.limit = options.limit ?? this.limit
		this.first()
	}

	protected key(cursor: number): PageKey {
		return {
			placeId: this.placeId,
			cursor,
			order: this.order,
			region: this.region,
			limit: this.limit,
		}
	}

	protected fetch(key: PageKey): CancellableRequest<GameServerRecordPage> {
		return accountBackend.ListRecordedGameServers({
			placeId: this.placeId,
			cursor: key.cursor as number,
			order: this.order,
			region: this.region,
			limit: this.limit,
		})
	}

	protected nextCursor(page: GameServerRecordPage): number | null {
		return page.nextCursor || null
	}
}

// ServerStatsLoader loads the regions and newest version RoValra has recorded for a place.
export class ServerStatsLoader {
	stats = $state<GameServerStats | null>(null)
	loading = $state(false)
	failed = $state(false)
	#request: ReturnType<typeof accountBackend.GetGameServerStats> | null = null

	constructor(readonly placeId: number) {}

	async load(): Promise<void> {
		void this.#request?.cancel()
		const request = accountBackend.GetGameServerStats(this.placeId)
		this.#request = request
		this.loading = true
		this.failed = false
		try {
			const stats = await request
			if (this.#request === request) this.stats = stats
		} catch {
			if (this.#request === request) this.failed = true
		} finally {
			if (this.#request === request) {
				this.#request = null
				this.loading = false
			}
		}
	}

	dispose(): void {
		void this.#request?.cancel()
		this.#request = null
	}
}

// formatUptime shortens the time since a server started, such as "3h 12m" or "2d 4h".
export function formatUptime(startedMs: number, now = Date.now()): string {
	const minutes = Math.max(0, Math.floor((now - startedMs) / 60_000)),
		days = Math.floor(minutes / 1440),
		hours = Math.floor((minutes % 1440) / 60)
	if (days) return `${days}d ${hours}h`
	if (hours) return `${hours}h ${minutes % 60}m`
	return `${minutes}m`
}

const countryNames = new Intl.DisplayNames(["en"], { type: "region" })

function countryName(code: string): string {
	try {
		return countryNames.of(code) ?? code
	} catch {
		return code
	}
}

// recordPlace names where a server runs, such as "Amsterdam, Netherlands".
export function recordPlace(record: GameServerRecord): string {
	const place = record.city || record.region,
		country = record.country || countryName(record.countryCode)
	return [...new Set([place, country].filter(Boolean))].join(", ")
}

// recordLocation names where a server runs in full, such as "Amsterdam, North Holland, Netherlands".
export function recordLocation(record: GameServerRecord): string {
	return [
		...new Set([record.city, record.region, record.country].filter(Boolean)),
	].join(", ")
}

export function regionLabel(region: GameServerRegion): string {
	const country = countryName(region.countryCode),
		cities = (region.cities ?? []).filter((city) => city !== country).join(", ")
	return cities ? `${cities}, ${country}` : country
}

// outdated reports whether a server runs an older version than the newest one RoValra has seen.
export function outdated(version: number, newestVersion: number): boolean {
	return version > 0 && newestVersion > 0 && version < newestVersion
}

export function versionTooltip(version: number, newestVersion: number): string {
	return outdated(version, newestVersion)
		? `Place version ${version}. The newest running version is ${newestVersion}.`
		: `Place version ${version}`
}
