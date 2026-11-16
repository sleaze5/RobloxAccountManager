<script lang="ts">
	import cubeIcon from "@phosphor-icons/core/regular/cube.svg?raw"
	import slidersHorizontalIcon from "@phosphor-icons/core/regular/sliders-horizontal.svg?raw"
	import { tick } from "svelte"
	import type { RenderedModel } from "../backend/bridge"
	import Icon from "./Icon.svelte"
	import RenderedModelSceneControls from "./RenderedModelSceneControls.svelte"

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
		isSceneControlsOpen = $state(false),
		sceneButton = $state<HTMLButtonElement>()

	const sceneControlsId = $props.id()

	// three.js is large, so the viewer loads only when a model is shown.
	const viewerImport = $derived(
		model && !isWebGLUnavailable
			? import("./RenderedModelViewer.svelte")
			: undefined,
	)

	async function closeSceneControls(): Promise<void> {
		isSceneControlsOpen = false
		await tick()
		sceneButton?.focus()
	}
</script>

{#snippet unavailable(message: string, canRetry: boolean)}
	<div
		class="relative flex animate-enter flex-col items-center gap-2 p-3 text-center text-xs text-fg-muted"
		role="alert">
		<Icon svg={cubeIcon} size={28} class="animate-bloom" aria-hidden="true" />
		<p>{message}</p>
		{#if canRetry}
			<button class="button" type="button" onclick={onRetry}>Try again</button>
		{/if}
	</div>
{/snippet}

{#if isWebGLUnavailable}
	{@render unavailable(
		"3D needs graphics acceleration, which isn't available on this computer.",
		false,
	)}
{:else if model && viewerImport}
	{#await viewerImport then { default: RenderedModelViewer }}
		{#key model}
			<RenderedModelViewer
				{model}
				{label}
				onerror={() => (isWebGLUnavailable = true)} />
		{/key}
		<button
			bind:this={sceneButton}
			class="absolute top-2 right-2 z-10 button-icon bg-overlay"
			type="button"
			aria-label="Camera and lighting"
			aria-expanded={isSceneControlsOpen}
			aria-controls={sceneControlsId}
			data-open={isSceneControlsOpen || undefined}
			data-tooltip="Camera and lighting"
			onclick={() => (isSceneControlsOpen = !isSceneControlsOpen)}>
			<Icon svg={slidersHorizontalIcon} size={16} aria-hidden="true" />
		</button>
		{#if isSceneControlsOpen}
			<RenderedModelSceneControls
				id={sceneControlsId}
				onClose={() => void closeSceneControls()} />
		{/if}
	{:catch}
		{@render unavailable("The 3D viewer couldn't start.", false)}
	{/await}
{:else if error}
	{@render unavailable(error, true)}
{:else}
	<span class="absolute inset-3 skeleton rounded-md" role="status">
		<span class="sr-only">Loading the 3D model</span>
	</span>
{/if}
