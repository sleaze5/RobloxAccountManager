import { tick } from "svelte"

// PageLocationKeys tell two locations of a page apart: context names the list
// or search that the page shows, and selection names what it opened there,
// such as a profile and its sub-page. An empty selection means that nothing
// is open.
export interface PageLocationKeys {
	context: string
	selection: string
}

const maxEntryCount = 100

// PageHistory remembers, in memory only, the locations that one page showed,
// so that the page goes back and forward between them like a browser.
export class PageHistory<Location> {
	#entries: Location[] = []
	#index = -1
	#isRestoring = false
	readonly #keysOf: (location: Location) => PageLocationKeys
	readonly #restore: (location: Location) => Promise<void> | void

	constructor(
		keysOf: (location: Location) => PageLocationKeys,
		restore: (location: Location) => Promise<void> | void,
	) {
		this.#keysOf = keysOf
		this.#restore = restore
	}

	// record adds the location that the page shows now. A location that only
	// opens something in the list that the last location showed, such as a
	// search that then opens its result, replaces it, so that it is one step.
	record(location: Location): void {
		if (this.#isRestoring) return
		const keys = this.#keysOf(location),
			current = this.#entries[this.#index],
			currentKeys = current === undefined ? null : this.#keysOf(current)
		if (current !== undefined && this.#hasSameKeys(current, location)) {
			this.#entries[this.#index] = location
			return
		}
		if (currentKeys?.context === keys.context && currentKeys.selection === "") {
			this.#entries[this.#index] = location
			// Opening the location before it again, such as by searching for
			// it again, is not a new step.
			const previous = this.#entries[this.#index - 1]
			if (previous !== undefined && this.#hasSameKeys(previous, location)) {
				this.#entries.splice(this.#index, 1)
				this.#index--
			}
		} else {
			this.#entries = [
				...this.#entries.slice(0, this.#index + 1),
				location,
			].slice(-maxEntryCount)
			this.#index = this.#entries.length - 1
		}
	}

	#hasSameKeys(first: Location, second: Location): boolean {
		const firstKeys = this.#keysOf(first),
			secondKeys = this.#keysOf(second)
		return (
			firstKeys.context === secondKeys.context &&
			firstKeys.selection === secondKeys.selection
		)
	}

	clear(): void {
		this.#entries = []
		this.#index = -1
	}

	back(): Promise<void> {
		return this.#go(-1)
	}

	forward(): Promise<void> {
		return this.#go(1)
	}

	async #go(step: -1 | 1): Promise<void> {
		const target = this.#entries[this.#index + step]
		if (target === undefined || this.#isRestoring) return
		this.#index += step
		// The page passes through other locations while it restores one, such
		// as a search before its result opens, which are not steps.
		this.#isRestoring = true
		try {
			await this.#restore(target)
			await tick()
		} finally {
			this.#isRestoring = false
		}
	}
}
