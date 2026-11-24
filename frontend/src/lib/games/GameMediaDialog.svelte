<script lang="ts">
	import caretLeftIcon from "@phosphor-icons/core/regular/caret-left.svg?raw"
	import caretRightIcon from "@phosphor-icons/core/regular/caret-right.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import imageIcon from "@phosphor-icons/core/regular/image.svg?raw"
	import pauseIcon from "@phosphor-icons/core/regular/pause.svg?raw"
	import playIcon from "@phosphor-icons/core/regular/play.svg?raw"
	import speakerHighIcon from "@phosphor-icons/core/regular/speaker-high.svg?raw"
	import speakerSlashIcon from "@phosphor-icons/core/regular/speaker-slash.svg?raw"
	import { accountBackend, GameMediaKind } from "../backend/bridge"
	import type { GameMedia } from "../backend/bridge"
	import { formatClock } from "../shared/duration"
	import ExpandedImageDialog from "../shared/ExpandedImageDialog.svelte"
	import Icon from "../shared/Icon.svelte"

	// GameMediaDialog shows the images and Roblox videos of a game one at a
	// time, as large as the window allows. A video plays with the app's own
	// controls below the frame.
	let {
		gameName,
		accountId,
		media,
		initialIndex,
		onClose,
	}: {
		gameName: string
		accountId: number
		media: GameMedia[]
		initialIndex: number
		onClose: () => void
	} = $props()

	let index = $state(0),
		video = $state<HTMLVideoElement>(),
		videoUrl = $state(""),
		hasVideoFailed = $state(false),
		videoLoadCount = $state(0),
		isPaused = $state(true),
		isMuted = $state(false),
		isWaitingForVideo = $state(false),
		currentTimeSeconds = $state(0),
		durationSeconds = $state(0)

	$effect.pre(() => {
		index = Math.min(Math.max(initialIndex, 0), media.length - 1)
	})

	const current = $derived(media[index]),
		isVideo = $derived(current?.kind === GameMediaKind.MediaKindVideo),
		noun = $derived(isVideo ? "Video" : "Image"),
		canSeek = $derived(!!videoUrl && durationSeconds > 0),
		playLabel = $derived(isPaused ? "Play video" : "Pause video"),
		muteLabel = $derived(isMuted ? "Turn on sound" : "Turn off sound"),
		timeLabel = $derived(
			`${formatClock(currentTimeSeconds)} / ${formatClock(durationSeconds)}`,
		)

	// The video address is signed and expires, so each opened video reads its
	// own. Opening a video is the request to play it, so it starts at once.
	$effect(() => {
		const requestedVideoAssetId = isVideo ? current.videoAssetId : 0
		void videoLoadCount
		videoUrl = ""
		hasVideoFailed = false
		currentTimeSeconds = durationSeconds = 0
		isPaused = true
		if (!requestedVideoAssetId) return
		if (accountId <= 0) {
			hasVideoFailed = true
			return
		}
		let isCurrent = true
		const request = accountBackend.GetGameVideoURL(accountId, requestedVideoAssetId)
		void (async () => {
			try {
				const resolvedUrl = await request
				if (isCurrent) videoUrl = resolvedUrl
			} catch {
				// The backend logs the failure, and the frame says that the video
				// is unavailable.
				if (isCurrent) hasVideoFailed = true
			}
		})()
		return () => {
			isCurrent = false
			void request.cancel()
		}
	})

	function step(direction: -1 | 1): void {
		index = (index + direction + media.length) % media.length
	}

	function togglePlay(): void {
		if (video?.paused) startPlayback()
		else video?.pause()
	}

	function startPlayback(): void {
		video?.play().catch(() => {
			// The WebView can refuse to start a video on its own, and the
			// Play button stays available.
		})
	}
</script>

<!-- The arrow keys step through the media, except in the seek bar, which
uses them to seek. -->
<svelte:window
	onkeydown={(event) => {
		if (media.length < 2 || event.target instanceof HTMLInputElement) return
		if (event.key === "ArrowLeft") step(-1)
		else if (event.key === "ArrowRight") step(1)
	}} />

