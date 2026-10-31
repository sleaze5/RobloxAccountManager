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
	class="absolute top-full right-0 z-10 mt-0.5 menu w-48"
	data-account-menu
	use:floatMenu={{ anchor, align: "center" }}
	in:menuIn
	out:menuOut>
	<button class="menu-item" type="button" {disabled} onclick={onBrowser}>
		<GlobeHemisphereWestIcon size={16} aria-hidden="true" />
		<span>Sign in with browser</span>
	</button>
	<button class="menu-item" type="button" {disabled} onclick={onPasteCookie}>
		<ClipboardTextIcon size={16} aria-hidden="true" />
		<span>Paste cookie</span>
	</button>
</div>
