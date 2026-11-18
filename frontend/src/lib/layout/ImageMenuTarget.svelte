<script lang="ts">
	import type { Snippet } from "svelte"
	import { getWorkspace } from "./workspace-state.svelte"

	// ImageMenuTarget opens the image menu on a right-click, Shift+F10, or the
	// ContextMenu key, and looks like the image it wraps. label names the
	// image, such as "@builderman's profile image".
	let {
		imageUrl,
		label,
		children,
	}: { imageUrl: string; label: string; children: Snippet } = $props()

	const workspace = getWorkspace()

	function openMenu(event: MouseEvent | KeyboardEvent): void {
		workspace.openImageContextMenu(event, { imageUrl, label })
	}
</script>

<button
	type="button"
	class="shrink-0 cursor-default rounded-md"
	aria-label={label}
	aria-haspopup="menu"
	aria-expanded={workspace.imageContextMenu?.image.imageUrl === imageUrl}
	disabled={!imageUrl}
	onkeydown={(event) => {
		if (event.key === "ContextMenu" || (event.shiftKey && event.key === "F10")) {
			openMenu(event)
		}
	}}
	oncontextmenu={openMenu}>
	{@render children()}
</button>
