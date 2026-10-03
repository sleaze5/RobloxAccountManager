<script lang="ts">
	import { appSettings } from "../settings/settings-store.svelte"
	import { formatTimestamp } from "../shared/timestamp"
	import { formatUptime } from "./game-servers-state.svelte"

	let { firstSeenMs }: { firstSeenMs: number } = $props()
	const uptime = $derived(formatUptime(firstSeenMs)),
		tooltip = $derived(
			`Up about ${uptime}. First seen by RoValra ${formatTimestamp(firstSeenMs, appSettings.timestampHoverFormat)}.`,
		)
</script>

{#if firstSeenMs}
	<span class="game-server-number numeric" aria-label={tooltip} data-tooltip={tooltip}
		>{uptime}</span>
{:else}
	<span class="game-server-faint numeric" aria-label="Uptime unknown">—</span>
{/if}
