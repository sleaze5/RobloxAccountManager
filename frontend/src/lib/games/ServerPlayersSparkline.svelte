<script lang="ts">
	// ServerPlayersSparkline draws the player count of each time that this app
	// listed a server, scaled to the server's maximum.
	let {
		history,
		maxPlayers,
	}: {
		history: number[]
		maxPlayers: number
	} = $props()

	const width = 56,
		height = 16,
		inset = 2
	const ceiling = $derived(Math.max(maxPlayers, ...history, 1)),
		points = $derived(
			history.map((playing, index) => ({
				x:
					inset +
					(index / Math.max(history.length - 1, 1)) * (width - inset * 2),
				y: height - inset - (playing / ceiling) * (height - inset * 2),
			})),
		),
		last = $derived(points.at(-1)),
		label = $derived(
			`Players in the last ${history.length} listings: ${history.join(", ")}`,
		)
</script>

<svg
	class="shrink-0 overflow-visible text-fg-muted"
	{width}
	{height}
	viewBox={`0 0 ${width} ${height}`}
	role="img"
	aria-label={label}>
	<line
		x1={inset}
		x2={width - inset}
		y1={height - inset}
		y2={height - inset}
		class="stroke-line-strong"
		stroke-width="1" />
	<polyline
		points={points.map((point) => `${point.x},${point.y}`).join(" ")}
		fill="none"
		stroke="currentColor"
		stroke-width="1.5"
		stroke-linecap="round"
		stroke-linejoin="round" />
	{#if last}
		<circle cx={last.x} cy={last.y} r="2" class="fill-fg" />
	{/if}
</svg>
