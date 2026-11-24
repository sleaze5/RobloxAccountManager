<script lang="ts">
	import {
		defaultModelViewerScene,
		modelViewerScene,
		resetModelViewerScene,
	} from "./model-viewer-scene.svelte"
	import type { ModelViewerScene } from "./model-viewer-scene.svelte"

	interface SceneSlider {
		setting: keyof ModelViewerScene
		label: string
		min: number
		max: number
		step: number
		format: (value: number) => string
	}

	let { id, onClose }: { id: string; onClose: () => void } = $props()

	let panel = $state<HTMLDivElement>()

	const percent = new Intl.NumberFormat("en-US", { style: "percent" }),
		formatDegrees = (degrees: number): string => `${degrees}°`,
		formatShare = (share: number): string => percent.format(share),
		angleSlider = {
			min: -180,
			max: 180,
			step: 5,
			format: formatDegrees,
		},
		shareSlider = { min: 0, max: 3, step: 0.05, format: formatShare },
		sliderGroups: { label: string; sliders: SceneSlider[] }[] = [
			{
				label: "Camera",
				sliders: [
					{
						setting: "fieldOfViewDegrees",
						label: "Field of view",
						min: 1,
						max: 120,
						step: 1,
						format: formatDegrees,
					},
				],
			},
			{
				label: "Environment",
				sliders: [
					{ setting: "brightness", label: "Brightness", ...shareSlider },
				],
			},
			{
				label: "Light",
				sliders: [
					{ setting: "lightIntensity", label: "Intensity", ...shareSlider },
					{
						setting: "lightAngleXDegrees",
						label: "Horizontal angle",
						...angleSlider,
					},
					{
						setting: "lightAngleYDegrees",
						label: "Vertical angle",
						...angleSlider,
					},
				],
			},
		],
		isDefault = $derived(
			sliderGroups.every(({ sliders }) =>
				sliders.every(
					({ setting }) =>
						modelViewerScene[setting] === defaultModelViewerScene[setting],
				),
			),
		)

	// setSetting rounds to the step, because range inputs add floating point
	// noise such as 0.9500000001.
	function setSetting(slider: SceneSlider, rawValue: number): void {
		modelViewerScene[slider.setting] = Number(
			(Math.round(rawValue / slider.step) * slider.step).toFixed(2),
		)
	}
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
	class="pointer-events-none absolute inset-x-2 top-11 bottom-2 z-10 flex items-start justify-end">
	<div
		bind:this={panel}
		{id}
		class="pointer-events-auto flex max-h-full w-56 animate-enter flex-col gap-1 overflow-y-auto rounded-lg border border-line-strong bg-overlay p-3 shadow-float"
		role="group"
		aria-label="Camera and lighting"
		data-closes-on-escape>
		{#each sliderGroups as group (group.label)}
			<div
				class="flex flex-col gap-1.5 first:-mt-2"
				role="group"
				aria-label={group.label}>
				<div class="menu-heading mx-0" aria-hidden="true">{group.label}</div>
				{#each group.sliders as slider (slider.setting)}
					{@const settingValue = modelViewerScene[slider.setting]}
					<label class="flex flex-col gap-0.5 text-xs">
						<span class="flex items-center justify-between gap-2">
							<span class="text-fg-muted">{slider.label}</span>
							<span class="font-mono text-2xs text-fg"
								>{slider.format(settingValue)}</span>
						</span>
						<input
							class="range"
							type="range"
							min={slider.min}
							max={slider.max}
							step={slider.step}
							value={settingValue}
							style:--range-fill={`${((settingValue - slider.min) / (slider.max - slider.min)) * 100}%`}
							aria-valuetext={slider.format(settingValue)}
							oninput={(event) =>
								setSetting(
									slider,
									event.currentTarget.valueAsNumber,
								)} />
					</label>
				{/each}
			</div>
		{/each}
		<button
			class="button self-end"
			type="button"
			disabled={isDefault}
			onclick={resetModelViewerScene}>Reset</button>
	</div>
</div>
