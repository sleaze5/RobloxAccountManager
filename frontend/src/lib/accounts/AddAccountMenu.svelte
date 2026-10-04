<script lang="ts">
	import ClipboardTextIcon from "phosphor-svelte/lib/ClipboardTextIcon"
	import GlobeHemisphereWestIcon from "phosphor-svelte/lib/GlobeHemisphereWestIcon"
	import { menuIn, menuOut } from "../shared/presence"

	let {
		disabled,
		onPasteCookie,
		onBrowser,
	}: { disabled: boolean; onPasteCookie: () => void; onBrowser: () => void } =
		$props()
</script>

<div class="add-account-menu account-menu" data-account-menu in:menuIn out:menuOut>
	<button type="button" {disabled} onclick={onPasteCookie}>
		<ClipboardTextIcon size={16} aria-hidden="true" />
		<span>Paste cookie</span>
	</button>
	<button type="button" {disabled} onclick={onBrowser}>
		<GlobeHemisphereWestIcon size={16} aria-hidden="true" />
		<span>Browser</span>
	</button>
</div>
