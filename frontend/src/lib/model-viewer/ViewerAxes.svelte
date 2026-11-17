<script lang="ts">
	import type { CameraPose } from "./model-subject"

	// ViewerAxes shows where the X, Y, and Z axes of the model point on
	// screen, as short rods with a ball at the positive end and a faint rod
	// toward the negative end, so that an axis that points at the camera still
	// shows as a dot.
	let { pose }: { pose: CameraPose } = $props()

	const axisLength = 1,
		negativeAxisLength = 0.6,
		labelDistance = 1.38

	// Each axis in the space of the camera is the matching row of the rotation
	// matrix of the camera. The screen points Y down, and positive depth points
	// at the viewer, so the nearer axes draw last.
	const axes = $derived.by(() => {
		const [x, y, z, w] = pose.orientation,
			rows = [
				[1 - 2 * (y * y + z * z), 2 * (x * y - z * w), 2 * (x * z + y * w)],
				[2 * (x * y + z * w), 1 - 2 * (x * x + z * z), 2 * (y * z - x * w)],
				[2 * (x * z - y * w), 2 * (y * z + x * w), 1 - 2 * (x * x + y * y)],
			]
		return (["X", "Y", "Z"] as const)
			.map((name, index) => {
				const [screenX = 0, screenY = 0, depth = 0] = rows[index] ?? []
				return {
					name,
					screenX,
					screenY: -screenY,
					depth,
					// A label fades as its axis turns toward or away from the
					// camera, because it would sit on the center.
					labelOpacity: Math.min(
						Math.max((Math.hypot(screenX, screenY) - 0.2) / 0.3, 0),
						1,
					),
				}
			})
			.sort((first, second) => first.depth - second.depth)
	})
</script>

<svg
	class="pointer-events-none absolute bottom-1 left-1 size-14"
	viewBox="-1.6 -1.6 3.2 3.2"
	aria-hidden="true">
	{#each axes as axis (axis.name)}
		<g
			class={[
				axis.name === "X" && "fill-axis-x stroke-axis-x",
				axis.name === "Y" && "fill-axis-y stroke-axis-y",
				axis.name === "Z" && "fill-axis-z stroke-axis-z",
			]}>
			<line
				x1="0"
				y1="0"
				x2={-axis.screenX * negativeAxisLength}
				y2={-axis.screenY * negativeAxisLength}
				stroke-width="0.08"
				stroke-linecap="round"
				opacity="0.35" />
			<line
				x1="0"
				y1="0"
				x2={axis.screenX * axisLength}
				y2={axis.screenY * axisLength}
				stroke-width="0.12"
				stroke-linecap="round" />
			<circle
				cx={axis.screenX * axisLength}
				cy={axis.screenY * axisLength}
				r="0.14"
				stroke="none" />
			<text
				class="font-mono font-semibold"
				x={axis.screenX * labelDistance}
				y={axis.screenY * labelDistance}
				font-size="0.6"
				text-anchor="middle"
				dominant-baseline="central"
				stroke="none"
				opacity={axis.labelOpacity}>{axis.name}</text>
		</g>
	{/each}
</svg>
