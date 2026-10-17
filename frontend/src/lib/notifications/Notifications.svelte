<script lang="ts">
	import BellIcon from "phosphor-svelte/lib/BellIcon"
	import CheckCircleIcon from "phosphor-svelte/lib/CheckCircleIcon"
	import WarningCircleIcon from "phosphor-svelte/lib/WarningCircleIcon"
	import WarningIcon from "phosphor-svelte/lib/WarningIcon"
	import XIcon from "phosphor-svelte/lib/XIcon"
	import { toastIn, toastMove, toastOut } from "../shared/presence"
	import type {
		NotificationAction,
		NotificationCenter,
	} from "./notification-center.svelte"

	const toneIcons = {
		error: WarningCircleIcon,
		info: BellIcon,
		success: CheckCircleIcon,
		warning: WarningIcon,
	}

	let { center }: { center: NotificationCenter } = $props(),
		runningAction = $state("")

	async function runAction(
		notificationId: string,
		index: number,
		action: NotificationAction,
	): Promise<void> {
		if (runningAction) {
			return
		}
		runningAction = `${notificationId}:${index}`
		try {
			await action.onClick()
		} finally {
			runningAction = ""
		}
	}
</script>

<section class="notification-region" aria-label="Notifications" aria-live="polite">
	{#each center.items as notification (notification.id)}
		{@const ToneIcon = toneIcons[notification.tone]}
		<article
			class="notification-toast"
			role="status"
			in:toastIn
			out:toastOut
			animate:toastMove
			onpointerenter={() => center.pause(notification.id)}
			onpointerleave={(event) => {
				if (!event.currentTarget.contains(document.activeElement))
					center.resume(notification.id)
			}}
			onfocusin={() => center.pause(notification.id)}
			onfocusout={(event) => {
				const next = event.relatedTarget
				if (
					!(next instanceof Node && event.currentTarget.contains(next)) &&
					!event.currentTarget.matches(":hover")
				)
					center.resume(notification.id)
			}}>
			<div class="notification-icon {notification.tone}">
				<ToneIcon size={16} aria-hidden="true" />
			</div>
			<div class="notification-content">
				<strong>{notification.title}</strong>
				<p>{notification.message}</p>
				{#if notification.actions.length > 0}
					<div class="notification-actions">
						{#each notification.actions as action, index}
							<button
								class={action.emphasis
									? "primary-action compact"
									: "control-button text-button compact"}
								type="button"
								disabled={runningAction !== ""}
								onclick={() =>
									void runAction(notification.id, index, action)}>
								{action.label}
							</button>
						{/each}
					</div>
				{/if}
			</div>
			{#if notification.dismissible}
				<button
					class="notification-dismiss"
					type="button"
					aria-label={`Dismiss ${notification.title}`}
					disabled={runningAction !== ""}
					onclick={() => center.dismiss(notification.id)}>
					<XIcon size={14} aria-hidden="true" />
				</button>
			{/if}
		</article>
	{/each}
</section>