<ExpandedImageDialog label={`${gameName} media`} aspectRatio={16 / 9} {onClose}>
	{#snippet frame()}
		{#key index}
			{#if isVideo}
				{#if videoUrl && !hasVideoFailed}
					<!-- svelte-ignore a11y_media_has_caption: Roblox serves game videos without captions. -->
					<video
						bind:this={video}
						class="size-full animate-enter bg-deep object-contain"
						src={videoUrl}
						poster={current.imageUrl || undefined}
						preload="auto"
						aria-label={`${gameName} video`}
						bind:paused={isPaused}
						bind:muted={isMuted}
						bind:currentTime={currentTimeSeconds}
						bind:duration={durationSeconds}
						onloadeddata={startPlayback}
						onwaiting={() => (isWaitingForVideo = true)}
						onplaying={() => (isWaitingForVideo = false)}
						onerror={() => (hasVideoFailed = true)}
						onclick={togglePlay}></video>
				{:else if hasVideoFailed}
					<div class="empty-state">
						<p>Video unavailable</p>
						<button
							class="button"
							type="button"
							onclick={() => videoLoadCount++}>
							Try again
						</button>
					</div>
				{:else}
					{#if current.imageUrl}
						<img
							class="size-full object-contain opacity-45"
							src={current.imageUrl}
							alt=""
							draggable="false"
							referrerpolicy="no-referrer" />
					{/if}
					<span class="absolute inset-0 m-auto size-fit" role="status">
						<Icon
							svg={circleNotchIcon}
							class="spinner text-fg-bright"
							size={28}
							aria-label="Loading video…" />
					</span>
				{/if}
			{:else if current.imageUrl}
				<img
					class="size-full animate-enter object-contain"
					src={current.imageUrl}
					alt={current.altText}
					draggable="false"
					referrerpolicy="no-referrer" />
			{:else}
				<div class="empty-state">
					<Icon svg={imageIcon} size={28} aria-hidden="true" />
					<p>Image unavailable</p>
				</div>
			{/if}
		{/key}
	{/snippet}
	{#snippet controls()}
		<button
			class="button-icon"
			type="button"
			aria-label="Previous"
			data-tooltip="Previous"
			disabled={media.length < 2}
			onclick={() => step(-1)}>
			<Icon svg={caretLeftIcon} data-nudge="left" size={16} aria-hidden="true" />
		</button>
		<span
			class="shrink-0 font-mono text-2xs text-fg-muted tabular-nums"
			aria-live="polite">{noun} {index + 1} of {media.length}</span>
		<button
			class="button-icon"
			type="button"
			aria-label="Next"
			data-tooltip="Next"
			disabled={media.length < 2}
			onclick={() => step(1)}>
			<Icon
				svg={caretRightIcon}
				data-nudge="right"
				size={16}
				aria-hidden="true" />
		</button>
		{#if isVideo}
			<span class="mx-1 h-4 w-px shrink-0 bg-line-strong" aria-hidden="true"
			></span>
			<button
				class="button-icon"
				type="button"
				aria-label={playLabel}
				aria-busy={isWaitingForVideo && !isPaused}
				data-tooltip={playLabel}
				disabled={!videoUrl || hasVideoFailed}
				onclick={togglePlay}>
				{#if isWaitingForVideo && !isPaused}
					<Icon
						svg={circleNotchIcon}
						class="spinner"
						size={16}
						aria-hidden="true" />
				{:else}
					{#key isPaused}
						<Icon
							svg={isPaused ? playIcon : pauseIcon}
							class="animate-pop"
							size={16}
							aria-hidden="true" />
					{/key}
				{/if}
			</button>
			<label class="flex min-w-0 flex-1 items-center gap-2">
				<span class="sr-only">Video time</span>
				<input
					class="range min-w-0 flex-1"
					type="range"
					min="0"
					max={durationSeconds || 1}
					step="0.1"
					value={currentTimeSeconds}
					disabled={!canSeek}
					style:--range-fill={`${canSeek ? (currentTimeSeconds / durationSeconds) * 100 : 0}%`}
					aria-valuetext={timeLabel}
					oninput={(event) => {
						if (video) video.currentTime = event.currentTarget.valueAsNumber
					}} />
				<span class="shrink-0 font-mono text-2xs text-fg-muted tabular-nums"
					>{timeLabel}</span>
			</label>
			<button
				class="button-icon"
				type="button"
				aria-label={muteLabel}
				aria-pressed={isMuted}
				data-tooltip={muteLabel}
				disabled={!videoUrl || hasVideoFailed}
				onclick={() => (isMuted = !isMuted)}>
				<Icon
					svg={isMuted ? speakerSlashIcon : speakerHighIcon}
					size={16}
					aria-hidden="true" />
			</button>
		{/if}
	{/snippet}
</ExpandedImageDialog>
