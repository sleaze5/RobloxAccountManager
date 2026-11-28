<script lang="ts">
	// SkeletonTile stands in for an item tile in a grid: a square image, a
	// name, and the type or price line below it when the real tile has one.
	let { hasDetailLine = true }: { hasDetailLine?: boolean } = $props()
</script>

<li class="flex flex-col gap-1.5">
	<span class="block aspect-square w-full skeleton rounded-lg"></span>
	<span class="flex flex-col">
		<span class="text-xs"><span class="skeleton-line w-3/4"></span></span>
		{#if hasDetailLine}
			<span class="text-2xs"><span class="skeleton-line w-1/2"></span></span>
		{/if}
	</span>
</li>
