export function isMoveShortcut(event: KeyboardEvent): boolean {
	return (
		(event.ctrlKey || event.metaKey) &&
		event.shiftKey &&
		(event.key === "ArrowUp" || event.key === "ArrowDown")
	)
}
