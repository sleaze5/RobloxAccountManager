<script lang="ts">
	import listChecksIcon from "@phosphor-icons/core/regular/list-checks.svg?raw"
	import Icon from "./Icon.svelte"

	// SelectionModeButton turns selection mode of a list on and off from the
	// header of its sidebar. noun names the rows, such as "friends".
	let {
		noun,
		isSelecting,
		disabled = false,
		onToggle,
	}: {
		noun: string
		isSelecting: boolean
		disabled?: boolean
		onToggle: () => void
	} = $props()
</script>

<button
	class="button-ghost"
	type="button"
	aria-label={`Select ${noun}`}
	aria-pressed={isSelecting}
	data-open={isSelecting || undefined}
	data-tooltip={isSelecting ? "Stop selecting" : `Select ${noun}`}
	data-tooltip-side="bottom-end"
	{disabled}
	onclick={onToggle}>
	<Icon svg={listChecksIcon} size={16} aria-hidden="true" />
</button>
