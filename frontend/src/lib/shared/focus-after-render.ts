import { tick } from "svelte"

// focusAfterRender focuses the element that the selector names once the
// pending state changes show, such as a field of a page that is opening.
export async function focusAfterRender(selector: string): Promise<void> {
	await tick()
	document.querySelector<HTMLElement>(selector)?.focus()
}
