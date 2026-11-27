<script lang="ts">
	import caretUpIcon from "@phosphor-icons/core/regular/caret-up.svg?raw"
	import type { HTMLButtonAttributes } from "svelte/elements"
	import Icon from "./Icon.svelte"

	// SelectionActionsButton opens the menu of actions for the selected rows
	// of a list, from its selection toolbar.
	let {
		isOpen,
		onToggle,
		...rest
	}: {
		isOpen: boolean
		onToggle: (trigger: HTMLButtonElement) => void
	} & Omit<HTMLButtonAttributes, "onclick"> = $props()
</script>

<button
	class="button"
	type="button"
	aria-haspopup="menu"
	aria-expanded={isOpen}
	data-open={isOpen || undefined}
	{...rest}
	onclick={(event) => onToggle(event.currentTarget)}>
	Actions
	<Icon
		svg={caretUpIcon}
		class={["transition-transform duration-100", !isOpen && "rotate-180"]}
		size={14}
		aria-hidden="true" />
</button>
