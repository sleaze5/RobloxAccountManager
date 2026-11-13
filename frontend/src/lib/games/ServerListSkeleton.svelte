<script lang="ts">
	import SkeletonList from "../shared/SkeletonList.svelte"

	// ServerListSkeleton stands in for a server list while its first page
	// loads. The rows are as tall as server rows, so the list does not move
	// when they arrive.
	let { label }: { label: string } = $props()

	const cellWidths = [
		["w-16", "w-32", "w-12", "w-24"],
		["w-12", "w-24", "w-16", "w-28"],
		["w-16", "w-28", "w-10", "w-20"],
	]
</script>

<div class="min-h-0 flex-1 overflow-hidden">
	<p class="px-3 pt-2 text-2xs"><span class="skeleton-line w-40"></span></p>
	<SkeletonList {label} rowCount={10}>
		{#snippet row(index)}
			<div
				class="flex h-11 items-center gap-6 border-b border-line-subtle px-3 text-2xs">
				{#each cellWidths[index % cellWidths.length] as width, cell (cell)}
					<span class={["skeleton-line", width]}></span>
				{/each}
			</div>
		{/snippet}
	</SkeletonList>
</div>
