<script lang="ts">
	import cubeIcon from "@phosphor-icons/core/regular/cube.svg?raw"
	import type { RenderedModel } from "../backend/bridge"
	import FrameUnavailable from "../shared/FrameUnavailable.svelte"
	import type { CameraPose } from "./model-subject"
	import ModelViewerControls from "./ModelViewerControls.svelte"
	import ViewerReadout from "./ViewerReadout.svelte"

	// RenderedModelView fills its container with a 3D model, or with its
	// loading or error state. label names what the model shows, such as
	// "@builderman's avatar".
	let {
		model,
		error,
		label,
		onRetry,
	}: {
		model: RenderedModel | null
		error: string
		label: string
		onRetry: () => void
	} = $props()

	let isWebGLUnavailable = $state(false),
		viewer = $state<{ resetView: () => void }>(),
		cameraPose = $state.raw<CameraPose>()

	// three.js is large, so the viewer loads only when a model is shown.
	const viewerImport = $derived(
		model && !isWebGLUnavailable
			? import("./RenderedModelViewer.svelte")
			: undefined,
	)
</script>

{#if isWebGLUnavailable}
	<FrameUnavailable
		icon={cubeIcon}
		message="3D needs graphics acceleration, which isn't available on this computer." />
{:else if model && viewerImport}
	{#await viewerImport then { default: RenderedModelViewer }}
		{#key model}
			<RenderedModelViewer
				bind:this={viewer}
				{model}
				{label}
				onerror={() => (isWebGLUnavailable = true)}
				onposechange={(pose) => (cameraPose = pose)} />
		{/key}
		{#if cameraPose}
			<ViewerReadout pose={cameraPose} />
		{/if}
		<ModelViewerControls onResetView={() => viewer?.resetView()} />
	{:catch}
		<FrameUnavailable icon={cubeIcon} message="The 3D viewer couldn't start." />
	{/await}
{:else if error}
	<FrameUnavailable icon={cubeIcon} message={error} {onRetry} />
{:else}
	<span class="absolute inset-3 skeleton rounded-md" role="status">
		<span class="sr-only">Loading the 3D model</span>
	</span>
{/if}
