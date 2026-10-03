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
	class="profile-balance"
	class:profile-value-unavailable={unavailable}
	aria-label={tooltip}
	data-tooltip={tooltip}>
	R$ {balance}{#if pending !== null && pending !== undefined && pending > 0}<span
			class="profile-balance-pending">{` (+${amount(pending)})`}</span
		>{/if}
</button>
