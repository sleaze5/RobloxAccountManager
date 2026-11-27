<script lang="ts">
	import caretRightIcon from "@phosphor-icons/core/regular/caret-right.svg?raw"
	import Icon from "./Icon.svelte"
	import type { MenuItemTone } from "./menu-tone"

	// detail shows the value that the item acts on, such as the user ID that
	// it copies, in technical text after the label. hasSubmenu marks an item
	// that opens a submenu beside it on hover or focus, instead of acting.
	let {
		label,
		icon,
		tone,
		detail = "",
		accessibleLabel,
		tooltip,
		disabled = false,
		hasSubmenu = false,
		onSelect,
	}: {
		label: string
		icon: string
		tone?: MenuItemTone
		detail?: string
		accessibleLabel?: string
		tooltip?: string
		disabled?: boolean
		hasSubmenu?: boolean
		onSelect?: () => void
	} = $props()
</script>

<button
	class={[
		"group/menu-item",
		tone === "danger"
			? "menu-item-danger"
			: tone === "warning"
				? "menu-item-warning"
				: "menu-item",
	]}
	type="button"
	role="menuitem"
	aria-label={accessibleLabel}
	aria-haspopup={hasSubmenu ? "menu" : undefined}
	data-tooltip={tooltip}
	{disabled}
	onclick={onSelect}>
	<Icon svg={icon} size={16} aria-hidden="true" />
	<span class="shrink-0">{label}</span>
	{#if hasSubmenu}
		<Icon
			svg={caretRightIcon}
			data-nudge="right"
			size={16}
			class="ml-auto shrink-0"
			aria-hidden="true" />
	{:else if detail}
		<small
			class="ml-auto min-w-0 truncate pl-2 font-mono text-2xs font-normal text-fg-faint transition-colors duration-100 group-hover/menu-item:text-fg-muted"
			>{detail}</small>
	{/if}
</button>
