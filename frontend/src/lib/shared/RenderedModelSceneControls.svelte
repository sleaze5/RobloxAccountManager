<script lang="ts">
	import {
		defaultRenderedModelScene,
		renderedModelScene,
		resetRenderedModelScene,
	} from "./rendered-model-scene.svelte"
	import type { RenderedModelScene } from "./rendered-model-scene.svelte"

	let { id, onClose }: { id: string; onClose: () => void } = $props()

	let panel = $state<HTMLDivElement>()

	const percent = new Intl.NumberFormat("en-US", { style: "percent" }),
		formatDegrees = (degrees: number): string => `${degrees}°`,
		sliders: {
			setting: keyof RenderedModelScene
			label: string
			min: number
			max: number
			step: number
			format: (value: number) => string
		}[] = [
			{
				setting: "fieldOfViewDegrees",
				label: "Field of view",
				min: 10,
				max: 70,
				step: 1,
				format: formatDegrees,
			},
			{
				setting: "lightIntensity",
				label: "Light",
				min: 0,
				max: 2.5,
				step: 0.05,
				format: (share) => percent.format(share),
			},
			{
				setting: "brightness",
				label: "Brightness",
				min: 0,
				max: 2.5,
				step: 0.05,
				format: (share) => percent.format(share),
			},
			{
				setting: "lightAngleDegrees",
				label: "Light angle",
				min: -180,
				max: 180,
				step: 5,
				format: formatDegrees,
			},
		],
		isDefault = $derived(
			sliders.every(
				({ setting }) =>
					renderedModelScene[setting] === defaultRenderedModelScene[setting],
			),
		)
</script>

<!-- Escape closes the panel before it reaches a dialog around the viewer. -->
<svelte:window
	onkeydowncapture={(event) => {
		if (
			event.key !== "Escape" ||
			!(event.target instanceof Node) ||
			!panel?.contains(event.target)
		) {
			return
		}
		event.preventDefault()
		event.stopImmediatePropagation()
		onClose()
	}} />

<div
	bind:this={panel}
	{id}
	class="absolute top-11 right-2 z-10 flex w-56 animate-enter flex-col gap-1.5 rounded-lg border border-line-strong bg-overlay p-3 shadow-float"
	role="group"
	aria-label="Camera and lighting"
	data-closes-on-escape>
	{#each sliders as slider (slider.setting)}
		{@const value = renderedModelScene[slider.setting]}
		<label class="flex flex-col gap-0.5 text-xs">
			<span class="flex items-center justify-between gap-2">
				<span class="text-fg-muted">{slider.label}</span>
				<span class="font-mono text-2xs text-fg">{slider.format(value)}</span>
			</span>
			<input
				class="range"
				type="range"
				min={slider.min}
				max={slider.max}
				step={slider.step}
				{value}
				style:--range-fill={`${((value - slider.min) / (slider.max - slider.min)) * 100}%`}
				aria-valuetext={slider.format(value)}
				oninput={(event) =>
					(renderedModelScene[slider.setting] =
						event.currentTarget.valueAsNumber)} />
		</label>
	{/each}
	<button
		class="mt-1 button self-end"
		type="button"
		disabled={isDefault}
		onclick={resetRenderedModelScene}>Reset</button>
</div>
