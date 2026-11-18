<script lang="ts">
	import arrowCounterClockwiseIcon from "@phosphor-icons/core/regular/arrow-counter-clockwise.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import pauseIcon from "@phosphor-icons/core/regular/pause.svg?raw"
	import playIcon from "@phosphor-icons/core/regular/play.svg?raw"
	import speedometerIcon from "@phosphor-icons/core/regular/speedometer.svg?raw"
	import { tick } from "svelte"
	import Icon from "../shared/Icon.svelte"
	import AnimationPicker from "./AnimationPicker.svelte"
	import type { AnimationLibrary, AnimationPlayback } from "./model-subject"

	// AnimationControls plays the animations and emotes of the avatar in a 3D
	// viewer: the animation picker, and a seek row with a play button, the
	// seek bar, and a speed button that opens a speed slider above it. A wide
	// bar, such as under an expanded viewer, puts them in one row, and so does
	// a bar without the picker, which plays one animation, such as the preview
	// of an animation item. playback is undefined until the avatar shows, which
	// disables the controls, so that the bar keeps its place while the avatar
	// loads. library adds what the user owns and Play by ID to the picker.
	let {
		playback,
		library,
		isWide = false,
		hasAnimationPicker = true,
	}: {
		playback: AnimationPlayback | undefined
		library?: AnimationLibrary
		isWide?: boolean
		hasAnimationPicker?: boolean
	} = $props()

	const isOneRow = $derived(isWide || !hasAnimationPicker)

	const minSpeed = 0.25,
		maxSpeed = 3,
		speedStep = 0.05,
		defaultSpeed = 1,
		seconds = new Intl.NumberFormat("en-US", {
			minimumFractionDigits: 2,
			maximumFractionDigits: 2,
		}),
		// Two decimals keep the width of the label while the speed changes.
		speedFactor = new Intl.NumberFormat("en-US", {
			minimumFractionDigits: 2,
			maximumFractionDigits: 2,
		}),
		speedPanelId = $props.id(),
		speedLabelId = `${speedPanelId}-label`

	let isSpeedPanelOpen = $state(false),
		speedArea = $state<HTMLDivElement>(),
		speedButton = $state<HTMLButtonElement>(),
		speedInput = $state<HTMLInputElement>()

	// Whether the animation played when a drag of the seek bar began, so that
	// it plays on once the drag ends.
	let wasPlayingBeforeSeek = false

	const isPlaying = $derived(!!playback?.isPlaying),
		isAnimationLoading = $derived(!!playback?.isAnimationLoading),
		isAnimationFailed = $derived(!!playback?.isAnimationFailed),
		timeSeconds = $derived(playback?.timeSeconds ?? 0),
		lengthSeconds = $derived(playback?.lengthSeconds ?? 0),
		canSeek = $derived(!!playback && !isAnimationLoading && lengthSeconds > 0),
		timeLabel = $derived(
			`${seconds.format(timeSeconds)} / ${seconds.format(lengthSeconds)} s`,
		),
		fillPercent = $derived(canSeek ? (timeSeconds / lengthSeconds) * 100 : 0),
		playLabel = $derived(isPlaying ? "Pause animation" : "Play animation"),
		speed = $derived(playback?.speed ?? defaultSpeed),
		speedLabel = $derived(`${speedFactor.format(speed)}×`)

	function togglePlay(): void {
		if (isPlaying) playback?.pause()
		else playback?.play()
	}

	function pauseForSeek(): void {
		wasPlayingBeforeSeek = isPlaying
		playback?.pause()
	}

	function resumeAfterSeek(): void {
		if (wasPlayingBeforeSeek) playback?.play()
		wasPlayingBeforeSeek = false
	}

	// setSpeed rounds to the step, because range inputs add floating point
	// noise such as 0.9500000001.
	function setSpeed(rawSpeed: number): void {
		playback?.setSpeed(
			Number((Math.round(rawSpeed / speedStep) * speedStep).toFixed(2)),
		)
	}

	async function toggleSpeedPanel(): Promise<void> {
		isSpeedPanelOpen = !isSpeedPanelOpen
		await tick()
		speedInput?.focus()
	}

	async function closeSpeedPanel(): Promise<void> {
		isSpeedPanelOpen = false
		await tick()
		speedButton?.focus()
	}

	// The panel closes when the controls lose their avatar, such as while the
	// next avatar loads.
	$effect(() => {
		if (!playback) isSpeedPanelOpen = false
	})
</script>

