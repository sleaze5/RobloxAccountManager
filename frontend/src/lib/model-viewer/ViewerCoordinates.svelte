<script lang="ts">
	import type { CameraPose } from "./model-subject"

	// ViewerCoordinates shows where the camera stands, where it looks, and how
	// far apart they are, in studs, in three rows as tall as the axes.
	let { pose }: { pose: CameraPose } = $props()

	const studs = new Intl.NumberFormat("en-US", {
		minimumFractionDigits: 1,
		maximumFractionDigits: 1,
	})

	// Each row names its point in short, as technical readouts do, and in
	// full for screen readers.
	const rows = $derived([
		{ label: "Camera", shortLabel: "cam", point: pose.position },
		{ label: "Target", shortLabel: "tgt", point: pose.target },
	])
	const distance = $derived(
		Math.hypot(
			pose.position[0] - pose.target[0],
			pose.position[1] - pose.target[1],
			pose.position[2] - pose.target[2],
		),
	)
</script>

<dl
	class="flex animate-enter flex-col border-l border-line-subtle px-1.5"
	role="group"
	aria-label="Camera coordinates in studs">
	{#each rows as row (row.label)}
		<div class="flex items-baseline gap-1.5">
			<dt class="w-7 text-fg-faint">
				<span aria-hidden="true">{row.shortLabel}</span>
				<span class="sr-only">{row.label}</span>
			</dt>
			{#each row.point as coordinate, axisIndex (axisIndex)}
				<dd
					class={[
						"w-9 text-right",
						axisIndex === 0 && "text-axis-x",
						axisIndex === 1 && "text-axis-y",
						axisIndex === 2 && "text-axis-z",
					]}>
					{studs.format(coordinate)}
				</dd>
			{/each}
		</div>
	{/each}
	<div class="flex items-baseline gap-1.5">
		<dt class="w-7 text-fg-faint">
			<span aria-hidden="true">dist</span>
			<span class="sr-only">Distance</span>
		</dt>
		<dd class="text-fg">{studs.format(distance)}</dd>
	</div>
</dl>
