<script lang="ts">
	import type { CameraPose } from "./model-subject"
	import { viewerOverlays } from "./model-viewer-scene.svelte"
	import ViewerAxes from "./ViewerAxes.svelte"
	import ViewerCoordinates from "./ViewerCoordinates.svelte"

	// ViewerReadout is the readout docked in the lower left corner of a 3D
	// viewer: the axes, and the coordinates of the camera while they are shown.
	let { pose }: { pose: CameraPose } = $props()
</script>

<div class="frame-readout-bottom-left">
	<ViewerAxes {pose} />
	{#if viewerOverlays.isCoordinatesShown}
		<ViewerCoordinates {pose} />
	{/if}
</div>
