export function moveListFocus(event: KeyboardEvent): void {
	if (event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) return
	if (event.key !== "ArrowDown" && event.key !== "ArrowUp") return
	event.preventDefault()
	const rows = Array.from(document.querySelectorAll<HTMLElement>("[data-list-item]")),
		index = rows.indexOf(event.currentTarget as HTMLElement)
	rows[index + (event.key === "ArrowDown" ? 1 : -1)]?.focus()
}

export function isMoveShortcut(event: KeyboardEvent): boolean {
	return (
		(event.ctrlKey || event.metaKey) &&
		event.shiftKey &&
		(event.key === "ArrowUp" || event.key === "ArrowDown")
	)
}
