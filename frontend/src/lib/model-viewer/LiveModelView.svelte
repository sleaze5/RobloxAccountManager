<script lang="ts">
	import cubeIcon from "@phosphor-icons/core/regular/cube.svg?raw"
	import { untrack } from "svelte"
	import FrameUnavailable from "../shared/FrameUnavailable.svelte"
	import ModelViewerControls from "./ModelViewerControls.svelte"
	import { orbitViewKeysDescription } from "./orbit-view-keys"
	import { modelViewerScene, viewerOverlays } from "./model-viewer-scene.svelte"
	import ViewerAxes from "./ViewerAxes.svelte"
	import ViewerCoordinates from "./ViewerCoordinates.svelte"
	import {
		applyLiveModelSceneSettings,
		attachLiveModelView,
		startLiveModelScene,
	} from "./live-model-scene.svelte"
	import type { LiveModelViewport } from "./live-model-scene.svelte"
	import type { AnimationPlayback, LiveModelSubject } from "./model-subject"

	// LiveModelView fills its container with the subject, which the
	// application builds itself, so that every change shows at once. label
	// names the subject, such as "@builderman's avatar". isLoading is true
	// while a part of the subject downloads or builds. playback plays the
	// animations of an avatar, and is undefined for an accessory. When
	// onFailure is set, a subject that cannot be built calls it instead of
	// offering to try again.
	let {
		accountId,
		subject,
		label,
		isLoading = $bindable(false),
		playback = $bindable(),
		onFailure,
	}: {
		accountId: number
		subject: LiveModelSubject
		label: string
		isLoading?: boolean
		playback?: AnimationPlayback
		onFailure?: () => void
	} = $props()

	let host = $state<HTMLDivElement>(),
		viewport = $state<LiveModelViewport>(),
		isUnavailable = $state(false)

	const controlsDescriptionId = $props.id()

	$effect(() => {
		const viewHost = host,
			startingAccountId = untrack(() => accountId)
		if (!viewHost) return
		let attachedViewport: LiveModelViewport | undefined,
			isClosed = false
		const attach = async (): Promise<void> => {
			const canDraw = await startLiveModelScene(startingAccountId)
			if (isClosed) return
			if (!canDraw) {
				isUnavailable = true
				return
			}
			attachedViewport = attachLiveModelView(
				viewHost,
				`3D view of ${untrack(() => label)}`,
				controlsDescriptionId,
			)
			viewport = attachedViewport
		}
		void attach()
		return () => {
			isClosed = true
			attachedViewport?.detach()
		}
	})

	$effect(() => {
		const shownAccountId = accountId,
			shownSubject = subject,
			shownViewport = viewport
		if (shownViewport) {
			untrack(() => shownViewport.show(shownAccountId, shownSubject))
		}
	})

	// The scene follows the camera and lighting controls without rebuilding.
	$effect(() => {
		const sceneSettings = { ...modelViewerScene }
		if (viewport) untrack(() => applyLiveModelSceneSettings(sceneSettings))
	})

	$effect(() => {
		isLoading = !!viewport?.isLoading
	})

	$effect(() => {
		playback =
			viewport && subject.kind === "avatar" && viewport.hasShownSubject
				? viewport
				: undefined
		return () => {
			playback = undefined
		}
	})

	$effect(() => {
		if (viewport?.isFailed && onFailure) untrack(onFailure)
	})
</script>

{#if isUnavailable}
	<FrameUnavailable
		icon={cubeIcon}
		message="3D needs graphics acceleration, which isn't available on this computer." />
{:else}
	<div
		bind:this={host}
		class="relative size-full animate-pop overflow-hidden rounded-lg">
	</div>
	<span id={controlsDescriptionId} class="sr-only"
		>{orbitViewKeysDescription}{subject.kind === "avatar"
			? " Space plays or pauses the animation."
			: ""}</span>
	{#if viewport?.coveredFrameUrl}
		<img
			class="absolute inset-0 size-full object-contain"
			src={viewport.coveredFrameUrl}
			alt=""
			draggable="false" />
	{/if}
	{#if viewport?.isFailed && !onFailure}
		<div class="absolute inset-0 grid place-items-center bg-canvas">
			<FrameUnavailable
				icon={cubeIcon}
				message="Could not load the 3D model. Try again."
				onRetry={() => viewport?.retry()} />
		</div>
	{:else if !viewport?.hasShownSubject}
		<span class="absolute inset-3 skeleton rounded-md" role="status">
			<span class="sr-only">Loading the 3D model</span>
		</span>
	{:else if viewport && !viewport.coveredFrameUrl}
		{#if viewport.cameraPose}
			<ViewerAxes pose={viewport.cameraPose} />
			{#if viewerOverlays.isCoordinatesShown}
				<ViewerCoordinates pose={viewport.cameraPose} />
			{/if}
		{/if}
		<ModelViewerControls onResetView={() => viewport?.resetView()} />
	{/if}
{/if}
