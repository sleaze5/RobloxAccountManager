<script lang="ts">
	import { appSettings } from "../settings/settings-store.svelte"
	import { formatTimestamp } from "./timestamp"

	let { value }: { value: number } = $props()

	const dateTime = $derived(new Date(value).toISOString()),
		shown = $derived(formatTimestamp(value, appSettings.timestampFormat)),
		tooltip = $derived(formatTimestamp(value, appSettings.timestampHoverFormat))

	function refreshTooltip(event: Event): void {
		if (!(event.currentTarget instanceof HTMLElement)) {
			return
		}

		const text = formatTimestamp(value, appSettings.timestampHoverFormat)
		event.currentTarget.dataset.tooltip = text
		event.currentTarget.ariaLabel = text
	}
</script>

<button
	class="timestamp"
	type="button"
	aria-label={tooltip}
	data-tooltip={tooltip}
	data-tooltip-side="top"
	onpointerover={refreshTooltip}
	onfocus={refreshTooltip}>
	<time datetime={dateTime}>{shown}</time>
</button>

<style>
	.timestamp {
		padding: 0;
		border: 0;
		border-radius: var(--radius-control);
		background: transparent;
		color: var(--color-text-muted);
		font: inherit;
		font-weight: var(--font-medium);
		cursor: help;
		transition: color var(--duration-fast) var(--ease-standard);
	}

	.timestamp:hover,
	.timestamp:focus-visible {
		color: var(--color-text);
	}

	.timestamp:focus-visible {
		outline: 1px solid var(--color-focus-ring);
		outline-offset: 2px;
	}
</style>
