<script lang="ts">
	import checkCircleIcon from "@phosphor-icons/core/regular/check-circle.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import warningCircleIcon from "@phosphor-icons/core/regular/warning-circle.svg?raw"
	import xCircleIcon from "@phosphor-icons/core/regular/x-circle.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { CleanPhaseProgress } from "./roblox-cleaning"

	// hasFailed marks the running phase of a run that stopped with an error.
	let {
		phase,
		hasFailed,
		size = 16,
	}: { phase: CleanPhaseProgress; hasFailed: boolean; size?: number } = $props()
</script>

{#if hasFailed}
	<Icon
		svg={xCircleIcon}
		{size}
		class="shrink-0 animate-pop text-danger"
		aria-hidden="true" />
	<span class="sr-only">Failed:</span>
{:else if phase.state === "running"}
	<Icon svg={circleNotchIcon} {size} class="shrink-0 spinner" aria-hidden="true" />
	<span class="sr-only">In progress:</span>
{:else if phase.state === "waiting"}
	<Icon
		svg={circleNotchIcon}
		{size}
		class="shrink-0 text-fg-muted"
		aria-hidden="true" />
	<span class="sr-only">Waiting:</span>
{:else if phase.skippedCount > 0}
	<Icon
		svg={warningCircleIcon}
		{size}
		class="shrink-0 animate-pop text-warning"
		aria-hidden="true" />
	<span class="sr-only">Done with skipped items:</span>
{:else}
	<Icon
		svg={checkCircleIcon}
		{size}
		class="shrink-0 animate-bloom text-success"
		aria-hidden="true" />
	<span class="sr-only">Done:</span>
{/if}
