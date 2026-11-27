<script lang="ts">
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import type { Snippet } from "svelte"
	import CheckboxMark from "./CheckboxMark.svelte"
	import Icon from "./Icon.svelte"

	// SelectionToolbar docks at the bottom of a list in selection mode. It
	// selects every shown row, counts the selected rows, and holds the
	// actions for them.
	let {
		label,
		selectedCount,
		isAllShownSelected,
		canSelectAll,
		isBusy = false,
		onToggleAll,
		onStop,
		children,
	}: {
		label: string
		selectedCount: number
		isAllShownSelected: boolean
		canSelectAll: boolean
		isBusy?: boolean
		onToggleAll: () => void
		onStop: () => void
		children: Snippet
	} = $props()
	const isPartlySelected = $derived(selectedCount > 0 && !isAllShownSelected)
</script>

<div
	class="absolute inset-x-0 bottom-0 flex h-11 animate-enter items-center gap-2 border-t border-line bg-overlay pr-2 pl-3"
	role="toolbar"
	aria-label={label}>
	<span class="checkbox shrink-0">
		<input
			type="checkbox"
			aria-label={isAllShownSelected ? "Clear selection" : "Select all shown"}
			data-tooltip={isAllShownSelected ? "Clear selection" : "Select all shown"}
			checked={isAllShownSelected}
			indeterminate={isPartlySelected}
			disabled={!canSelectAll || isBusy}
			onchange={onToggleAll} />
		<CheckboxMark isIndeterminate={isPartlySelected} />
	</span>
	<span class="min-w-0 flex-1 truncate text-xs text-fg-muted" aria-live="polite"
		><strong class="font-mono font-medium text-fg">{selectedCount}</strong>
		selected</span>
	{@render children()}
	<button
		class="button-ghost-sm"
		type="button"
		aria-label="Stop selecting"
		data-tooltip="Stop selecting"
		disabled={isBusy}
		onclick={onStop}>
		<Icon svg={xIcon} size={14} aria-hidden="true" />
	</button>
</div>
