import { SvelteMap } from "svelte/reactivity"
import {
	accountBackend,
	GameServerOrder,
	GameServerRecordOrder,
} from "../backend/bridge"
import type {
	FriendsInGame,
	GameDatacenter,
	GameServer,
	GameServerDetails,
	GameServerPage,
	GameServerRecord,
	GameServerRecordPage,
	GameServerRegion,
	GameServerStats,
	ListedPrivateServer,
	ListedPrivateServerPage,
	PrivateServer,
	PrivateServerPage,
	SavedPrivateServer,
} from "../backend/bridge"
import { appSettings } from "../settings/settings-store.svelte"
import type { CancellableRequest } from "../shared/cancellable-request"
import { errorMessage } from "../shared/errors"
import { detailsKeepMs, isFresh } from "../shared/freshness"

export type ServerSource = "roblox" | "private" | "rovalra" | "rbxservers"
export type ServerPageSize = 10 | 25 | 50 | 100
export type RecordPageSize = 10 | 50 | 100

const serverPageFreshMs = 30_000,
	maxCachedServerPages = 50

type PageKey = { placeId: number } & Record<string, string | number | boolean>

// A page stays after it stops being fresh, so that a revisit shows it at once
// while it refreshes.
export class ServerPageCache<Page> {
	readonly #pages = new Map<
		string,
		{ placeId: number; page: Page; fetchedAt: number }
	>()

	get(key: PageKey): { page: Page; isFresh: boolean } | null {
		const id = JSON.stringify(key),
			entry = this.#pages.get(id)
		if (!entry) return null
		const ageMs = Date.now() - entry.fetchedAt
		if (ageMs < detailsKeepMs) {
			return { page: entry.page, isFresh: ageMs < serverPageFreshMs }
		}
		this.#pages.delete(id)
		return null
	}

