import { accountBackend } from "../backend/bridge"
import type { OwnedPrivateServer, PrivateServerSettings } from "../backend/bridge"
import type { AccountStore } from "./account-store.svelte"
import { AccountScope } from "./account-scope"
import { errorMessage } from "../shared/errors"

export class OwnedPrivateServersState {
	servers = $state<OwnedPrivateServer[] | null>(null)
	nextCursor = $state("")
	loading = $state(false)
	error = $state("")
	readonly #scope: AccountScope

	constructor(accountId: number, store: AccountStore) {
		this.#scope = new AccountScope(accountId, store)
	}

	refresh(): Promise<void> {
		return this.#load("")
	}

	loadMore(): Promise<void> {
		return this.nextCursor ? this.#load(this.nextCursor) : Promise.resolve()
	}

	update(settings: PrivateServerSettings): void {
		const server = this.servers?.find(
			(ownedServer) => ownedServer.serverId === settings.serverId,
		)
		if (!server) return
		server.name = settings.name
		server.active = settings.active
		server.renews = settings.renews
		server.expiresAtMs = settings.expiresAtMs || server.expiresAtMs
	}

	async #load(cursor: string): Promise<void> {
		if (!this.#scope.isCurrent || this.loading) return
		this.loading = true
		this.error = ""
		try {
			const page = await this.#scope.track(
				accountBackend.ListOwnedPrivateServers(this.#scope.accountId, cursor),
			)
			if (!this.#scope.isCurrent) return
			const loaded = cursor ? (this.servers ?? []) : [],
				known = new Set(loaded.map((server) => server.serverId))
			this.servers = [
				...loaded,
				...(page.servers ?? []).filter((server) => !known.has(server.serverId)),
			]
			this.nextCursor = page.nextCursor
		} catch (error) {
			if (!this.#scope.isCurrent) return
			this.error = errorMessage(
				error,
				"Could not load private servers. Refresh to retry.",
			)
		} finally {
			if (this.#scope.isCurrent) this.loading = false
		}
	}

	dispose(): void {
		this.#scope.dispose()
		this.servers = null
		this.nextCursor = ""
		this.loading = false
		this.error = ""
	}
}
