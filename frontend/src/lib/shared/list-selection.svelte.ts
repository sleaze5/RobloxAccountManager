// ListSelection holds the rows that the user marks while a list is in
// selection mode, so that an action runs on all of them at once.
export class ListSelection<Key> {
	isSelecting = $state(false)
	selectedKeys = $state.raw<Key[]>([])

	get selectedCount(): number {
		return this.selectedKeys.length
	}

	setSelecting(isSelecting: boolean): void {
		this.isSelecting = isSelecting
		this.selectedKeys = []
	}

	clear(): void {
		this.selectedKeys = []
	}

	has(key: Key): boolean {
		return this.selectedKeys.includes(key)
	}

	add(keys: Key[]): void {
		this.selectedKeys = [
			...this.selectedKeys,
			...new Set(keys.filter((key) => !this.has(key))),
		]
	}

	toggle(key: Key): void {
		this.selectedKeys = this.has(key)
			? this.selectedKeys.filter((selectedKey) => selectedKey !== key)
			: [...this.selectedKeys, key]
	}

	areAllSelected(shownKeys: Key[]): boolean {
		return shownKeys.length > 0 && shownKeys.every((key) => this.has(key))
	}

	// toggleAll selects every shown row, or clears the selection when every
	// shown row is already selected.
	toggleAll(shownKeys: Key[]): void {
		this.selectedKeys = this.areAllSelected(shownKeys) ? [] : [...shownKeys]
	}

	// keepOnly drops the selected rows that the list no longer has, such as
	// rows that an action removed.
	keepOnly(remainingKeys: Key[]): void {
		const remaining = new Set(remainingKeys)
		this.selectedKeys = this.selectedKeys.filter((key) => remaining.has(key))
	}
}

// isMultiSelectGesture tells whether a click on a row, or Enter or Space on it,
// holds Ctrl, or Command on macOS. As in the account list, the gesture turns on
// selection mode with the shown row and that row selected.
export function isMultiSelectGesture(event: MouseEvent | KeyboardEvent): boolean {
	return event.ctrlKey || event.metaKey
}
