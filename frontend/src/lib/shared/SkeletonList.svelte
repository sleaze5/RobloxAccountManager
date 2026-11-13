<script lang="ts">
	import type { Snippet } from "svelte"

	// SkeletonList stands in for the rows of a list while its first page loads,
	// so that the list does not jump when the rows arrive.
	let {
		label,
		rowCount = 8,
		className = "",
		row,
	}: {
		label: string
		rowCount?: number
		className?: string
		row: Snippet<[number]>
	} = $props()

	const rowIndexes = $derived(
		Array.from({ length: rowCount }, (_unused, index) => index),
	)
</script>

<div class={className} role="status" aria-busy="true">
	<span class="sr-only">{label}</span>
	<div aria-hidden="true">
		{#each rowIndexes as index (index)}{@render row(index)}{/each}
	</div>
</div>
