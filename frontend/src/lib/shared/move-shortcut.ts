// moveListFocus moves the focus between the rows of a list. A list is one tab
// stop, the selected row or the first row, so that Tab moves past the list.
export function moveListFocus(event: KeyboardEvent): void {
	if (event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) return
	const rows = Array.from(document.querySelectorAll<HTMLElement>("[data-list-item]")),
		index = rows.indexOf(event.currentTarget as HTMLElement)
	let nextRow: HTMLElement | undefined
	if (event.key === "ArrowDown") nextRow = rows[index + 1]
	else if (event.key === "ArrowUp") nextRow = rows[index - 1]
	else if (event.key === "Home") nextRow = rows[0]
	else if (event.key === "End") nextRow = rows.at(-1)
	else return
	event.preventDefault()
	nextRow?.focus()
}

export function isMoveShortcut(event: KeyboardEvent): boolean {
	return (
		(event.ctrlKey || event.metaKey) &&
		event.shiftKey &&
		(event.key === "ArrowUp" || event.key === "ArrowDown")
	)
}
