<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import cubeIcon from "@phosphor-icons/core/regular/cube.svg?raw"
	import type { RenderedModel } from "../backend/bridge"
	import FrameUnavailable from "../shared/FrameUnavailable.svelte"
	import type { CameraPose } from "./model-subject"
	import ModelLoadingPlaceholder from "./ModelLoadingPlaceholder.svelte"
	import ModelSnapshotButton from "./ModelSnapshotButton.svelte"
	import ModelViewerControls from "./ModelViewerControls.svelte"
	import ModelViewerUnavailable from "./ModelViewerUnavailable.svelte"
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
		viewer = $state<{ resetView: () => void; captureFrame: () => string }>(),
		cameraPose = $state.raw<CameraPose>()

	// three.js is large, so the viewer loads only when a model is shown.
	const viewerImport = $derived(
		model && !isWebGLUnavailable
			? import("./RenderedModelViewer.svelte")
			: undefined,
	)

	notifyErrors("3D model not loaded", () => error)
</script>

{#if isWebGLUnavailable}
	<ModelViewerUnavailable isGraphicsAccelerationMissing />
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
		<ModelSnapshotButton
			{label}
			captureFrame={() => viewer?.captureFrame() ?? ""} />
	{:catch}
		<ModelViewerUnavailable isGraphicsAccelerationMissing={false} />
	{/await}
{:else if error}
	<FrameUnavailable icon={cubeIcon} message="3D model unavailable" {onRetry} />
{:else}
	<ModelLoadingPlaceholder />
{/if}
