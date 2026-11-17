<script lang="ts">
	import type { CameraPose } from "./model-subject"

	// ViewerCoordinates shows where the camera stands, where it looks, and how
	// far apart they are, in studs.
	let { pose }: { pose: CameraPose } = $props()

	const studs = new Intl.NumberFormat("en-US", {
		minimumFractionDigits: 1,
		maximumFractionDigits: 1,
	})

	const rows = $derived([
		{ label: "Camera", point: pose.position },
		{ label: "Target", point: pose.target },
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
	class="pointer-events-none absolute bottom-2 left-16 flex animate-enter flex-col gap-0.5 rounded-md border border-line-strong bg-overlay px-1.5 py-1 font-mono text-2xs text-fg-muted tabular-nums"
	role="group"
	aria-label="Camera coordinates in studs">
	{#each rows as row (row.label)}
		<div class="flex items-baseline gap-1.5">
			<dt class="w-12 font-sans font-medium text-fg">{row.label}</dt>
			{#each row.point as coordinate, axisIndex (axisIndex)}
				<dd
					class={[
						"w-10 text-right",
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
		<dt class="w-12 font-sans font-medium text-fg">Distance</dt>
		<dd>{studs.format(distance)} studs</dd>
	</div>
</dl>
