import { maxTagNameLength, normalizeTagNameInput } from "./tag-name"

export class TagNameInputState {
	value = $state("")
	#separatorPending = false
	#clearError: () => void

	constructor(clearError: () => void) {
		this.#clearError = clearError
	}

	update(event: Event): void {
		const input = event.currentTarget as HTMLInputElement,
			rawValue = input.value,
			insertedAtEnd =
				input.selectionStart === rawValue.length &&
				input.selectionEnd === rawValue.length &&
				rawValue.startsWith(this.value),
			insertedValue = insertedAtEnd ? rawValue.slice(this.value.length) : "",
			value =
				this.#separatorPending && /^[A-Za-z0-9]/.test(insertedValue)
					? `${this.value}-${insertedValue}`
					: rawValue,
			normalized = normalizeTagNameInput(value)
				.slice(0, maxTagNameLength)
				.replace(/-$/, "")

		this.#separatorPending =
			normalized.length > 0 && (rawValue.endsWith("-") || /\s$/u.test(rawValue))
		this.value = normalized
		input.value = normalized
		this.#clearError()
	}

	resetSeparator(): void {
		this.#separatorPending = false
	}

	finish(): void {
		this.resetSeparator()
		this.#clearError()
	}

	keydown(event: KeyboardEvent): void {
		if (!this.#separatorPending) {
			return
		}

		const input = event.currentTarget as HTMLInputElement,
			atEnd =
				input.selectionStart === this.value.length &&
				input.selectionEnd === this.value.length
		if (event.key === "Backspace" && atEnd) {
			event.preventDefault()
			this.resetSeparator()
			return
		}
		if (
			!atEnd ||
			(!/^[A-Za-z0-9 -]$/.test(event.key) &&
				!["Alt", "Control", "Meta", "Shift"].includes(event.key))
		) {
			this.resetSeparator()
		}
	}
}
