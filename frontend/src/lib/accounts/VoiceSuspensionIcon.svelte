<script lang="ts">
	import microphoneSlashIcon from "@phosphor-icons/core/regular/microphone-slash.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { VoiceSuspension } from "../backend/bridge"
	import { CountdownClock } from "../shared/countdown-clock.svelte"
	import { formatClock } from "../shared/duration"

	let { voiceSuspension }: { voiceSuspension: VoiceSuspension } = $props()

	const clock = new CountdownClock(),
		endsAtMs = $derived(voiceSuspension.endsAtMs ?? 0),
		remainingSeconds = $derived(Math.ceil((endsAtMs - clock.nowMs) / 1000))

	$effect(() => clock.track(endsAtMs))
</script>

{#if !endsAtMs || remainingSeconds > 0}
	<span
		class="ml-auto grid shrink-0 animate-pop place-items-center text-warning"
		data-tooltip={endsAtMs
			? `Voice chat suspended, ends in ${formatClock(remainingSeconds, "days")}`
			: "Voice chat suspended"}>
		<Icon svg={microphoneSlashIcon} size={12} aria-hidden="true" />
	</span>
{/if}
