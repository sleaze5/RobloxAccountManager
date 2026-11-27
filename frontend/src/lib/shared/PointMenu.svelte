<script lang="ts">
	import type { Snippet } from "svelte"
	import { floatMenuAtPoint, menuIn, menuKeyboard, menuOut } from "./presence"

	// PointMenu is a context menu that opens at a point: at the pointer for a
	// right click, or under the trigger for a keyboard or button press. owner
	// is the data attribute that marks the menu and its triggers, so that a
	// click on a trigger toggles the menu instead of closing and reopening it.
	let {
		label,
		point,
		owner,
		onClose,
		children,
	}: {
		label: string
		point: { x: number; y: number }
		owner: string
		onClose: () => void
		children: Snippet
	} = $props()

	function closeOutside(event: MouseEvent): void {
		if (event.target instanceof Element && event.target.closest(`[${owner}]`)) {
			return
		}
		onClose()
	}
</script>

<svelte:window
	onclick={closeOutside}
	onkeydown={(event) => {
		if (event.key === "Escape") onClose()
	}} />

<div
	class="fixed menu"
	role="menu"
	aria-label={label}
	tabindex="-1"
	{...{ [owner]: "" }}
	onkeydown={(event) => {
		if (event.key === "Tab") onClose()
	}}
	use:floatMenuAtPoint={point}
	use:menuKeyboard
	in:menuIn
	out:menuOut>
	{@render children()}
</div>
