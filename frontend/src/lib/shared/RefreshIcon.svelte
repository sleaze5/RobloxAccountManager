<script lang="ts">
	import arrowsClockwiseIcon from "@phosphor-icons/core/regular/arrows-clockwise.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import Icon from "./Icon.svelte"

	let { isRefreshing, size = 16 }: { isRefreshing: boolean; size?: number } = $props()
</script>

{#if isRefreshing}
	<Icon svg={circleNotchIcon} class="spinner" {size} aria-hidden="true" />
{:else}
	<Icon svg={arrowsClockwiseIcon} data-turn {size} aria-hidden="true" />
{/if}
