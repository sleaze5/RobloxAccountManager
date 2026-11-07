<script lang="ts">
	import arrowSquareOutIcon from "@phosphor-icons/core/regular/arrow-square-out.svg?raw"
	import { Browser } from "@wailsio/runtime"
	import Icon from "../shared/Icon.svelte"

	let { label, url }: { label: string; url: string } = $props()
</script>

<button
	class="-my-0.5 button-ghost-sm"
	type="button"
	aria-label={label}
	data-tooltip={label}
	data-tooltip-side="bottom-end"
	onclick={() => void Browser.OpenURL(url)}>
	<Icon svg={arrowSquareOutIcon} size={14} aria-hidden="true" />
</button>
