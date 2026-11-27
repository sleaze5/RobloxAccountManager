<script lang="ts">
	import dotsThreeIcon from "@phosphor-icons/core/regular/dots-three.svg?raw"
	import AnchoredActionMenu from "./AnchoredActionMenu.svelte"
	import type { MenuAction } from "./AnchoredActionMenu.svelte"
	import Icon from "./Icon.svelte"

	// MoreActionsMenu is the More actions button at the top right of a profile
	// or a details dialog, with the menu of its less frequent actions, such as
	// Open in browser and the values to copy. label names the menu, such as
	// "Brick actions".
	let { label, actions }: { label: string; actions: MenuAction[] } = $props()
	const menuId = $props.id()
	let anchor = $state<HTMLButtonElement | null>(null)

	function closeAfter(action: MenuAction): MenuAction {
		return {
			...action,
			onSelect: () => {
				anchor = null
				action.onSelect()
			},
		}
	}
</script>

<button
	class="button-icon"
	type="button"
	aria-label="More actions"
	aria-haspopup="menu"
	aria-expanded={!!anchor}
	aria-controls={anchor ? menuId : undefined}
	data-open={anchor ? true : undefined}
	data-tooltip="More actions"
	onclick={(event) => (anchor = anchor ? null : event.currentTarget)}>
	<Icon svg={dotsThreeIcon} size={16} aria-hidden="true" />
</button>

{#if anchor}
	<AnchoredActionMenu
		id={menuId}
		{label}
		{anchor}
		actions={actions.map(closeAfter)}
		onClose={() => (anchor = null)} />
{/if}
