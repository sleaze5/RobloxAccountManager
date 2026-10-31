<script lang="ts">
	import { outdated, versionTooltip } from "./game-servers-state.svelte"

	let { version, newestVersion }: { version: number; newestVersion: number } =
		$props()
	const old = $derived(outdated(version, newestVersion)),
		tooltip = $derived(versionTooltip(version, newestVersion))
</script>

{#if version}
	<span
		class="server-number justify-self-end text-right"
		aria-label={tooltip}
		data-tooltip={tooltip}
		>{#if old}<span class="server-tag">Old</span>{/if}{version}</span>
{:else}
	<span class="justify-self-end text-right text-fg-faint" aria-label="Version unknown"
		>-</span>
{/if}
