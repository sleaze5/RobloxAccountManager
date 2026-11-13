import { errorMessage } from "./errors"

interface CancellableRequest<T> extends Promise<T> {
	cancel(): unknown
}

export interface SectionFields<Snapshot, Section> {
	section: Section
	fields: (keyof Snapshot)[]
}

interface SectionResponse<Section> {
	section: Section
	isComplete: boolean
}

// SectionLoad fills one snapshot from requests that each read one section, so
// that every section shows as soon as it arrives. A section that fails keeps
// its last values. The snapshot stays until reset, so a revisit shows the last
// values at once while they refresh.
export class SectionLoad<Snapshot extends object, Section> {
	readonly snapshot: Snapshot
	loadedSections = $state<Section[]>([])
	loadingSections = $state<Section[]>([])
	failedSections = $state<Section[]>([])
	error = $state("")
	#loadCount = 0
	readonly #requests = new Set<CancellableRequest<unknown>>()
	readonly #sections: SectionFields<Snapshot, Section>[]
	readonly #createEmptySnapshot: () => Snapshot
	readonly #errorFallback: string

	constructor(
		sections: SectionFields<Snapshot, Section>[],
		createEmptySnapshot: () => Snapshot,
		errorFallback: string,
	) {
		this.#sections = sections
		this.#createEmptySnapshot = createEmptySnapshot
		this.#errorFallback = errorFallback
		this.snapshot = $state(createEmptySnapshot())
	}

	get isLoading(): boolean {
		return this.loadingSections.length > 0
	}

	get hasAnySection(): boolean {
		return this.loadedSections.length > 0
	}

	hasSection(section: Section): boolean {
		return this.loadedSections.includes(section)
	}

	// isSectionPending is true until the section loads or becomes unavailable,
	// including before the first load starts, so that a placeholder holds its
	// place from the first frame.
	isSectionPending(section: Section): boolean {
		return !this.hasSection(section) && !this.isSectionUnavailable(section)
	}

	// A failed request ends the other requests of its load, so the sections that
	// they would have read are unavailable too.
	isSectionUnavailable(section: Section): boolean {
		return (
			this.failedSections.includes(section) ||
			(!!this.error && !this.isLoading && !this.hasSection(section))
		)
	}

	// load reads every section at once, in the order of the section list. It
	// returns false when a newer load or a reset replaced it. A rejected request
	// means that the session failed, so it ends the other requests too.
	async load(
		readSection: (
			section: Section,
		) => CancellableRequest<Snapshot & SectionResponse<Section>>,
		isCurrent: () => boolean,
	): Promise<boolean> {
		this.#cancelRequests()
		const loadCount = ++this.#loadCount,
			isCurrentLoad = () => loadCount === this.#loadCount && isCurrent()
		this.error = ""
		this.failedSections = []
		this.loadingSections = this.#sections.map(({ section }) => section)
		await Promise.allSettled(
			this.#sections.map(({ section, fields }) =>
				this.#loadSection(section, fields, readSection, isCurrentLoad),
			),
		)
		return isCurrentLoad()
	}

	async #loadSection(
		section: Section,
		fields: (keyof Snapshot)[],
		readSection: (
			section: Section,
		) => CancellableRequest<Snapshot & SectionResponse<Section>>,
		isCurrentLoad: () => boolean,
	): Promise<void> {
		const request = readSection(section)
		this.#requests.add(request)
		try {
			const sectionSnapshot = await request
			if (!isCurrentLoad()) return
			if (sectionSnapshot.section !== section) {
				throw new Error("The data changed while loading. Refresh to retry.")
			}
			if (!sectionSnapshot.isComplete) {
				this.failedSections.push(section)
				return
			}
			for (const field of fields) {
				Object.assign(this.snapshot, { [field]: sectionSnapshot[field] })
			}
			if (!this.loadedSections.includes(section))
				this.loadedSections.push(section)
		} catch (error) {
			if (!isCurrentLoad()) return
			this.failedSections.push(section)
			this.error ||= errorMessage(error, this.#errorFallback)
			this.#cancelRequests()
		} finally {
			this.#requests.delete(request)
			if (isCurrentLoad()) {
				this.loadingSections = this.loadingSections.filter(
					(loadingSection) => loadingSection !== section,
				)
			}
		}
	}

	#cancelRequests(): void {
		for (const request of this.#requests) void request.cancel()
		this.#requests.clear()
	}

	// cancel stops the running load and keeps the sections that arrived.
	cancel(): void {
		this.#loadCount++
		this.#cancelRequests()
		this.loadingSections = []
	}

	reset(): void {
		this.cancel()
		Object.assign(this.snapshot, this.#createEmptySnapshot())
		this.loadedSections = []
		this.failedSections = []
		this.error = ""
	}
}
