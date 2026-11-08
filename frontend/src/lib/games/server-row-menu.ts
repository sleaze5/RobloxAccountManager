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
