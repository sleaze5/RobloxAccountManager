<script lang="ts">
	import cubeIcon from "@phosphor-icons/core/regular/cube.svg?raw"
	import { untrack } from "svelte"
	import FrameUnavailable from "../shared/FrameUnavailable.svelte"
	import ModelLoadingPlaceholder from "./ModelLoadingPlaceholder.svelte"
	import ModelSnapshotButton from "./ModelSnapshotButton.svelte"
	import ModelViewerControls from "./ModelViewerControls.svelte"
	import ModelViewerUnavailable from "./ModelViewerUnavailable.svelte"
	import { orbitViewKeysDescription } from "./orbit-camera"
	import { modelViewerScene } from "./model-viewer-scene.svelte"
	import ViewerReadout from "./ViewerReadout.svelte"
	import {
		applyLiveModelSceneSettings,
		attachLiveModelView,
		startLiveModelScene,
	} from "./live-model-scene.svelte"
	import type { LiveModelViewport } from "./live-model-scene.svelte"
	import type {
		AdjustmentGuide,
		AnimationPlayback,
		LiveModelSubject,
		ThumbnailFraming,
	} from "./model-subject"

	// LiveModelView fills its container with the subject, which the
	// application builds itself, so that every change shows at once. label
	// names the subject, such as "@builderman's avatar". isLoading is true
	// while a part of the subject downloads or builds. playback plays the
	// animations of an avatar, and is undefined for an accessory.
	// adjustmentGuide shows the guides of the accessory that the user adjusts.
	// thumbnailFraming shows the avatar as Roblox frames one of its profile
	// images, with the camera and the pose locked. When
	// onFailure is set, a subject that cannot be built calls it instead of
	// offering to try again.
	let {
		accountId,
		subject,
		label,
		isLoading = $bindable(false),
		playback = $bindable(),
		adjustmentGuide,
		thumbnailFraming,
		onFailure,
	}: {
		accountId: number
		subject: LiveModelSubject
		label: string
		isLoading?: boolean
		playback?: AnimationPlayback
		adjustmentGuide?: AdjustmentGuide
		thumbnailFraming?: ThumbnailFraming
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

	$effect(() => {
		const shownGuide = adjustmentGuide,
			shownViewport = viewport
		if (shownViewport) {
			untrack(() => shownViewport.showAdjustmentGuide(shownGuide))
		}
	})

	$effect(() => {
		const shownFraming = thumbnailFraming,
			shownViewport = viewport
		if (shownViewport) {
			untrack(() => shownViewport.showThumbnailFraming(shownFraming))
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
			viewport &&
			subject.kind === "avatar" &&
			viewport.hasShownSubject &&
			!thumbnailFraming
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
	<ModelViewerUnavailable isGraphicsAccelerationMissing />
{:else}
	<div
		bind:this={host}
		class={[
			"relative size-full animate-pop overflow-hidden",
			// A headshot shows in a circle wherever Roblox shows a profile picture.
			thumbnailFraming?.image === "headshot"
				? "rounded-full border border-line-strong"
				: "rounded-lg",
		]}>
	</div>
	<span id={controlsDescriptionId} class="sr-only"
		>{thumbnailFraming
			? "The camera and the pose follow the thumbnail settings."
			: `${orbitViewKeysDescription}${subject.kind === "avatar" ? " Space plays or pauses the animation." : ""}`}</span>
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
				message="The 3D model could not load."
				onRetry={() => viewport?.retry()} />
		</div>
	{:else if !viewport?.hasShownSubject}
		<ModelLoadingPlaceholder />
	{:else if viewport && !viewport.coveredFrameUrl}
		{#if viewport.cameraPose && !thumbnailFraming}
			<ViewerReadout pose={viewport.cameraPose} />
		{/if}
		{#if !thumbnailFraming}
			<ModelViewerControls onResetView={() => viewport?.resetView()} />
		{/if}
		<ModelSnapshotButton
			{label}
			captureFrame={() => viewport?.captureShownFrame() ?? ""} />
	{/if}
{/if}
