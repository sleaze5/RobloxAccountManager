// MultiSelection holds the rows of a list that Ctrl+click marks, as the account
// list does, beside the row that the page shows. A right-click on a marked row
// acts on every marked row.
export class MultiSelection<Key> {
	keys = $state<Key[]>([])

	get isMultiple(): boolean {
		return this.keys.length > 1
	}

	has(key: Key): boolean {
		return this.keys.includes(key)
	}

	// toggle marks the row, or unmarks it. The first mark also marks the row
	// that the page shows, so that Ctrl+click adds to it.
	toggle(key: Key, shownKey: Key | undefined): void {
		const keys =
			this.keys.length === 0 && shownKey !== undefined && shownKey !== key
				? [shownKey]
				: this.keys
		this.keys = keys.includes(key)
			? keys.filter((markedKey) => markedKey !== key)
			: [...keys, key]
	}

	clear(): void {
		this.keys = []
	}

	// keysForMenu returns the marked rows when the menu opens on one of them, and
	// otherwise unmarks every row and returns the row alone.
	keysForMenu(key: Key): Key[] {
		if (this.isMultiple && this.has(key)) return [...this.keys]
		this.clear()
		return [key]
	}
}

export function isMultiSelectGesture(event: MouseEvent | KeyboardEvent): boolean {
	return event.ctrlKey || event.metaKey
}
