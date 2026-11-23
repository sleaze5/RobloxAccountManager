import { accountBackend } from "../backend/bridge"
import type { DeveloperProduct } from "../backend/bridge"
import { errorMessage } from "../shared/errors"
import { isFresh } from "../shared/freshness"

// UniverseDeveloperProducts holds the developer products of one universe. The
// games store keeps it while the place stays selected, so that the list
// reopens at once.
export class UniverseDeveloperProducts {
	developerProducts = $state<DeveloperProduct[] | null>(null)
	loading = $state(false)
	error = $state("")
	#fetchedAt = 0
	#request: ReturnType<typeof accountBackend.ListDeveloperProducts> | null = null

	constructor(readonly universeId: number) {}

	// load shows the kept products and requests them again when they are no
	// longer fresh.
	async load(): Promise<void> {
		if (this.#request || (this.developerProducts && isFresh(this.#fetchedAt))) {
			return
		}
		await this.refresh()
	}

	async refresh(): Promise<void> {
		void this.#request?.cancel()
		const request = accountBackend.ListDeveloperProducts(this.universeId)
		this.#request = request
		this.loading = true
		this.error = ""
		try {
			const developerProducts = await request
			if (this.#request !== request) return
			this.developerProducts = developerProducts ?? []
			this.#fetchedAt = Date.now()
		} catch (error) {
			if (this.#request === request) {
				this.error = errorMessage(
					error,
					"Could not load developer products. Try again.",
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
