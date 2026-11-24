<script lang="ts">
	import type { Snippet } from "svelte"
	import { getWorkspace } from "./workspace-state.svelte"

	// ImageMenuTarget opens the image menu on a right-click, Shift+F10, or the
	// ContextMenu key, and looks like the image it wraps. label names the
	// image, such as "@builderman's profile image". onOpen makes a press open
	// what the image shows, which openLabel names, such as "Show avatar".
	let {
		imageUrl,
		label,
		openLabel = "",
		onOpen,
		children,
	}: {
		imageUrl: string
		label: string
		openLabel?: string
		onOpen?: () => void
		children: Snippet
	} = $props()

	const workspace = getWorkspace()

	function openMenu(event: MouseEvent | KeyboardEvent): void {
		if (!imageUrl) return
		workspace.openImageContextMenu(event, { imageUrl, label })
	}
</script>

<button
	type="button"
	class={["shrink-0 rounded-md", onOpen ? "cursor-pointer" : "cursor-default"]}
	aria-label={openLabel ? `${label}. ${openLabel}` : label}
	data-tooltip={openLabel || undefined}
	aria-haspopup="menu"
	aria-expanded={workspace.imageContextMenu?.image.imageUrl === imageUrl}
	disabled={!imageUrl && !onOpen}
	onclick={onOpen}
	onkeydown={(event) => {
		if (event.key === "ContextMenu" || (event.shiftKey && event.key === "F10")) {
			openMenu(event)
		}
	}}
	oncontextmenu={openMenu}>
	{@render children()}
</button>
