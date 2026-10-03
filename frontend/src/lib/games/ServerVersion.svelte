<script lang="ts">
	import { outdated, versionTooltip } from "./game-servers-state.svelte"

	let { version, newestVersion }: { version: number; newestVersion: number } =
		$props()
	const old = $derived(outdated(version, newestVersion)),
		tooltip = $derived(versionTooltip(version, newestVersion))
</script>

{#if version}
	<span class="game-server-number numeric" aria-label={tooltip} data-tooltip={tooltip}
		>{#if old}<span class="game-server-tag">Old</span>{/if}{version}</span>
{:else}
	<span class="game-server-faint numeric" aria-label="Version unknown">—</span>
{/if}
