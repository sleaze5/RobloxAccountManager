<script lang="ts">
	// SkeletonMediaRow stands in for a list row with an image, a name, and a
	// second line. The widths change from row to row so that the rows read as
	// text.
	let {
		index,
		className = "h-12",
		imageClass = "size-9 rounded-md",
	}: { index: number; className?: string; imageClass?: string } = $props()

	const nameWidths = ["w-32", "w-24", "w-36", "w-28"],
		detailWidths = ["w-20", "w-16", "w-24"]
</script>

<div
	class={[
		"flex w-full items-center gap-2 border-b border-line-subtle px-3",
		className,
	]}>
	<span class={["shrink-0 skeleton", imageClass]}></span>
	<span class="flex min-w-0 flex-col gap-1">
		<span class="text-sm"
			><span class={["skeleton-line", nameWidths[index % nameWidths.length]]}
			></span
			></span>
		<span class="text-2xs"
			><span class={["skeleton-line", detailWidths[index % detailWidths.length]]}
			></span
			></span>
	</span>
</div>
