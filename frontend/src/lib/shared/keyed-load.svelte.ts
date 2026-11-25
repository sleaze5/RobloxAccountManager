import type { AccountScope } from "../accounts/account-scope"
import type { CancellableRequest } from "./cancellable-request"
import { errorMessage } from "./errors"

// KeyedLoad holds what a page reads for one key at a time, such as the 3D
// model of the opened item. A key names what is read. The loaded value stays
// while the next one loads, so that a page can keep showing it.
export class KeyedLoad<Loaded> {
	loaded = $state.raw<Loaded | null>(null)
	// loadedKey names what loaded holds.
	loadedKey = $state("")
	// requestedKey names what the last load reads. isLoading and error belong
	// to it.
	requestedKey = $state("")
	isLoading = $state(false)
	error = $state("")
	readonly #scope: AccountScope
	#request: CancellableRequest<Loaded> | null = null

	constructor(scope: AccountScope) {
		this.#scope = scope
	}

	loadedFor(key: string): Loaded | null {
		return this.loadedKey === key ? this.loaded : null
	}

	errorFor(key: string): string {
		return this.requestedKey === key ? this.error : ""
	}

	// load reads what key names, unless it is already loaded or loading. A
	// failed read loads again on the next call. The empty key of a new load
	// matches nothing, so its first call always reads.
	async load(
		key: string,
		read: () => CancellableRequest<Loaded>,
		fallbackError: string,
	): Promise<void> {
		if (!this.#scope.isCurrent) return
		if (
			key === this.requestedKey &&
			((key === this.loadedKey && this.loaded !== null) || this.isLoading)
		) {
			return
		}
		void this.#request?.cancel()
		const request = read()
		this.#request = request
		this.requestedKey = key
		this.isLoading = true
		this.error = ""
		try {
			const loaded = await this.#scope.track(request)
			if (request !== this.#request || !this.#scope.isCurrent) return
			this.loaded = loaded
			this.loadedKey = key
		} catch (error) {
			if (request !== this.#request || !this.#scope.isCurrent) return
			this.loaded = null
			this.loadedKey = ""
			this.error = errorMessage(error, fallbackError)
		} finally {
			if (request === this.#request) {
				this.#request = null
				this.isLoading = false
			}
		}
	}
}
