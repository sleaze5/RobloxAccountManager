export type ServerMenuOpener = (
	key: string,
	trigger: HTMLElement,
	toggle: boolean,
) => void

export function openServerMenuFromRow(
	event: MouseEvent & { currentTarget: HTMLElement },
	key: string,
	onMenu: ServerMenuOpener,
): void {
	const trigger = event.currentTarget.querySelector<HTMLElement>(
		"[data-server-actions]",
	)
	if (!trigger) return
	event.preventDefault()
	onMenu(key, trigger, true)
}

// friendServerKey keys a friend server row. Rows in the Roblox list key by job
// ID alone, so the prefix keeps a server that is in both lists apart.
export function friendServerKey(placeId: number, jobId: string): string {
	return `friend-${placeId}-${jobId}`
}
