<script lang="ts">
	import clipboardTextIcon from "@phosphor-icons/core/regular/clipboard-text.svg?raw"
	import fileArrowDownIcon from "@phosphor-icons/core/regular/file-arrow-down.svg?raw"
	import globeHemisphereWestIcon from "@phosphor-icons/core/regular/globe-hemisphere-west.svg?raw"
	import Icon from "../shared/Icon.svelte"
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
	use:floatMenu={{ anchor, align: "center" }}
	in:menuIn
	out:menuOut>
	<button class="menu-item" type="button" {disabled} onclick={onBrowser}>
		<Icon svg={globeHemisphereWestIcon} size={16} aria-hidden="true" />
		<span>Sign in with browser</span>
	</button>
	<button class="menu-item" type="button" {disabled} onclick={onPasteCookie}>
		<Icon svg={clipboardTextIcon} size={16} aria-hidden="true" />
		<span>Paste cookie</span>
	</button>
	<button class="menu-item" type="button" {disabled} onclick={onImportFile}>
		<Icon svg={fileArrowDownIcon} size={16} aria-hidden="true" />
		<span>Import export file</span>
	</button>
</div>
