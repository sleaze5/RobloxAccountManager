<script lang="ts">
	import lockIcon from "@phosphor-icons/core/regular/lock.svg?raw"
	import prohibitIcon from "@phosphor-icons/core/regular/prohibit.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { Moderation } from "../backend/bridge"
	import { CountdownClock } from "../shared/countdown-clock.svelte"
	import {
		banEndsAtMs,
		moderationStatus,
		resolvesInBrowser,
	} from "./moderation.svelte"

	let { moderation }: { moderation: Moderation | null } = $props()

	const clock = new CountdownClock(),
		status = $derived(moderationStatus(moderation, clock.nowMs))

	$effect(() => clock.track(banEndsAtMs(moderation)))
</script>

<small class="flex w-full min-w-0 items-center gap-1 text-2xs text-fg">
	<Icon
		svg={resolvesInBrowser(moderation)
			? lockIcon
			: status.tone === "warning"
				? warningIcon
				: prohibitIcon}
		class={[
			"shrink-0 animate-pop",
			status.tone === "warning" ? "text-warning" : "text-danger-muted",
		]}
		size={12}
		aria-hidden="true" />
	<span
		class={[
			"shrink-0 font-medium",
			status.tone === "warning" ? "text-warning" : "text-danger-muted",
		]}>{status.label}</span>
	{#if status.detail}<span class="min-w-0 truncate font-mono tabular-nums"
			>{status.detail}</span
		>{/if}
</small>
