<script lang="ts">
	import microphoneSlashIcon from "@phosphor-icons/core/regular/microphone-slash.svg?raw"
	import { untrack } from "svelte"
	import Icon from "../shared/Icon.svelte"
	import type { VoiceSuspension } from "../backend/bridge"
	import { CountdownClock } from "../shared/countdown-clock.svelte"
	import { formatClock } from "../shared/duration"

	type Verification = "checking" | "still-reported" | "failed"

	let {
		voiceSuspension,
		onVerify,
	}: { voiceSuspension: VoiceSuspension; onVerify: () => Promise<boolean> } = $props()

	// Roblox can lift a suspension a little after its end time, so an ended
	// suspension is checked again a few times before it waits for a refresh.
	const verificationAttemptLimit = 5,
		verificationRetryDelayMs = 60_000

	let verification = $state<Verification>("checking")

	const clock = new CountdownClock(),
		endsAtMs = $derived(voiceSuspension.endsAtMs ?? 0),
		remainingSeconds = $derived(Math.ceil((endsAtMs - clock.nowMs) / 1000)),
		hasEnded = $derived(endsAtMs > 0 && remainingSeconds <= 0),
		tooltip = $derived.by(() => {
			if (!endsAtMs) return "Voice chat suspended"
			if (!hasEnded) {
				return `Voice chat suspended, ends in ${formatClock(remainingSeconds, "days")}`
			}
			switch (verification) {
				case "checking":
					return "Voice chat suspension ended, checking Roblox"
				case "still-reported":
					return "Voice chat suspension ended, but Roblox still reports it"
				case "failed":
					return "Voice chat suspension ended, could not check Roblox"
			}
		})

	$effect(() => clock.track(endsAtMs))

	$effect(() => {
		if (!hasEnded) return
		let isCancelled = false,
			attemptCount = 0,
			retryTimer = 0
		const verify = async (): Promise<void> => {
			attemptCount++
			verification = "checking"
			const hasAnswered = await untrack(onVerify)
			if (isCancelled) return
			verification = hasAnswered ? "still-reported" : "failed"
			if (attemptCount < verificationAttemptLimit) {
				retryTimer = window.setTimeout(verify, verificationRetryDelayMs)
			}
		}
		void verify()
		return () => {
			isCancelled = true
			window.clearTimeout(retryTimer)
		}
	})
</script>

<span
	class="ml-auto grid shrink-0 animate-pop place-items-center text-warning"
	data-tooltip={tooltip}>
	<Icon svg={microphoneSlashIcon} size={14} aria-hidden="true" />
</span>
