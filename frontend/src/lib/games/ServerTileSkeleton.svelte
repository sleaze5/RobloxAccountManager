<script lang="ts">
	import SkeletonList from "../shared/SkeletonList.svelte"

	// ServerTileSkeleton stands in for a list of server tiles while its first
	// page loads. Each placeholder has the parts of a tile, so the list does not
	// move when the tiles arrive. A recorded server has fewer, wider numbers and
	// no players row.
	let { label, isRecorded = false }: { label: string; isRecorded?: boolean } =
		$props()

	const titleWidths = ["w-36", "w-28", "w-40"],
		statSpans = $derived(isRecorded ? [1, 1, 2, 3] : [2, 1, 1, 1, 1, 1, 1])
</script>

<div class="min-h-0 flex-1 overflow-hidden">
	<p class="px-3 pt-2 text-2xs"><span class="skeleton-line w-56"></span></p>
	<SkeletonList {label} rowCount={4} className="p-3">
		{#snippet row(index)}
			<div class="mb-2 server-tile hover:border-line-subtle">
				<div class="flex h-6 items-center gap-3">
					<span class="text-sm"
						><span
							class={[
								"skeleton-line",
								titleWidths[index % titleWidths.length],
							]}></span
						></span>
					<span class="text-2xs"
						><span class="skeleton-line w-40"></span></span>
				</div>
				<div class="server-stats">
					{#each statSpans as span, stat (stat)}
						<div
							class={{
								"col-span-2": span === 2,
								"col-span-3": span === 3,
							}}>
							<span class="text-2xs"
								><span class="skeleton-line w-10"></span></span>
							<span class="text-sm"
								><span class="skeleton-line w-12"></span></span>
						</div>
					{/each}
				</div>
				{#if !isRecorded}
					<div
						class="flex min-h-6 items-center border-t border-line-subtle pt-2 text-2xs">
						<span class="skeleton-line w-24"></span>
						<span class="ml-auto skeleton-line w-52"></span>
					</div>
				{/if}
			</div>
		{/snippet}
	</SkeletonList>
</div>
