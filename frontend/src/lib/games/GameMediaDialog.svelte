<script lang="ts">
	import caretLeftIcon from "@phosphor-icons/core/regular/caret-left.svg?raw"
	import caretRightIcon from "@phosphor-icons/core/regular/caret-right.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import imageIcon from "@phosphor-icons/core/regular/image.svg?raw"
	import pauseIcon from "@phosphor-icons/core/regular/pause.svg?raw"
	import playIcon from "@phosphor-icons/core/regular/play.svg?raw"
	import speakerHighIcon from "@phosphor-icons/core/regular/speaker-high.svg?raw"
	import speakerLowIcon from "@phosphor-icons/core/regular/speaker-low.svg?raw"
	import speakerSlashIcon from "@phosphor-icons/core/regular/speaker-slash.svg?raw"
	import { GameMediaKind } from "../backend/bridge"
	import type { GameMedia } from "../backend/bridge"
	import { formatClock } from "../shared/duration"
	import ExpandedImageDialog from "../shared/ExpandedImageDialog.svelte"
	import Icon from "../shared/Icon.svelte"

	// GameMediaDialog shows the images and Roblox videos of a game one at a
	// time, as large as the window allows. A video plays with the app's own
	// controls below the frame.
	let {
		gameName,
		media,
		initialIndex,
		onClose,
	}: {
		gameName: string
		media: GameMedia[]
		initialIndex: number
		onClose: () => void
	} = $props()

	let index = $state(0),
		video = $state<HTMLVideoElement>(),
		hasVideoFailed = $state(false),
		isVideoLoaded = $state(false),
		videoLoadCount = $state(0),
		isPaused = $state(true),
		isMuted = $state(false),
		volume = $state(1),
		isWaitingForVideo = $state(false),
		currentTimeSeconds = $state(0),
		durationSeconds = $state(0)

	$effect.pre(() => {
		index = Math.min(Math.max(initialIndex, 0), media.length - 1)
	})

	const current = $derived(media[index]),
		isVideo = $derived(current?.kind === GameMediaKind.MediaKindVideo),
		noun = $derived(isVideo ? "Video" : "Image"),
		// The app serves each video from its own address, so that the player
		// can seek in it. Roblox keeps the file gzipped and cannot serve parts
		// of it.
		videoUrl = $derived(isVideo ? `/game-videos/${current.videoAssetId}` : ""),
		areVideoControlsDisabled = $derived(!isVideoLoaded || hasVideoFailed),
		canSeek = $derived(isVideoLoaded && durationSeconds > 0),
		playLabel = $derived(isPaused ? "Play video" : "Pause video"),
		// Sound is off when the video is muted or its volume is at zero, and the
		// speaker shows the level the user hears.
		isSilent = $derived(isMuted || volume === 0),
		heardVolume = $derived(isSilent ? 0 : volume),
		speakerIcon = $derived(
			isSilent
				? speakerSlashIcon
				: volume < 0.5
					? speakerLowIcon
					: speakerHighIcon,
		),
		muteLabel = $derived(isSilent ? "Turn on sound" : "Turn off sound"),
		timeLabel = $derived(
			`${formatClock(currentTimeSeconds)} / ${formatClock(durationSeconds)}`,
		)

	// Opening a video is the request to play it, so it starts as soon as it
	// loads. Another video, or Try again, starts over.
	$effect(() => {
		void index
		void videoLoadCount
		hasVideoFailed = isVideoLoaded = false
		currentTimeSeconds = durationSeconds = 0
		isPaused = true
	})

	function step(direction: -1 | 1): void {
		index = (index + direction + media.length) % media.length
	}

	function togglePlay(): void {
		if (video?.paused) startPlayback()
		else video?.pause()
	}

	// Turning on sound at zero volume would stay silent, so it also raises
	// the volume to half.
	function toggleSound(): void {
		if (isSilent) {
			isMuted = false
			if (volume === 0) volume = 0.5
		} else isMuted = true
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
				{#if hasVideoFailed}
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
					{#key videoLoadCount}
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
							bind:volume
							bind:currentTime={currentTimeSeconds}
							bind:duration={durationSeconds}
							onloadeddata={() => {
								isVideoLoaded = true
								startPlayback()
							}}
							onwaiting={() => (isWaitingForVideo = true)}
							onplaying={() => (isWaitingForVideo = false)}
							onerror={() => (hasVideoFailed = true)}
							onclick={togglePlay}></video>
					{/key}
					{#if !isVideoLoaded}
						{#if current.imageUrl}
							<img
								class="absolute inset-0 size-full bg-deep object-contain opacity-45"
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
				disabled={areVideoControlsDisabled}
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
				aria-pressed={isSilent}
				data-tooltip={muteLabel}
				disabled={areVideoControlsDisabled}
				onclick={toggleSound}>
				{#key speakerIcon}
					<Icon
						svg={speakerIcon}
						class="animate-pop"
						size={16}
						aria-hidden="true" />
				{/key}
			</button>
			<label class="flex w-20 shrink-0 items-center">
				<span class="sr-only">Volume</span>
				<input
					class="range"
					type="range"
					min="0"
					max="1"
					step="0.05"
					value={heardVolume}
					disabled={areVideoControlsDisabled}
					style:--range-fill={`${heardVolume * 100}%`}
					aria-valuetext={`${Math.round(heardVolume * 100)}%`}
					oninput={(event) => {
						volume = event.currentTarget.valueAsNumber
						isMuted = volume === 0
					}} />
			</label>
		{/if}
	{/snippet}
</ExpandedImageDialog>
