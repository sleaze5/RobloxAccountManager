<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import pauseIcon from "@phosphor-icons/core/regular/pause.svg?raw"
	import playIcon from "@phosphor-icons/core/regular/play.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Select from "../shared/Select.svelte"
	import type { AnimationPlayback } from "./model-subject"

	// AnimationControls plays the animations and emotes of the avatar in a 3D
	// viewer: the animation, a play button next to the seek bar, and the speed.
	// A wide bar, such as under an expanded viewer, puts them in one row.
	// playback is undefined until the avatar shows, which disables the
	// controls, so that the bar keeps its place while the avatar loads.
	let {
		playback,
		isWide = false,
	}: { playback: AnimationPlayback | undefined; isWide?: boolean } = $props()

	const speedOptions = [0.25, 0.5, 1, 1.5, 2].map((speed) => ({
			value: speed,
			label: `${speed}×`,
		})),
		seconds = new Intl.NumberFormat("en-US", {
			minimumFractionDigits: 2,
			maximumFractionDigits: 2,
		})

	// Whether the animation played when a drag of the seek bar began, so that
	// it plays on once the drag ends.
	let wasPlayingBeforeSeek = false

	const animationOptions = $derived(
			(playback?.animationChoices ?? []).map((choice) => ({
				value: choice.id,
				label: choice.label,
				group: choice.isEmote ? "Emotes" : "Animations",
			})),
		),
		isPlaying = $derived(!!playback?.isPlaying),
		isAnimationLoading = $derived(!!playback?.isAnimationLoading),
		timeSeconds = $derived(playback?.timeSeconds ?? 0),
		lengthSeconds = $derived(playback?.lengthSeconds ?? 0),
		canSeek = $derived(!!playback && !isAnimationLoading && lengthSeconds > 0),
		timeLabel = $derived(
			`${seconds.format(timeSeconds)} / ${seconds.format(lengthSeconds)} s`,
		),
		fillPercent = $derived(canSeek ? (timeSeconds / lengthSeconds) * 100 : 0),
		playLabel = $derived(isPlaying ? "Pause animation" : "Play animation")

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
</script>

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
{/snippet}

<div
	class={["flex gap-2", isWide ? "items-center" : "flex-col"]}
	role="group"
	aria-label="Animation">
	<div class={["flex items-center gap-2", isWide ? "min-w-0 flex-1" : ""]}>
		<Select
			label="Animation"
			value={playback?.animationId ?? ""}
			options={animationOptions}
			disabled={!playback}
			className={isWide ? "w-40" : "min-w-0 flex-1"}
			onChange={(animationId) => playback?.chooseAnimation(animationId)} />
		{#if isWide}
			{@render playbackTimeline()}
		{/if}
		<Select
			label="Playback speed"
			value={playback?.speed ?? 1}
			options={speedOptions}
			disabled={!playback}
			className="w-20"
			onChange={(speed) => playback?.setSpeed(speed)} />
	</div>
	{#if !isWide}
		<div class="flex items-center gap-2">
			{@render playbackTimeline()}
		</div>
	{/if}
</div>
