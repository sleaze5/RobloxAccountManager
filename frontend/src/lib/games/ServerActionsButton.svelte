<script lang="ts">
	import MoreHorizontal from "@lucide/svelte/icons/ellipsis"

	let {
		open,
		menuID,
		onMenu,
	}: {
		open: boolean
		menuID: string
		onMenu: (trigger: HTMLElement, toggle: boolean) => void
	} = $props()
</script>

<button
	class="game-server-actions icon-action"
	type="button"
	aria-label="Server actions"
	aria-haspopup="menu"
	aria-expanded={open}
	aria-controls={open ? menuID : undefined}
	data-tooltip="Server actions"
	data-tooltip-side="bottom-end"
	onclick={(event) => onMenu(event.currentTarget, true)}
	onkeydown={(event) => {
		if (
			event.key === "ArrowDown" ||
			event.key === "ContextMenu" ||
			(event.shiftKey && event.key === "F10")
		) {
			event.preventDefault()
			onMenu(event.currentTarget, false)
		}
	}}>
	<MoreHorizontal size={16} aria-hidden="true" />
</button>
