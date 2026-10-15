<script lang="ts">
	import ClipboardTextIcon from "phosphor-svelte/lib/ClipboardTextIcon"
	import GlobeHemisphereWestIcon from "phosphor-svelte/lib/GlobeHemisphereWestIcon"
	import { floatMenu, menuIn, menuOut } from "../shared/presence"

	let {
		disabled,
		anchor,
		onPasteCookie,
		onBrowser,
	}: {
		disabled: boolean
		anchor?: HTMLElement
		onPasteCookie: () => void
		onBrowser: () => void
	} = $props()
</script>

<div
	class="add-account-menu account-menu"
	data-account-menu
	use:floatMenu={{ anchor, align: "center" }}
	in:menuIn
	out:menuOut>
	<button type="button" {disabled} onclick={onBrowser}>
		<GlobeHemisphereWestIcon size={16} aria-hidden="true" />
		<span>Sign in with browser</span>
	</button>
	<button type="button" {disabled} onclick={onPasteCookie}>
		<ClipboardTextIcon size={16} aria-hidden="true" />
		<span>Paste cookie</span>
	</button>
</div>
