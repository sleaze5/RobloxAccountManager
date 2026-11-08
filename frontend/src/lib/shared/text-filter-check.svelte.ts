import { TextModeration } from "../backend/bridge"
import type { TextFilter } from "../backend/bridge"

export class TextFilterCheck {
	#checked = $state<{ text: string; filter: TextFilter | null } | null>(null)
	readonly #currentText: () => string

	constructor(currentText: () => string) {
		this.#currentText = currentText
	}

	get current(): { text: string; filter: TextFilter | null } | null {
		return this.#checked?.text === this.#currentText() ? this.#checked : null
	}

	get isClean(): boolean {
		return this.current?.filter?.moderation === TextModeration.TextModerationNone
	}

	get isTagged(): boolean {
		const filter = this.current?.filter
		return !!filter && filter.moderation !== TextModeration.TextModerationNone
	}

	get filteredPreview(): string | undefined {
		return this.isTagged ? this.current?.filter?.filteredText : undefined
	}

	async run(text: string, check: () => Promise<TextFilter>): Promise<boolean> {
		try {
			const filter = await check()
			this.#checked = { text, filter }
			return filter.moderation === TextModeration.TextModerationNone
		} catch (error) {
			this.#checked = { text, filter: null }
			throw error
		}
	}

	clear(): void {
		this.#checked = null
	}
}
