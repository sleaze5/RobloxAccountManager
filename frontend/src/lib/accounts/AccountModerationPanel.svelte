<script lang="ts">
	import globeHemisphereWestIcon from "@phosphor-icons/core/regular/globe-hemisphere-west.svg?raw"
	import lockIcon from "@phosphor-icons/core/regular/lock.svg?raw"
	import prohibitIcon from "@phosphor-icons/core/regular/prohibit.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
	import { ModerationKind } from "../backend/bridge"
	import type { Moderation } from "../backend/bridge"
	import { CountdownClock } from "../shared/countdown-clock.svelte"
	import { formatClock } from "../shared/duration"
	import {
		banEndsAtMs,
		banLength,
		banRemainingSeconds,
		canReactivate,
		resolvesInBrowser,
	} from "./moderation.svelte"

	let {
		moderation,
		onOpenBrowser,
	}: { moderation: Moderation | null; onOpenBrowser: () => void } = $props()

	const clock = new CountdownClock(),
		kind = $derived(moderation?.kind),
		reactivatable = $derived(canReactivate(moderation, clock.nowMs)),
		inBrowser = $derived(resolvesInBrowser(moderation)),
		remainingSeconds = $derived(
			moderation?.endsAtMs ? banRemainingSeconds(moderation, clock.nowMs) : 0,
		),
		isWarning = $derived(
			kind === ModerationKind.ModerationWarned || reactivatable || inBrowser,
		),
		icon = $derived(inBrowser ? lockIcon : isWarning ? warningIcon : prohibitIcon),
		title = $derived.by(() => {
			if (moderation?.title) return moderation.title
			switch (kind) {
				case ModerationKind.ModerationWarned:
					return "Roblox warned this account"
				case ModerationKind.ModerationTerminated:
					return "Roblox terminated this account"
				case ModerationKind.ModerationBanned:
					return moderation?.endsAtMs
						? `Banned for ${banLength(moderation)}`
						: "Roblox banned this account"
				case ModerationKind.ModerationLocked:
					return "Account locked"
				default:
					return "Roblox restricted this account"
			}
		}),
		description = $derived.by(() => {
			switch (kind) {
				case ModerationKind.ModerationWarned:
					return "Reactivate to accept the warning and use this account again."
				case ModerationKind.ModerationTerminated:
					return "A terminated account cannot be reactivated."
				case ModerationKind.ModerationBanned:
					if (reactivatable) {
						return "The ban ended. Reactivate to use this account again."
					}
					return moderation?.endsAtMs
						? "You can reactivate this account when the ban ends."
						: "Roblox did not share when this ban ends."
				case ModerationKind.ModerationLocked:
					return "Roblox wants you to confirm you're human. Open the account in the browser and finish the check on Roblox."
				default:
					return "Open the account in the browser to see what Roblox needs and resolve it there."
			}
		})

	$effect(() => clock.track(banEndsAtMs(moderation)))
</script>

<section
	class={[
		"flex max-w-2xl animate-enter flex-col gap-3 rounded-lg border p-3",
		isWarning ? "border-warning/30 bg-warning/5" : "border-danger/40 bg-danger/5",
	]}
	aria-labelledby="account-moderation-title">
	<div class="flex items-start gap-3">
		<span
			class={[
				"grid size-8 shrink-0 animate-bloom place-items-center rounded-md border",
				isWarning
					? "border-warning/30 text-warning"
					: "border-danger/40 text-danger-muted",
			]}
			aria-hidden="true">
			<Icon svg={icon} size={16} aria-hidden="true" />
		</span>
		<div class="flex min-w-0 flex-1 flex-col gap-0.5">
			<h3 id="account-moderation-title" class="text-sm font-semibold text-fg">
				{title}
			</h3>
			<p class="text-xs text-fg-muted">{description}</p>
		</div>
		{#if remainingSeconds > 0}
			<div class="flex shrink-0 flex-col items-end gap-0.5" role="timer">
				<span class="text-2xs text-fg-muted">Ends in</span>
				<span
					class="font-mono text-sm font-semibold text-danger-muted tabular-nums"
					>{formatClock(remainingSeconds, "days")}</span>
			</div>
		{:else if inBrowser}
			<button
				class="button-primary shrink-0"
				type="button"
				onclick={onOpenBrowser}>
				<Icon svg={globeHemisphereWestIcon} size={14} aria-hidden="true" />
				Open in browser
			</button>
		{/if}
	</div>
	{#if moderation?.message}
		<figure class="flex flex-col gap-1 border-t border-line-subtle pt-2">
			<figcaption class="text-2xs font-medium text-fg-muted">
				Message from Roblox
			</figcaption>
			<blockquote
				class="text-xs leading-relaxed whitespace-pre-line text-fg select-text">
				{moderation.message}
			</blockquote>
		</figure>
	{/if}
	{#if moderation && (moderation.beginsAtMs || moderation.endsAtMs)}
		<dl
			class="flex flex-wrap gap-x-6 gap-y-1 border-t border-line-subtle pt-2 text-xs">
			{#if moderation.beginsAtMs}
				<div class="flex items-center gap-2">
					<dt class="text-fg-muted">Started</dt>
					<dd><Timestamp value={moderation.beginsAtMs} /></dd>
				</div>
			{/if}
			{#if moderation.endsAtMs}
				<div class="flex items-center gap-2">
					<dt class="text-fg-muted">
						{remainingSeconds > 0 ? "Ends" : "Ended"}
					</dt>
					<dd><Timestamp value={moderation.endsAtMs} /></dd>
				</div>
			{/if}
		</dl>
	{/if}
</section>
