export const maxTagNameLength = 40

const validTagNamePattern = /^[a-z0-9]+(?:-[a-z0-9]+)*$/,
	reservedTagNames = new Set(["all", "favorite", "favorites"])

export function normalizeTagNameInput(value: string): string {
	let normalized = "",
		separatorPending = false

	for (const character of value) {
		if (character >= "A" && character <= "Z") {
			if (separatorPending && normalized) {
				normalized += "-"
			}
			normalized += character.toLowerCase()
			separatorPending = false
			continue
		}
		if (
			(character >= "a" && character <= "z") ||
			(character >= "0" && character <= "9")
		) {
			if (separatorPending && normalized) {
				normalized += "-"
			}
			normalized += character
			separatorPending = false
			continue
		}
		if ((character === "-" || /\s/u.test(character)) && normalized) {
			separatorPending = true
			continue
		}

		separatorPending = false
	}

	return normalized
}

export function tagNameError(value: string): string {
	if (!value) {
		return "Enter a tag name."
	}
	if (value.length > maxTagNameLength) {
		return `Use ${maxTagNameLength} characters or fewer.`
	}
	if (!validTagNamePattern.test(value)) {
		return "Use lowercase letters and numbers, with single hyphens between words."
	}
	if (reservedTagNames.has(value)) {
		return "Choose a different name. This name is reserved."
	}

	return ""
}

export function isValidTagName(value: string): boolean {
	return tagNameError(value) === ""
}