<!-- Escape closes the speed panel before it reaches a dialog around the bar,
and a press outside it closes it. -->
<svelte:window
	onkeydowncapture={(event) => {
		if (
			!isSpeedPanelOpen ||
			event.key !== "Escape" ||
			!(event.target instanceof Node) ||
			!speedArea?.contains(event.target)
		) {
			return
		}
		event.preventDefault()
		event.stopImmediatePropagation()
		void closeSpeedPanel()
	}}
	onpointerdown={(event) => {
		if (
			isSpeedPanelOpen &&
			event.target instanceof Node &&
			!speedArea?.contains(event.target)
		) {
			isSpeedPanelOpen = false
		}
	}} />

{#snippet playbackTimeline()}
	<button
		class="button-icon"
		type="button"
		aria-label={playLabel}
		aria-busy={isAnimationLoading}
		data-tooltip={playLabel}
		disabled={!playback}
		onclick={togglePlay}>
		{#if isAnimationLoading && isPlaying}
			<Icon svg={circleNotchIcon} class="spinner" size={16} aria-hidden="true" />
		{:else}
			{#key isPlaying}
				<Icon
					svg={isPlaying ? pauseIcon : playIcon}
					class="animate-pop"
					size={16}
					aria-hidden="true" />
			{/key}
		{/if}
	</button>
	<label class="flex min-w-0 flex-1 items-center gap-2">
		<span class="sr-only">Animation time</span>
		<input
			class="range min-w-0 flex-1"
			type="range"
			min="0"
			max={lengthSeconds}
			step="0.01"
			value={timeSeconds}
			disabled={!canSeek}
			style:--range-fill={`${fillPercent}%`}
			aria-valuetext={timeLabel}
			onpointerdown={pauseForSeek}
			oninput={(event) => playback?.seek(event.currentTarget.valueAsNumber)}
			onchange={resumeAfterSeek}
			onpointercancel={resumeAfterSeek} />
		<span class="shrink-0 font-mono text-2xs text-fg-muted tabular-nums"
			>{isAnimationLoading ? "Loading…" : timeLabel}</span>
	</label>
	<div bind:this={speedArea} class="relative shrink-0">
		<button
			bind:this={speedButton}
			class="button-icon"
			type="button"
			aria-label={`Playback speed: ${speedLabel}`}
			aria-expanded={isSpeedPanelOpen}
			aria-controls={speedPanelId}
			data-open={isSpeedPanelOpen || undefined}
			data-tooltip={`Playback speed: ${speedLabel}`}
			disabled={!playback}
			onclick={() => void toggleSpeedPanel()}>
			<Icon svg={speedometerIcon} size={16} aria-hidden="true" />
		</button>
		{#if isSpeedPanelOpen}
			<div
				id={speedPanelId}
				class="absolute right-0 bottom-full z-20 mb-2 flex w-52 animate-enter flex-col gap-1 rounded-lg border border-line-strong bg-overlay py-2 pr-2 pl-3 shadow-float"
				role="group"
				aria-label="Playback speed"
				data-closes-on-escape>
				<div class="flex items-center gap-2 text-xs">
					<span id={speedLabelId} class="mr-auto text-fg-muted">Speed</span>
					<span class="font-mono text-2xs text-fg tabular-nums"
						>{speedLabel}</span>
					<button
						class="button-ghost-sm"
						type="button"
						aria-label={`Reset speed to ${speedFactor.format(defaultSpeed)}×`}
						data-tooltip={`Reset to ${speedFactor.format(defaultSpeed)}×`}
						disabled={speed === defaultSpeed}
						onclick={() => playback?.setSpeed(defaultSpeed)}>
						<Icon
							svg={arrowCounterClockwiseIcon}
							size={14}
							aria-hidden="true" />
					</button>
				</div>
				<input
					bind:this={speedInput}
					class="range mr-1"
					aria-labelledby={speedLabelId}
					type="range"
					min={minSpeed}
					max={maxSpeed}
					step={speedStep}
					value={speed}
					style:--range-fill={`${((speed - minSpeed) / (maxSpeed - minSpeed)) * 100}%`}
					aria-valuetext={speedLabel}
					oninput={(event) => setSpeed(event.currentTarget.valueAsNumber)} />
			</div>
		{/if}
	</div>
{/snippet}

<div
	class={["flex gap-2", isOneRow ? "items-center" : "flex-col"]}
	role="group"
	aria-label="Animation">
	<div class={["flex items-center gap-2", isOneRow ? "min-w-0 flex-1" : ""]}>
		{#if hasAnimationPicker}
			<AnimationPicker
				{playback}
				{library}
				className={isWide ? "w-48" : "min-w-0 flex-1"} />
		{/if}
		{#if isOneRow}
			{@render playbackTimeline()}
		{/if}
	</div>
	{#if !isOneRow}
		<div class="flex items-center gap-2">
			{@render playbackTimeline()}
		</div>
	{/if}
</div>
{#if isAnimationFailed}
	<p class="mt-2 animate-enter text-xs text-danger-muted" role="alert">
		Could not play this animation. Select another one.
	</p>
{/if}
