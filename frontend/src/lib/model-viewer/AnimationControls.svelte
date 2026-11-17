<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import pauseIcon from "@phosphor-icons/core/regular/pause.svg?raw"
	import playIcon from "@phosphor-icons/core/regular/play.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Select from "../shared/Select.svelte"
	import type { AnimationPlayback } from "./model-subject"

	// AnimationControls plays the animations and emotes of the avatar in a 3D
	// viewer: a play button, the animation, its speed, and a seek bar. A wide
	// bar, such as under an expanded viewer, puts them in one row.
	let {
		playback,
		isWide = false,
	}: { playback: AnimationPlayback; isWide?: boolean } = $props()

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
			playback.animationChoices.map((choice) => ({
				value: choice.id,
				label: choice.label,
				group: choice.isEmote ? "Emotes" : "Animations",
			})),
		),
		canSeek = $derived(!playback.isAnimationLoading && playback.lengthSeconds > 0),
		timeLabel = $derived(
			`${seconds.format(playback.timeSeconds)} / ${seconds.format(playback.lengthSeconds)} s`,
		),
		fillPercent = $derived(
			canSeek ? (playback.timeSeconds / playback.lengthSeconds) * 100 : 0,
		)

	function togglePlay(): void {
		if (playback.isPlaying) playback.pause()
		else playback.play()
	}
</script>

{#snippet seekBar()}
	<label class="flex min-w-0 flex-1 items-center gap-2">
		<span class="sr-only">Animation time</span>
		<input
			class="range min-w-0 flex-1"
			type="range"
			min="0"
			max={playback.lengthSeconds}
			step="0.01"
			value={playback.timeSeconds}
			disabled={!canSeek}
			style:--range-fill={`${fillPercent}%`}
			aria-valuetext={timeLabel}
			onpointerdown={() => {
				wasPlayingBeforeSeek = playback.isPlaying
				playback.pause()
			}}
			oninput={(event) => playback.seek(event.currentTarget.valueAsNumber)}
			onchange={() => {
				if (wasPlayingBeforeSeek) playback.play()
				wasPlayingBeforeSeek = false
			}} />
		<span class="shrink-0 font-mono text-2xs text-fg-muted tabular-nums"
			>{playback.isAnimationLoading ? "Loading…" : timeLabel}</span>
	</label>
{/snippet}

<div
	class={["flex gap-2", isWide ? "items-center" : "flex-col"]}
	role="group"
	aria-label="Animation">
	<div class={["flex items-center gap-2", isWide ? "min-w-0 flex-1" : ""]}>
		<button
			class="button-icon"
			type="button"
			aria-label={playback.isPlaying ? "Pause animation" : "Play animation"}
			aria-busy={playback.isAnimationLoading}
			data-tooltip={playback.isPlaying ? "Pause" : "Play"}
			onclick={togglePlay}>
			{#if playback.isAnimationLoading && playback.isPlaying}
				<Icon
					svg={circleNotchIcon}
					class="spinner"
					size={16}
					aria-hidden="true" />
			{:else}
				{#key playback.isPlaying}
					<Icon
						svg={playback.isPlaying ? pauseIcon : playIcon}
						class="animate-pop"
						size={16}
						aria-hidden="true" />
				{/key}
			{/if}
		</button>
		<Select
			label="Animation"
			value={playback.animationId}
			options={animationOptions}
			className={isWide ? "w-40" : "min-w-0 flex-1"}
			onChange={(animationId) => playback.chooseAnimation(animationId)} />
		{#if isWide}
			{@render seekBar()}
		{/if}
		<Select
			label="Playback speed"
			value={playback.speed}
			options={speedOptions}
			className="w-20"
			onChange={(speed) => playback.setSpeed(speed)} />
	</div>
	{#if !isWide}
		{@render seekBar()}
	{/if}
</div>
