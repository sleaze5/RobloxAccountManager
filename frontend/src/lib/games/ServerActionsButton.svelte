<script lang="ts">
	import DotsThreeIcon from "phosphor-svelte/lib/DotsThreeIcon"

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
	<DotsThreeIcon size={18} aria-hidden="true" />
</button>
