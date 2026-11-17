<script lang="ts">
	import cubeIcon from "@phosphor-icons/core/regular/cube.svg?raw"
	import FrameUnavailable from "../shared/FrameUnavailable.svelte"

	// ModelViewerUnavailable fills a 3D viewer that cannot show a model: on a
	// computer without graphics acceleration, or when the viewer did not start.
	let { isGraphicsAccelerationMissing }: { isGraphicsAccelerationMissing: boolean } =
		$props()
</script>

<FrameUnavailable
	icon={cubeIcon}
	message={isGraphicsAccelerationMissing
		? "3D needs graphics acceleration, which isn't available on this computer."
		: "The 3D viewer couldn't start."} />