	set(key: PageKey, page: Page): void {
		const now = Date.now()
		for (const [id, entry] of this.#pages) {
			if (now - entry.fetchedAt >= detailsKeepMs) this.#pages.delete(id)
		}
		const id = JSON.stringify(key)
		this.#pages.delete(id)
		this.#pages.set(id, { placeId: key.placeId, page, fetchedAt: now })
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

// ServerPager lists servers that load in pages. Each page that loads after the
// first adds its servers to page, without the servers that an earlier page
// already listed, because a server can move between pages while they load.
abstract class ServerPager<
	Cursor extends string | number,
	Server,
	Page extends { servers: Server[] | null },
> {
	// page holds the first page and the servers of every page after it.
	page = $state<Page | null>(null)
	loading = $state(false)
	loadingMore = $state(false)
	error = $state("")
	loadMoreError = $state("")
	// listVersion changes when the first page replaces the list, but not when
	// a later page adds to it.
	listVersion = $state(0)
	#nextCursor = $state<Cursor | null>(null)
	#request: CancellableRequest<Page> | null = null
	#requestKeyId = ""
	#shownKeyId = ""

	constructor(
		readonly placeId: number,
		private readonly firstCursor: Cursor,
		private readonly cache: ServerPageCache<Page>,
	) {}

	protected abstract key(cursor: Cursor): PageKey
	protected abstract fetch(key: PageKey): CancellableRequest<Page>
	protected abstract nextCursor(page: Page): Cursor | null
	protected abstract serverKey(server: Server): string | number

	get hasNext(): boolean {
		return this.page !== null && this.#nextCursor !== null
	}

	// first lists the servers again from the first page, such as after the
	// options changed.
	first(): void {
		void this.load()
	}

	refresh(): void {
		this.cache.invalidate(this.placeId)
		void this.load()
	}

	protected clear(): void {
		void this.#request?.cancel()
		this.#request = null
		this.#shownKeyId = ""
		this.#nextCursor = null
		this.page = null
		this.loading = this.loadingMore = false
		this.error = this.loadMoreError = ""
	}

	// load keeps the listed servers while their first page is fresh, so that
	// the list reopens where it was. Otherwise it shows the cached first page
	// at once and requests it when the cache has no fresh copy.
	async load(): Promise<void> {
		const key = this.key(this.firstCursor),
			keyId = JSON.stringify(key),
			cached = this.cache.get(key)
		if (this.#request && this.#requestKeyId === keyId) return
		if (this.page && this.#shownKeyId === keyId && cached?.isFresh) return
		void this.#request?.cancel()
		this.#request = null
		this.loadingMore = false
		this.error = this.loadMoreError = ""
		if (cached) this.#showFirstPage(keyId, cached.page)
		if (cached?.isFresh) {
			this.loading = false
			return
		}
		this.loading = true
		const request = this.fetch(key)
		this.#request = request
		this.#requestKeyId = keyId
		try {
			const page = await request
			if (this.#request !== request) return
			this.cache.set(key, page)
			this.#showFirstPage(keyId, page)
		} catch (error) {
			if (this.#request === request) {
				this.error = errorMessage(
					error,
					"Could not load the servers. Try again.",
				)
			}
		} finally {
			if (this.#request === request) {
				this.#request = null
				this.loading = false
			}
		}
	}

	async loadMore(): Promise<void> {
		const cursor = this.#nextCursor
		if (cursor === null || !this.page || this.loading || this.loadingMore) return
		const key = this.key(cursor),
			keyId = JSON.stringify(key),
			cached = this.cache.get(key)
		this.loadMoreError = ""
		if (cached?.isFresh) {
			this.#addPage(cached.page)
			return
		}
		this.loadingMore = true
		const request = this.fetch(key)
		this.#request = request
		this.#requestKeyId = keyId
		try {
			const page = await request
			if (this.#request !== request) return
			this.cache.set(key, page)
			this.#addPage(page)
		} catch (error) {
			if (this.#request === request) {
				this.loadMoreError = errorMessage(
					error,
					"Could not load more servers. Try again.",
				)
			}
		} finally {
			if (this.#request === request) {
				this.#request = null
				this.loadingMore = false
			}
		}
	}

	#showFirstPage(keyId: string, page: Page): void {
		this.#shownKeyId = keyId
		this.page = page
		this.#nextCursor = this.nextCursor(page)
		this.listVersion++
	}

	#addPage(page: Page): void {
		const listed = this.page?.servers ?? [],
			listedKeys = new Set(listed.map((server) => this.serverKey(server))),
			servers = [
				...listed,
				...(page.servers ?? []).filter(
					(server) => !listedKeys.has(this.serverKey(server)),
				),
			]
		this.page = Object.assign({}, page, { servers })
		this.#nextCursor = this.nextCursor(page)
	}

	dispose(): void {
		void this.#request?.cancel()
		this.#request = null
	}
}

export class GameServers extends ServerPager<string, GameServer, GameServerPage> {
	order = $state(GameServerOrder.ServerOrderRecommended)
	excludeFull = $state(false)
	limit = $state<ServerPageSize>(50)
	maxPing = $state(0)
	minFps = $state(0)
	// region filters the loaded servers by the place that their tiles name.
	region = $state("")

	constructor(
		placeId: number,
		cache: ServerPageCache<GameServerPage>,
		private readonly records: () => boolean,
	) {
		super(placeId, "", cache)
	}

	get servers(): GameServer[] {
		return (this.page?.servers ?? []).filter(
			(server) =>
				(!this.maxPing ||
					(server.pingMs !== null && server.pingMs <= this.maxPing)) &&
				server.fps >= this.minFps,
		)
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

	protected serverKey(server: GameServer): string {
		return server.serverId
	}
}

export class RecordedServers extends ServerPager<
	number,
	GameServerRecord,
	GameServerRecordPage
> {
	order = $state(GameServerRecordOrder.RecordOrderNewest)
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

	protected serverKey(server: GameServerRecord): string {
		return server.serverId
	}
}

export class PrivateServers extends ServerPager<
	string,
	PrivateServer,
	PrivateServerPage
> {
	#accountId: number

	constructor(
		placeId: number,
		cache: ServerPageCache<PrivateServerPage>,
		accountId: number,
	) {
		super(placeId, "", cache)
		this.#accountId = accountId
	}

	// showAccount lists the private servers of another account, and loads them
	// only while the list is visible.
	showAccount(accountId: number, isVisible: boolean): void {
		if (accountId === this.#accountId) return
		this.#accountId = accountId
		this.clear()
		if (isVisible && accountId) void this.load()
	}

	protected key(cursor: string): PageKey {
		return { placeId: this.placeId, cursor, accountId: this.#accountId }
	}

	protected fetch(key: PageKey): CancellableRequest<PrivateServerPage> {
		return accountBackend.ListPlacePrivateServers({
			placeId: this.placeId,
			accountId: key.accountId as number,
			cursor: key.cursor as string,
		})
	}

	protected nextCursor(page: PrivateServerPage): string | null {
		return page.nextCursor || null
	}

	protected serverKey(server: PrivateServer): number {
		return server.privateServerId
	}
}

export class ListedServers extends ServerPager<
	number,
	ListedPrivateServer,
	ListedPrivateServerPage
> {
	resolving = $state("")
	readonly #links = new Map<string, string>()
	#linkRequest: CancellableRequest<string> | null = null

	constructor(placeId: number, cache: ServerPageCache<ListedPrivateServerPage>) {
		super(placeId, 0, cache)
	}

	async launchLink(serverId: string): Promise<string | null> {
		const cached = this.#links.get(serverId)
		if (cached) return cached
		void this.#linkRequest?.cancel()
		const request = accountBackend.ResolveRBXServersLink(this.placeId, serverId)
		this.#linkRequest = request
		this.resolving = serverId
		try {
			const link = await request
			this.#links.set(serverId, link)
			return this.#linkRequest === request ? link : null
		} catch (error) {
			if (this.#linkRequest === request) throw error
			return null
		} finally {
			if (this.#linkRequest === request) {
				this.#linkRequest = null
				this.resolving = ""
			}
		}
	}

	override dispose(): void {
		super.dispose()
		void this.#linkRequest?.cancel()
		this.#linkRequest = null
	}

	protected key(): PageKey {
		return { placeId: this.placeId }
	}

	protected fetch(): CancellableRequest<ListedPrivateServerPage> {
		return accountBackend.ListRBXServersPrivateServers(this.placeId)
	}

	protected nextCursor(): null {
		return null
	}

	protected serverKey(server: ListedPrivateServer): string {
		return server.id
	}
}

export class SavedServers {
	servers = $state<SavedPrivateServer[] | null>(null)
	loading = $state(false)
	saving = $state(false)
	removing = $state<number | null>(null)
	error = $state("")
	actionError = $state("")
	#active = true
	#generation = 0

	constructor(readonly placeId: number) {}

	async load(): Promise<void> {
		if (!this.#active) return
		const generation = ++this.#generation,
			current = () => this.#active && generation === this.#generation
		this.loading = true
		this.error = ""
		try {
			const servers = await accountBackend.ListSavedPrivateServers(this.placeId)
			if (current()) this.servers = servers ?? []
		} catch (error) {
			if (current()) {
				this.error = errorMessage(
					error,
					"Could not load the saved links. Try again.",
				)
			}
		} finally {
			if (current()) this.loading = false
		}
	}

	#apply(change: (servers: SavedPrivateServer[]) => SavedPrivateServer[]): void {
		this.servers = change(this.servers ?? [])
		if (this.loading) void this.load()
	}

	async save(accountId: number, name: string, link: string): Promise<boolean> {
		if (!this.#active || this.saving) return false
		this.saving = true
		this.actionError = ""
		try {
			const server = await accountBackend.SavePrivateServerLink(
				this.placeId,
				accountId,
				name,
				link,
			)
			if (!this.#active) return false
			this.#apply((servers) => [
				...servers.filter((listed) => listed.id !== server.id),
				server,
			])
			return true
		} catch (error) {
			if (this.#active) {
				this.actionError = errorMessage(
					error,
					"Could not save the link. Try again.",
				)
			}
			return false
		} finally {
			if (this.#active) this.saving = false
		}
	}

	async remove(server: SavedPrivateServer): Promise<boolean> {
		if (!this.#active || this.removing !== null) return false
		this.removing = server.id
		this.actionError = ""
		try {
			await accountBackend.RemoveSavedPrivateServer(server.id)
			if (!this.#active) return false
			this.#apply((servers) =>
				servers.filter((listed) => listed.id !== server.id),
			)
			return true
		} catch (error) {
			if (this.#active) {
				this.actionError = errorMessage(
					error,
					"Could not remove the link. Try again.",
				)
			}
			return false
		} finally {
			if (this.#active) this.removing = null
		}
	}

	dispose(): void {
		this.#active = false
	}
}

// FriendsInGameLoader lists the shown account's friends who play any place of
// the universe, by server.
export class FriendsInGameLoader {
	friends = $state<FriendsInGame | null>(null)
	loading = $state(false)
	error = $state("")
	#accountId: number
	#fetchedAt = 0
	#request: ReturnType<typeof accountBackend.ListFriendsInGame> | null = null

	constructor(
		readonly universeId: number,
		accountId: number,
	) {
		this.#accountId = accountId
	}

	// showAccount switches accounts and loads only while visible.
	showAccount(accountId: number, isVisible: boolean): void {
		if (accountId === this.#accountId) return
		this.#accountId = accountId
		this.dispose()
		this.friends = null
		this.loading = false
		this.error = ""
		this.#fetchedAt = 0
		if (isVisible) void this.load()
	}

	// load shows kept friends and refetches stale ones.
	async load(): Promise<void> {
		if (this.#request) return
		if (this.friends && Date.now() - this.#fetchedAt < serverPageFreshMs) return
		await this.refresh()
	}

	async refresh(): Promise<void> {
		if (!this.#accountId) return
		void this.#request?.cancel()
		const request = accountBackend.ListFriendsInGame(
			this.#accountId,
			this.universeId,
		)
		this.#request = request
		this.loading = true
		this.error = ""
		try {
			const friends = await request
			if (this.#request !== request) return
			this.friends = friends
			this.#fetchedAt = Date.now()
		} catch (error) {
			if (this.#request === request) {
				this.error = errorMessage(
					error,
					"Could not load friends in this game. Try again.",
				)
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

const serverDetailsKeepMs = 5 * 60_000,
	// Each server's details cost one Roblox join request, so only a few run
	// at a time.
	serverDetailsRequestLimit = 2

export type ServerDetailsState =
	| { status: "loading" }
	| { status: "failed" }
	| { status: "loaded"; details: GameServerDetails }

// ServerDetailsLoader asks Roblox's join service about each server that comes
// into view, as the shown account, and keeps the answers for a few minutes.
export class ServerDetailsLoader {
	readonly #states = new SvelteMap<string, ServerDetailsState>()
	readonly #fetchedAt = new Map<string, number>()
	readonly #queue: string[] = []
	readonly #requests = new Set<CancellableRequest<GameServerDetails>>()
	#accountId: number

	constructor(
		readonly placeId: number,
		accountId: number,
	) {
		this.#accountId = accountId
	}

	get(serverId: string): ServerDetailsState | undefined {
		return this.#states.get(serverId)
	}

	showAccount(accountId: number): void {
		if (accountId === this.#accountId) return
		this.#accountId = accountId
		this.dispose()
		this.#states.clear()
		this.#fetchedAt.clear()
	}

	request(serverId: string): void {
		if (!this.#accountId) return
		const state = this.#states.get(serverId),
			fetchedAt = this.#fetchedAt.get(serverId) ?? 0
		if (state?.status === "loading") return
		if (state && Date.now() - fetchedAt < serverDetailsKeepMs) return
		this.#states.set(serverId, { status: "loading" })
		this.#queue.push(serverId)
		this.#next()
	}

	// refresh asks again about the servers that were asked about before.
	refresh(): void {
		this.#fetchedAt.clear()
		for (const serverId of [...this.#states.keys()]) {
			if (this.#states.get(serverId)?.status !== "loading") {
				this.#states.delete(serverId)
			}
		}
	}

	#next(): void {
		while (this.#requests.size < serverDetailsRequestLimit && this.#queue.length) {
			void this.#load(this.#queue.shift()!)
		}
	}

	async #load(serverId: string): Promise<void> {
		const request = accountBackend.GetGameServerDetails(
			this.placeId,
			this.#accountId,
			serverId,
		)
		this.#requests.add(request)
		try {
			const details = await request
			if (this.#requests.has(request)) {
				this.#states.set(serverId, { status: "loaded", details })
			}
		} catch {
			// The tile shows the details as unavailable, and the backend logs
			// the failure.
			if (this.#requests.has(request)) {
				this.#states.set(serverId, { status: "failed" })
			}
		} finally {
			if (this.#requests.delete(request)) {
				this.#fetchedAt.set(serverId, Date.now())
				this.#next()
			}
		}
	}

	dispose(): void {
		for (const request of this.#requests) void request.cancel()
		this.#requests.clear()
		for (const serverId of this.#queue) this.#states.delete(serverId)
		this.#queue.length = 0
	}
}

export class ServerStatsLoader {
	stats = $state<GameServerStats | null>(null)
	loading = $state(false)
	failed = $state(false)
	#fetchedAt = 0
	#request: ReturnType<typeof accountBackend.GetGameServerStats> | null = null

	constructor(readonly placeId: number) {}

	get isFresh(): boolean {
		return isFresh(this.#fetchedAt)
	}

	async load(): Promise<void> {
		void this.#request?.cancel()
		const request = accountBackend.GetGameServerStats(this.placeId)
		this.#request = request
		this.loading = true
		this.failed = false
		try {
			const stats = await request
			if (this.#request === request) {
				this.stats = stats
				this.#fetchedAt = Date.now()
			}
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

// PlaceServers holds the server lists of one place. The games store keeps it
// while the place stays selected, so that the Servers page reopens with its
// source, options, and pages.
export class PlaceServers {
	readonly roblox: GameServers
	readonly friendsInGame: FriendsInGameLoader
	readonly recorded: RecordedServers
	readonly private: PrivateServers
	readonly listed: ListedServers
	readonly saved: SavedServers
	readonly stats: ServerStatsLoader
	readonly details: ServerDetailsLoader

	constructor(
		readonly placeId: number,
		universeId: number,
		accountId: number,
		caches: {
			serverPages: ServerPageCache<GameServerPage>
			recordPages: ServerPageCache<GameServerRecordPage>
			privatePages: ServerPageCache<PrivateServerPage>
			listedPages: ServerPageCache<ListedPrivateServerPage>
		},
	) {
		this.roblox = new GameServers(
			placeId,
			caches.serverPages,
			() => appSettings.roValraEnabled,
		)
		this.friendsInGame = new FriendsInGameLoader(universeId, accountId)
		this.recorded = new RecordedServers(placeId, caches.recordPages)
		this.private = new PrivateServers(placeId, caches.privatePages, accountId)
		this.listed = new ListedServers(placeId, caches.listedPages)
		this.saved = new SavedServers(placeId)
		this.stats = new ServerStatsLoader(placeId)
		this.details = new ServerDetailsLoader(placeId, accountId)
	}

	dispose(): void {
		this.roblox.dispose()
		this.friendsInGame.dispose()
		this.recorded.dispose()
		this.private.dispose()
		this.listed.dispose()
		this.saved.dispose()
		this.stats.dispose()
		this.details.dispose()
	}
}

export function formatUptime(startedMs: number): string {
	return formatDuration(Date.now() - startedMs)
}

export function formatDuration(durationMs: number): string {
	if (durationMs < 60_000) return `${Math.max(0, Math.round(durationMs / 1000))}s`
	const minutes = Math.max(0, Math.floor(durationMs / 60_000)),
		days = Math.floor(minutes / 1440),
		hours = Math.floor((minutes % 1440) / 60)
	if (days) return `${days}d ${hours}h`
	if (hours) return `${hours}h ${minutes % 60}m`
	return `${minutes}m`
}

export type ServerHealth = {
	label: string
	tone: "success" | "warning" | "danger"
}

// serverHealth rates a server by its frame rate. Roblox servers aim for 60
// frames per second.
export function serverHealth(framesPerSecond: number): ServerHealth {
	if (framesPerSecond >= 45) return { label: "Smooth", tone: "success" }
	if (framesPerSecond >= 25) return { label: "Uneven", tone: "warning" }
	return { label: "Laggy", tone: "danger" }
}

// serverPlace names where a server runs: where RoValra places the datacenter
// that Roblox's join service names, or else where RoValra saw the server.
export function serverPlace(
	record: GameServerRecord | null | undefined,
	details: ServerDetailsState | undefined,
): string {
	const datacenter = details?.status === "loaded" ? details.details.datacenter : null
	if (datacenter) return datacenterPlace(datacenter)
	return record ? recordPlace(record) : ""
}

// serverPlaceVersion prefers the version that Roblox's join service names
// over RoValra's record.
export function serverPlaceVersion(
	record: GameServerRecord | null | undefined,
	details: ServerDetailsState | undefined,
): number {
	return (
		(details?.status === "loaded" ? details.details.placeVersion : 0) ||
		record?.placeVersion ||
		0
	)
}

export function datacenterPlace(datacenter: GameDatacenter): string {
	const area =
		datacenter.countryCode === "US"
			? datacenter.region || datacenter.country
			: datacenter.country || countryName(datacenter.countryCode)
	return [...new Set([datacenter.city, area].filter(Boolean))].join(", ")
}

export function datacenterLocation(datacenter: GameDatacenter): string {
	return [
		...new Set(
			[datacenter.city, datacenter.region, datacenter.country].filter(Boolean),
		),
	].join(", ")
}

const countryNames = new Intl.DisplayNames(["en"], { type: "region" })

function countryName(code: string): string {
	try {
		return countryNames.of(code) ?? code
	} catch {
		return code
	}
}

function recordPlace(record: GameServerRecord): string {
	const place = record.city || record.region,
		country = record.country || countryName(record.countryCode)
	return [...new Set([place, country].filter(Boolean))].join(", ")
}

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

export function outdated(version: number, newestVersion: number): boolean {
	return version > 0 && newestVersion > 0 && version < newestVersion
}
