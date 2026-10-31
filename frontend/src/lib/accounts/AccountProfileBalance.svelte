<script lang="ts">
	import type { AccountProfileSnapshot } from "../backend/bridge"

	let {
		snapshot,
		loading = false,
	}: {
		snapshot: AccountProfileSnapshot | null
		loading?: boolean
	} = $props()

	const exact = new Intl.NumberFormat("en-US"),
		compact = new Intl.NumberFormat("en-US", {
			notation: "compact",
			maximumFractionDigits: 1,
			minimumFractionDigits: 0,
		})

	function amount(value: number): string {
		return value >= 1_000 ? compact.format(value) : exact.format(value)
	}

	const robux = $derived(snapshot?.robux),
		pending = $derived(snapshot?.pendingRobux),
		unavailable = $derived(robux === null || robux === undefined),
		balance = $derived(
			robux === null || robux === undefined
				? loading
					? "…"
					: "Unavailable"
				: amount(robux),
		),
		tooltip = $derived(
			`Robux: ${
				robux === null || robux === undefined
					? loading
						? "Loading"
						: "Unavailable"
					: exact.format(robux)
			}` +
				(pending !== null && pending !== undefined && pending > 0
					? `\nPending Robux: ${exact.format(pending)}`
					: ""),
		)
</script>

<button
	type="button"
	class={[
		"inline-flex shrink-0 cursor-help items-center rounded-md font-mono text-xs whitespace-pre tabular-nums focus-visible:outline-offset-2",
		unavailable ? "text-fg-faint" : "text-fg",
	]}
	aria-label={tooltip}
	data-tooltip={tooltip}>
	R$ {balance}{#if pending !== null && pending !== undefined && pending > 0}<span
			class="text-fg-muted">{` (+${amount(pending)})`}</span
		>{/if}
</button>
