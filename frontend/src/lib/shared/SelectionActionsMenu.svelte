<script lang="ts">
	import AnchoredActionMenu from "./AnchoredActionMenu.svelte"
	import type { MenuAction } from "./AnchoredActionMenu.svelte"
	import SelectionActionsButton from "./SelectionActionsButton.svelte"

	// SelectionActionsMenu holds the actions for the selected rows of a list.
	// noun names the rows, such as "games".
	let {
		noun,
		selectedCount,
		actions,
		isBusy = false,
	}: {
		noun: string
		selectedCount: number
		actions: MenuAction[]
		isBusy?: boolean
	} = $props()
	const menuId = $props.id()
	let anchor = $state<HTMLButtonElement | null>(null)

	$effect(() => {
		if (selectedCount === 0 || isBusy) anchor = null
	})

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

<SelectionActionsButton
	isOpen={!!anchor}
	aria-controls={anchor ? menuId : undefined}
	aria-busy={isBusy}
	disabled={selectedCount === 0 || isBusy}
	onToggle={(trigger) => (anchor = anchor ? null : trigger)} />

{#if anchor}
	<AnchoredActionMenu
		id={menuId}
		label={`Actions for the selected ${noun}`}
		{anchor}
		actions={actions.map(closeAfter)}
		selection={{ count: selectedCount, noun }}
		onClose={() => (anchor = null)} />
{/if}
