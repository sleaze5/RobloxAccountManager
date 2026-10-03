<script lang="ts">
	import ClipboardPaste from "@lucide/svelte/icons/clipboard-paste"
	import Globe2 from "@lucide/svelte/icons/earth"
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
		<ClipboardPaste size={14} />
		<span>Paste cookie</span>
	</button>
	<button type="button" {disabled} onclick={onBrowser}>
		<Globe2 size={14} />
		<span>Browser</span>
	</button>
</div>
