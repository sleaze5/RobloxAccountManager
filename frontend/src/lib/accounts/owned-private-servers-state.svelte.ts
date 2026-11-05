import { accountBackend } from "../backend/bridge"
import type { OwnedPrivateServer, PrivateServerSettings } from "../backend/bridge"
import type { AccountStore } from "./account-store.svelte"
import { errorMessage } from "../shared/errors"

export class OwnedPrivateServersState {
	servers = $state<OwnedPrivateServer[] | null>(null)
	nextCursor = $state("")
	loading = $state(false)
	error = $state("")
	#active = true
	#request: ReturnType<typeof accountBackend.ListOwnedPrivateServers> | null = null
	readonly #accountId: number
	readonly #store: AccountStore

	constructor(accountId: number, store: AccountStore) {
		this.#accountId = accountId
		this.#store = store
	}

	#current(): boolean {
		return (
			this.#active &&
			this.#store.vault.unlocked &&
			this.#store.selectedAccount.id === this.#accountId
		)
	}

	refresh(): Promise<void> {
		return this.#load("")
	}

	loadMore(): Promise<void> {
		return this.nextCursor ? this.#load(this.nextCursor) : Promise.resolve()
	}

	update(settings: PrivateServerSettings): void {
		const server = this.servers?.find((item) => item.serverId === settings.serverId)
		if (!server) return
		server.name = settings.name
		server.active = settings.active
		server.renews = settings.renews
		server.expiresAtMs = settings.expiresAtMs || server.expiresAtMs
	}

	async #load(cursor: string): Promise<void> {
		if (!this.#current() || this.loading) return
		this.loading = true
		this.error = ""
		const request = accountBackend.ListOwnedPrivateServers(this.#accountId, cursor)
		this.#request = request
		try {
			const page = await request
			if (!this.#current()) return
			const loaded = cursor ? (this.servers ?? []) : [],
				known = new Set(loaded.map((server) => server.serverId))
			this.servers = [
				...loaded,
				...(page.servers ?? []).filter((server) => !known.has(server.serverId)),
			]
			this.nextCursor = page.nextCursor
		} catch (error) {
			if (!this.#current()) return
			this.error = errorMessage(
				error,
				"Could not load private servers. Refresh to retry.",
			)
		} finally {
			this.#request = null
			if (this.#current()) this.loading = false
		}
	}

	dispose(): void {
		this.#active = false
		void this.#request?.cancel()
		this.#request = null
		this.servers = null
		this.nextCursor = ""
		this.loading = false
		this.error = ""
	}
}
