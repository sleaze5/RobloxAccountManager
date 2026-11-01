<script lang="ts">
	import dotsThreeIcon from "@phosphor-icons/core/regular/dots-three.svg?raw"
	import Icon from "../shared/Icon.svelte"

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
	class="sticky right-0 btn-ghost justify-self-end bg-inherit"
	data-server-actions
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
	<Icon svg={dotsThreeIcon} size={16} aria-hidden="true" />
</button>
