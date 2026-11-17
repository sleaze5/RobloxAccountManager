<script lang="ts">
	import cubeIcon from "@phosphor-icons/core/regular/cube.svg?raw"
	import FrameUnavailable from "../shared/FrameUnavailable.svelte"
	import type { AnimationPlayback, ModelSubject } from "./model-subject"
	import RenderedModelView from "./RenderedModelView.svelte"

	// ModelFrame is the one 3D viewer of the application. It shows a subject
	// that the application builds itself with the live renderer, or a model
	// that Roblox renders, with the same toolbar, keys, and states. label names
	// the subject, such as "@builderman's avatar". isLoading is true while a
	// part of a live subject downloads or builds. onLiveFailure lets the caller
	// show a model that Roblox renders instead, when the live renderer cannot
	// build the subject. playback plays the animations of a live avatar, for
	// the AnimationControls that the caller places next to the frame.
	let {
		accountId,
		subject,
		label,
		isLoading = $bindable(false),
		playback = $bindable(),
		onLiveFailure,
	}: {
		accountId: number
		subject: ModelSubject | null
		label: string
		isLoading?: boolean
		playback?: AnimationPlayback
		onLiveFailure?: () => void
	} = $props()

	// The live renderer is large, so it loads the first time a live subject
	// shows. The import depends only on whether one shows, so that a changed
	// subject keeps the viewer.
	const isLiveSubjectShown = $derived(!!subject && subject.kind !== "rendered"),
		liveViewImport = $derived(
			isLiveSubjectShown ? import("./LiveModelView.svelte") : undefined,
		)
</script>

{#if subject?.kind === "rendered"}
	<RenderedModelView
		model={subject.model}
		error={subject.error}
		{label}
		onRetry={subject.onRetry} />
{:else if subject && liveViewImport}
	{#await liveViewImport}
		<span class="absolute inset-3 skeleton rounded-md" role="status">
			<span class="sr-only">Loading the 3D model</span>
		</span>
	{:then { default: LiveModelView }}
		<LiveModelView
			{accountId}
			{subject}
			{label}
			bind:isLoading
			bind:playback
			onFailure={onLiveFailure} />
	{:catch}
		<FrameUnavailable icon={cubeIcon} message="The 3D viewer couldn't start." />
	{/await}
{:else}
	<span class="absolute inset-3 skeleton rounded-md" role="status">
		<span class="sr-only">Loading the 3D model</span>
	</span>
{/if}
