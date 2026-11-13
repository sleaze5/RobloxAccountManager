<script lang="ts">
	import SkeletonList from "./SkeletonList.svelte"

	// SettingsRowsSkeleton stands in for the settings rows of a list while it
	// loads. Each row is as tall as a settings row with a title and a detail.
	let {
		label,
		rowCount = 4,
		hasIcon = true,
	}: { label: string; rowCount?: number; hasIcon?: boolean } = $props()

	const titleWidths = ["w-40", "w-32", "w-48", "w-36"],
		detailWidths = ["w-56", "w-44", "w-64"]
</script>

<SkeletonList {label} {rowCount} className="settings-rows">
	{#snippet row(index)}
		<div class="settings-row justify-start gap-3">
			{#if hasIcon}<span class="size-7 shrink-0 skeleton rounded-md"></span>{/if}
			<div class="settings-row-copy">
				<span class="text-sm"
					><span
						class={[
							"skeleton-line",
							titleWidths[index % titleWidths.length],
						]}></span
					></span>
				<span
					><span
						class={[
							"skeleton-line",
							detailWidths[index % detailWidths.length],
						]}></span
					></span>
			</div>
		</div>
	{/snippet}
</SkeletonList>
