<script lang="ts">
	import bellIcon from "@phosphor-icons/core/regular/bell.svg?raw"
	import caretUpIcon from "@phosphor-icons/core/regular/caret-up.svg?raw"
	import checkCircleIcon from "@phosphor-icons/core/regular/check-circle.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import warningCircleIcon from "@phosphor-icons/core/regular/warning-circle.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import type { Attachment } from "svelte/attachments"
	import { SvelteSet } from "svelte/reactivity"
	import Icon from "../shared/Icon.svelte"
	import { snackbarIn, snackbarOut, toastMove } from "../shared/presence"
	import type {
		AppNotification,
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

	const maxCollapsedNotificationCount = 3

	let { center }: { center: NotificationCenter } = $props(),
		runningAction = $state(""),
		isStackExpanded = $state(false)

	const wrappedNotificationIds = new SvelteSet<string>(),
		hiddenNotificationCount = $derived(
			Math.max(0, center.items.length - maxCollapsedNotificationCount),
		),
		visibleNotifications = $derived(
			isStackExpanded
				? center.items
				: center.items.slice(-maxCollapsedNotificationCount),
		)

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

	function dismissOnEscape(
		event: KeyboardEvent,
		notification: AppNotification,
	): void {
		if (event.key !== "Escape" || !notification.dismissible || runningAction) {
			return
		}
		event.stopPropagation()
		center.dismiss(notification.id)
	}

	// A pill only fits one line of text, so a snackbar whose message wraps switches to rounded corners.
	function trackMessageWrapping(notificationId: string): Attachment<HTMLElement> {
		return (message) => {
			const observer = new ResizeObserver(() => {
				const style = getComputedStyle(message),
					textHeightPixels =
						message.clientHeight -
						Number.parseFloat(style.paddingTop) -
						Number.parseFloat(style.paddingBottom)
				if (textHeightPixels > Number.parseFloat(style.lineHeight))
					wrappedNotificationIds.add(notificationId)
				else wrappedNotificationIds.delete(notificationId)
			})
			observer.observe(message)
			return () => {
				observer.disconnect()
				wrappedNotificationIds.delete(notificationId)
			}
		}
	}
</script>

<section
	class="pointer-events-none fixed inset-x-0 bottom-9 z-30 flex flex-col items-center gap-2 px-4"
	aria-label="Notifications"
	aria-live="polite">
	{#if hiddenNotificationCount > 0}
		<button
			class="pointer-events-auto inline-flex h-6 items-center gap-1 rounded-full border border-line-strong bg-overlay px-2.5 text-2xs font-medium text-fg-muted shadow-float transition-colors duration-100 hover:bg-raised-hover hover:text-fg"
			type="button"
			aria-expanded={isStackExpanded}
			onclick={() => (isStackExpanded = !isStackExpanded)}>
			<Icon
				svg={caretUpIcon}
				size={12}
				class={[
					"transition-transform duration-100",
					isStackExpanded && "rotate-180",
				]}
				aria-hidden="true" />
			{isStackExpanded ? "Show fewer" : `Show ${hiddenNotificationCount} more`}
		</button>
	{/if}
	{#each visibleNotifications as notification (notification.id)}
		{@const isPill = !wrappedNotificationIds.has(notification.id)}
		<article
			class={[
				"pointer-events-auto flex w-fit max-w-lg min-w-72 items-center gap-2.5 border border-line-strong bg-overlay py-1.5 pr-1.5 pl-3 shadow-float",
				isPill ? "rounded-full" : "rounded-xl",
			]}
			role="status"
			in:snackbarIn
			out:snackbarOut
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
			<span
				class={[
					"grid h-6 shrink-0 place-items-center self-start",
					toneColors[notification.tone],
				]}>
				<Icon svg={toneIcons[notification.tone]} size={16} aria-hidden="true" />
			</span>
			<p
				class="min-w-0 flex-1 py-0.5 text-xs leading-5"
				{@attach trackMessageWrapping(notification.id)}>
				<strong class="font-semibold text-fg">{notification.title}</strong>
				<span class="text-fg-muted">{notification.message}</span>
			</p>
			{#if notification.actions.length > 0}
				<div class="flex shrink-0 items-center gap-1.5">
					{#each notification.actions as action, index}
						<button
							class={[
								action.emphasis
									? "button-primary h-6 px-2.5"
									: "button h-6 px-2.5",
								isPill && "rounded-full",
							]}
							type="button"
							disabled={runningAction !== ""}
							onkeydown={(event) => dismissOnEscape(event, notification)}
							onclick={() =>
								void runAction(notification.id, index, action)}>
							{#if runningAction === `${notification.id}:${index}`}
								<Icon
									svg={circleNotchIcon}
									class="spinner"
									size={12}
									aria-hidden="true" />
							{/if}
							{action.label}
						</button>
					{/each}
				</div>
			{/if}
			{#if notification.dismissible}
				<button
					class={[
						"button-close text-fg-faint hover:bg-hover hover:text-fg",
						isPill && "rounded-full",
					]}
					type="button"
					aria-label={`Dismiss ${notification.title}`}
					disabled={runningAction !== ""}
					onkeydown={(event) => dismissOnEscape(event, notification)}
					onclick={() => center.dismiss(notification.id)}>
					<Icon svg={xIcon} size={14} aria-hidden="true" />
				</button>
			{/if}
		</article>
	{/each}
</section>
