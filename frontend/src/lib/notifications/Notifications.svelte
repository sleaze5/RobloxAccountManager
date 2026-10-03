<script lang="ts">
	import Bell from "@lucide/svelte/icons/bell"
	import X from "@lucide/svelte/icons/x"
	import { toastIn, toastOut } from "../shared/presence"
	import type {
		NotificationAction,
		NotificationCenter,
	} from "./notification-center.svelte"

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
		<article class="notification-toast" role="status" in:toastIn out:toastOut>
			<div class="notification-icon"><Bell size={15} aria-hidden="true" /></div>
			<div class="notification-content">
				<strong>{notification.title}</strong>
				<p>{notification.message}</p>
				{#if notification.actions.length > 0}
					<div class="notification-actions">
						{#each notification.actions as action, index}
							<button
								class:primary-action={action.emphasis}
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
					<X size={14} aria-hidden="true" />
				</button>
			{/if}
		</article>
	{/each}
</section>
