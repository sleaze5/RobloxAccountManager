<script lang="ts">
	let { value, unit = "" }: { value: number; unit?: string } = $props()

	const compact = new Intl.NumberFormat("en-US", {
			notation: "compact",
			maximumFractionDigits: 1,
		}),
		exact = new Intl.NumberFormat()

	const tooltip = $derived(`${exact.format(value)}${unit ? ` ${unit}` : ""}`)
</script>

<button
	class="cursor-help rounded-md tabular-nums focus-visible:outline-offset-2"
	type="button"
	aria-label={tooltip}
	data-tooltip={tooltip}
	data-tooltip-side="top">{compact.format(value)}</button>
