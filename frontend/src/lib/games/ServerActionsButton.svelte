<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import dotsThreeIcon from "@phosphor-icons/core/regular/dots-three.svg?raw"
	import Icon from "../shared/Icon.svelte"

	let {
		open,
		busy = false,
		menuID,
		onMenu,
	}: {
		open: boolean
		busy?: boolean
		menuID: string
		onMenu: (trigger: HTMLElement, toggle: boolean) => void
	} = $props()
</script>

<button
	class="sticky right-0 button-ghost justify-self-end bg-inherit"
	data-server-actions
	type="button"
	aria-label="Server actions"
	aria-haspopup="menu"
	aria-expanded={open}
	aria-controls={open ? menuID : undefined}
	aria-busy={busy}
	data-tooltip={busy ? "Loading launch link" : "Server actions"}
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
	{#if busy}
		<Icon svg={circleNotchIcon} class="spinner" size={16} aria-hidden="true" />
	{:else}
		<Icon svg={dotsThreeIcon} size={16} aria-hidden="true" />
	{/if}
</button>
