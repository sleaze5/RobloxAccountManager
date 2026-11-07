<script lang="ts">
	import bellIcon from "@phosphor-icons/core/regular/bell.svg?raw"
	import checkCircleIcon from "@phosphor-icons/core/regular/check-circle.svg?raw"
	import warningCircleIcon from "@phosphor-icons/core/regular/warning-circle.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { toastIn, toastMove, toastOut } from "../shared/presence"
	import type {
		NotificationAction,
		NotificationCenter,
	} from "./notification-center.svelte"

	const toneIcons = {
		error: warningCircleIcon,
		info: bellIcon,
		success: checkCircleIcon,
		warning: warningIcon,
	}

	const toneColors = {
		error: "text-danger",
		info: "text-fg-muted",
		success: "text-success",
		warning: "text-warning",
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

<section
	class="pointer-events-none fixed top-11 right-2 z-30 flex w-80 flex-col gap-2"
	aria-label="Notifications"
	aria-live="polite">
	{#each center.items as notification (notification.id)}
		{@const toneIcon = toneIcons[notification.tone]}
		<article
			class="pointer-events-auto flex gap-2 rounded-xl border border-line-strong bg-overlay p-2 shadow-float"
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
			<div
				class={[
					"grid size-6 shrink-0 place-items-center rounded-md border border-line bg-raised",
					toneColors[notification.tone],
				]}>
				<Icon svg={toneIcon} size={16} aria-hidden="true" />
			</div>
			<div class="min-w-0 flex-1">
				<strong class="block text-sm font-semibold text-fg"
					>{notification.title}</strong>
				<p class="mt-0.5 text-xs leading-snug text-fg-muted">
					{notification.message}
				</p>
				{#if notification.actions.length > 0}
					<div class="mt-1 flex justify-end gap-2">
						{#each notification.actions as action, index}
							<button
								class={action.emphasis
									? "button-primary h-6 px-2"
									: "button h-6"}
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
					class="button-close text-fg-faint hover:text-fg"
					type="button"
					aria-label={`Dismiss ${notification.title}`}
					disabled={runningAction !== ""}
					onclick={() => center.dismiss(notification.id)}>
					<Icon svg={xIcon} size={14} aria-hidden="true" />
				</button>
			{/if}
		</article>
	{/each}
</section>
