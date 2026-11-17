<script lang="ts">
	import cubeIcon from "@phosphor-icons/core/regular/cube.svg?raw"
	import { untrack } from "svelte"
	import FrameUnavailable from "../shared/FrameUnavailable.svelte"
	import { orbitViewKeysDescription } from "../shared/orbit-view-keys"
	import type { AvatarDraft } from "./avatar-draft"
	import {
		attachLiveAvatar,
		liveAvatar,
		retryLiveAvatar,
		showLiveAvatarDraft,
		startLiveAvatarRenderer,
	} from "./live-avatar-renderer.svelte"

	// AvatarLiveView fills its container with the 3D avatar that the
	// application builds from the draft, so that every change shows at once.
	// label names the avatar, such as "@builderman's avatar". isLoading is true
	// while a part of the avatar downloads or builds.
	let {
		accountId,
		draft,
		label,
		isLoading = $bindable(false),
	}: {
		accountId: number
		draft: AvatarDraft
		label: string
		isLoading?: boolean
	} = $props()

	let host = $state<HTMLDivElement>(),
		isStarted = $state(false),
		isUnavailable = $state(false)

	const controlsDescriptionId = $props.id()

	$effect(() => {
		const viewHost = host,
			startingAccountId = untrack(() => accountId)
		if (!viewHost) return
		let detach: (() => void) | undefined,
			isClosed = false
		const attach = async (): Promise<void> => {
			const canDraw = await startLiveAvatarRenderer(startingAccountId)
			if (isClosed) return
			if (!canDraw) {
				isUnavailable = true
				return
			}
			detach = attachLiveAvatar({
				host: viewHost,
				canvasLabel: `3D view of ${label}`,
				descriptionId: controlsDescriptionId,
			})
			isStarted = true
		}
		void attach()
		return () => {
			isClosed = true
			detach?.()
		}
	})

	$effect(() => {
		const shownAccountId = accountId,
			shownDraft = draft
		if (isStarted) untrack(() => showLiveAvatarDraft(shownAccountId, shownDraft))
	})

	$effect(() => {
		isLoading = isStarted && liveAvatar.isLoading
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
	<span id={controlsDescriptionId} class="sr-only">{orbitViewKeysDescription}</span>
	{#if liveAvatar.isFailed}
		<div class="absolute inset-0 grid place-items-center bg-canvas">
			<FrameUnavailable
				icon={cubeIcon}
				message="Could not load the 3D avatar. Try again."
				onRetry={retryLiveAvatar} />
		</div>
	{:else if !liveAvatar.hasShownAvatar}
		<span class="absolute inset-3 skeleton rounded-md" role="status">
			<span class="sr-only">Loading the 3D avatar</span>
		</span>
	{/if}
{/if}
