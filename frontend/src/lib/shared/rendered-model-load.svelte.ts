import type { AccountScope } from "../accounts/account-scope"
import type { RenderedModel } from "../backend/bridge"
import type { CancellableRequest } from "./cancellable-request"
import { errorMessage } from "./errors"

// RenderedModelLoad holds the 3D model that a page shows, such as an avatar or
// an item. A key names what a model shows. The loaded model stays while the
// next one loads, so that a page can keep showing it.
export class RenderedModelLoad {
	model = $state.raw<RenderedModel | null>(null)
	// modelKey names what model shows.
	modelKey = $state("")
	// requestedKey names what the last load reads. isLoading and error belong
	// to it.
	requestedKey = $state("")
	isLoading = $state(false)
	error = $state("")
	readonly #scope: AccountScope
	#request: CancellableRequest<RenderedModel> | null = null

	constructor(scope: AccountScope) {
		this.#scope = scope
	}

	// load reads the model that key names, unless it is already loaded or
	// loading. A failed read loads again on the next call. The empty key of a
	// new load matches no model, so its first call always reads.
	async load(
		key: string,
		readModel: () => CancellableRequest<RenderedModel>,
		fallbackError: string,
	): Promise<void> {
		if (!this.#scope.isCurrent) return
		if (
			key === this.requestedKey &&
			((key === this.modelKey && this.model) || this.isLoading)
		) {
			return
		}
		void this.#request?.cancel()
		const request = readModel()
		this.#request = request
		this.requestedKey = key
		this.isLoading = true
		this.error = ""
		try {
			const model = await this.#scope.track(request)
			if (request !== this.#request || !this.#scope.isCurrent) return
			this.model = model
			this.modelKey = key
		} catch (error) {
			if (request !== this.#request || !this.#scope.isCurrent) return
			this.model = null
			this.modelKey = ""
			this.error = errorMessage(error, fallbackError)
		} finally {
			if (request === this.#request) {
				this.#request = null
				this.isLoading = false
			}
		}
	}
}
