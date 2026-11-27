<script lang="ts">
	import clipboardTextIcon from "@phosphor-icons/core/regular/clipboard-text.svg?raw"
	import fileArrowDownIcon from "@phosphor-icons/core/regular/file-arrow-down.svg?raw"
	import globeHemisphereWestIcon from "@phosphor-icons/core/regular/globe-hemisphere-west.svg?raw"
	import MenuItem from "../shared/MenuItem.svelte"
	import { floatMenu, menuIn, menuOut } from "../shared/presence"

	let {
		disabled,
		anchor,
		onPasteCookie,
		onBrowser,
		onImportFile,
	}: {
		disabled: boolean
		anchor?: HTMLElement
		onPasteCookie: () => void
		onBrowser: () => void
		onImportFile: () => void
	} = $props()
</script>

<div
	class={["menu", !anchor && "absolute top-full right-0 z-10 mt-0.5"]}
	data-account-menu
	role="menu"
	aria-label="Add account"
	tabindex="-1"
	use:floatMenu={{ anchor, align: "center" }}
	in:menuIn
	out:menuOut>
	<MenuItem
		label="Sign in with browser"
		icon={globeHemisphereWestIcon}
		{disabled}
		onSelect={onBrowser} />
	<MenuItem
		label="Paste cookies"
		icon={clipboardTextIcon}
		{disabled}
		onSelect={onPasteCookie} />
	<MenuItem
		label="Import data"
		icon={fileArrowDownIcon}
		{disabled}
		onSelect={onImportFile} />
</div>
