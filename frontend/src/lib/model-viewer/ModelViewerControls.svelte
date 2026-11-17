<script lang="ts">
	import arrowCounterClockwiseIcon from "@phosphor-icons/core/regular/arrow-counter-clockwise.svg?raw"
	import rulerIcon from "@phosphor-icons/core/regular/ruler.svg?raw"
	import slidersHorizontalIcon from "@phosphor-icons/core/regular/sliders-horizontal.svg?raw"
	import { tick } from "svelte"
	import Icon from "../shared/Icon.svelte"
	import { viewerOverlays } from "./model-viewer-scene.svelte"
	import ModelViewerSceneControls from "./ModelViewerSceneControls.svelte"

	// ModelViewerControls is the toolbar in the corner of every 3D viewer: the
	// coordinates of the camera, a button that returns the camera to where it
	// starts, and the camera and lighting panel.
	let { onResetView }: { onResetView: () => void } = $props()

	let isSceneControlsOpen = $state(false),
		sceneButton = $state<HTMLButtonElement>()

	const sceneControlsId = $props.id()

	async function closeSceneControls(): Promise<void> {
		isSceneControlsOpen = false
		await tick()
		sceneButton?.focus()
	}
</script>

<div class="absolute top-2 right-2 z-10 flex gap-1">
	<button
		class="button-icon bg-overlay"
		type="button"
		aria-label="Show coordinates"
		aria-pressed={viewerOverlays.isCoordinatesShown}
		data-open={viewerOverlays.isCoordinatesShown || undefined}
		data-tooltip={viewerOverlays.isCoordinatesShown
			? "Hide coordinates"
			: "Show coordinates"}
		onclick={() =>
			(viewerOverlays.isCoordinatesShown = !viewerOverlays.isCoordinatesShown)}>
		<Icon svg={rulerIcon} size={16} aria-hidden="true" />
	</button>
	<button
		class="button-icon bg-overlay"
		type="button"
		aria-label="Reset view"
		data-tooltip="Reset view"
		onclick={onResetView}>
		<Icon svg={arrowCounterClockwiseIcon} size={16} aria-hidden="true" />
	</button>
	<button
		bind:this={sceneButton}
		class="button-icon bg-overlay"
		type="button"
		aria-label="Camera and lighting"
		aria-expanded={isSceneControlsOpen}
		aria-controls={sceneControlsId}
		data-open={isSceneControlsOpen || undefined}
		data-tooltip="Camera and lighting"
		onclick={() => (isSceneControlsOpen = !isSceneControlsOpen)}>
		<Icon svg={slidersHorizontalIcon} size={16} aria-hidden="true" />
	</button>
</div>
{#if isSceneControlsOpen}
	<ModelViewerSceneControls
		id={sceneControlsId}
		onClose={() => void closeSceneControls()} />
{/if}
