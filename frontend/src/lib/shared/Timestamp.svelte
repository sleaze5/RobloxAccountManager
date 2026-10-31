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
	class="cursor-help rounded-md font-medium text-fg-muted transition-colors duration-100 hover:text-fg focus-visible:text-fg focus-visible:outline-offset-2"
	type="button"
	aria-label={tooltip}
	data-tooltip={tooltip}
	data-tooltip-side="top"
	onpointerover={refreshTooltip}
	onfocus={refreshTooltip}>
	<time datetime={dateTime}>{shown}</time>
</button>
