<script lang="ts">
	import MicrophoneIcon from "phosphor-svelte/lib/MicrophoneIcon"
	import MicrophoneSlashIcon from "phosphor-svelte/lib/MicrophoneSlashIcon"
	import { VoiceChatState } from "../backend/bridge"
	import type { VoiceChat } from "../backend/bridge"
	import { formatClock } from "../shared/duration"

	let { voiceChat, label }: { voiceChat: VoiceChat | null; label: string } = $props()

	const states: Record<VoiceChatState, { tone: string; label: string }> = {
		[VoiceChatState.$zero]: { tone: "muted", label: "Unavailable" },
		[VoiceChatState.VoiceChatOn]: { tone: "", label: "On" },
		[VoiceChatState.VoiceChatOff]: { tone: "", label: "Off" },
		[VoiceChatState.VoiceChatIneligible]: { tone: "muted", label: "Not eligible" },
		[VoiceChatState.VoiceChatSuspended]: { tone: "warning", label: "Suspended" },
	}

	let now = $state(Date.now())

	const status = $derived(states[voiceChat?.state ?? VoiceChatState.$zero]),
		suspendedUntil = $derived(
			voiceChat?.state === VoiceChatState.VoiceChatSuspended
				? voiceChat.suspendedUntilMs
				: 0,
		)

	$effect(() => {
		const until = suspendedUntil,
			current = Date.now()
		now = current
		if (until <= current) return
		const timer = window.setInterval(() => {
			now = Date.now()
			if (now >= until) window.clearInterval(timer)
		}, 1000)
		return () => window.clearInterval(timer)
	})
</script>

<div class={status.tone}>
	<dt>
		{#if voiceChat?.state === VoiceChatState.VoiceChatOn}<MicrophoneIcon
				size={16}
				aria-hidden="true" />{:else}<MicrophoneSlashIcon
				size={16}
				aria-hidden="true" />{/if}
		{label}
	</dt>
	<dd>
		<span>{status.label}</span>
		{#if suspendedUntil > now}
			<small class="profile-countdown"
				>Ends in <time datetime={new Date(suspendedUntil).toISOString()}
					>{formatClock(
						Math.ceil((suspendedUntil - now) / 1000),
						"days",
					)}</time
				></small>
		{:else if suspendedUntil}
			<small>Ended. Refresh to update.</small>
		{/if}
	</dd>
</div>
