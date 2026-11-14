<script lang="ts">
	import microphoneIcon from "@phosphor-icons/core/regular/microphone.svg?raw"
	import microphoneSlashIcon from "@phosphor-icons/core/regular/microphone-slash.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { VoiceChatState } from "../backend/bridge"
	import type { VoiceChat } from "../backend/bridge"
	import { CountdownClock } from "../shared/countdown-clock.svelte"
	import { formatClock } from "../shared/duration"

	let { voiceChat, label }: { voiceChat: VoiceChat | null; label: string } = $props()

	const states: Record<VoiceChatState, { tone: string; label: string }> = {
		[VoiceChatState.$zero]: { tone: "muted", label: "Unavailable" },
		[VoiceChatState.VoiceChatOn]: { tone: "", label: "On" },
		[VoiceChatState.VoiceChatOff]: { tone: "", label: "Off" },
		[VoiceChatState.VoiceChatIneligible]: { tone: "muted", label: "Not eligible" },
		[VoiceChatState.VoiceChatSuspended]: { tone: "warning", label: "Suspended" },
	}

	const clock = new CountdownClock(),
		status = $derived(states[voiceChat?.state ?? VoiceChatState.$zero]),
		suspendedUntil = $derived(
			voiceChat?.state === VoiceChatState.VoiceChatSuspended
				? voiceChat.suspendedUntilMs
				: 0,
		)

	$effect(() => clock.track(suspendedUntil))
</script>

<div data-tone={status.tone || undefined}>
	<dt>
		{#if voiceChat?.state === VoiceChatState.VoiceChatOn}<Icon
				svg={microphoneIcon}
				size={16}
				aria-hidden="true" />{:else}<Icon
				svg={microphoneSlashIcon}
				size={16}
				aria-hidden="true" />{/if}
		{label}
	</dt>
	<dd>
		<span>{status.label}</span>
		{#if suspendedUntil > clock.nowMs}
			<small class="tabular-nums"
				>Ends in <time datetime={new Date(suspendedUntil).toISOString()}
					>{formatClock(
						Math.ceil((suspendedUntil - clock.nowMs) / 1000),
						"days",
					)}</time
				></small>
		{:else if suspendedUntil}
			<small>Ended. Refresh to update.</small>
		{/if}
	</dd>
</div>
