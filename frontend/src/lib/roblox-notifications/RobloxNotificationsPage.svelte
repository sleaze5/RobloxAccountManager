<script lang="ts">
	import bellIcon from "@phosphor-icons/core/regular/bell.svg?raw"
	import broomIcon from "@phosphor-icons/core/regular/broom.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import funnelIcon from "@phosphor-icons/core/regular/funnel.svg?raw"
	import { Browser } from "@wailsio/runtime"
	import Icon from "../shared/Icon.svelte"
	import AccountPageHeader from "../accounts/AccountPageHeader.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import Select from "../shared/Select.svelte"
	import { onMount, untrack } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { RobloxNotificationCategory } from "../backend/bridge"
	import type { RobloxNotificationUser } from "../backend/bridge"
	import { errorMessage } from "../shared/errors"
	import DismissAllRobloxNotificationsDialog from "./DismissAllRobloxNotificationsDialog.svelte"
	import RobloxNotificationCard from "./RobloxNotificationCard.svelte"
	import { RobloxNotificationState } from "./roblox-notification-state.svelte"

	let {
		store,
		onBack,
		onOpenChat,
		onFollowUser,
		onSearchPlace,
	}: {
		store: AccountStore
		onBack: () => void
		onOpenChat: (userId: number) => void
		onFollowUser: (user: RobloxNotificationUser) => void
		onSearchPlace: (placeId: number) => void
	} = $props()

	const account = untrack(() => store.selectedAccount),
		stream = untrack(() => new RobloxNotificationState(account.id, store)),
		categories: {
			value: RobloxNotificationCategory
			label: string
			noun: string
		}[] = [
			{
				value: RobloxNotificationCategory.$zero,
				label: "All types",
				noun: "notifications",
			},
			{
				value: RobloxNotificationCategory.RobloxNotificationInvites,
				label: "Invites",
				noun: "invites",
			},
			{
				value: RobloxNotificationCategory.RobloxNotificationFriends,
				label: "Friends",
				noun: "friend notifications",
			},
			{
				value: RobloxNotificationCategory.RobloxNotificationCommunities,
				label: "Communities",
				noun: "community notifications",
			},
			{
				value: RobloxNotificationCategory.RobloxNotificationExperiences,
				label: "Experiences",
				noun: "experience notifications",
			},
			{
				value: RobloxNotificationCategory.RobloxNotificationOther,
				label: "Other",
				noun: "other notifications",
			},
		],
		categoryNoun = $derived(
			categories.find((category) => category.value === stream.category)?.noun ??
				"notifications",
		)

	let isDismissAllOpen = $state(false)

	async function openLink(url: string): Promise<void> {
		try {
			await Browser.OpenURL(url)
		} catch (error) {
			stream.error = errorMessage(error, "Could not open the link. Try again.")
		}
	}

	function loadMoreWhenVisible(node: HTMLElement, _nextStartIndex: number) {
		const observer = new IntersectionObserver(
			(entries) => {
				if (
					entries.some((entry) => entry.isIntersecting) &&
					!stream.loadMoreFailed
				) {
					void stream.loadMore()
				}
			},
			{
				root: node.closest("[data-roblox-notifications-scroll]"),
				rootMargin: "0px 0px 160px 0px",
			},
		)
		observer.observe(node)
		return {
			update() {
				observer.unobserve(node)
				observer.observe(node)
			},
			destroy: () => observer.disconnect(),
		}
	}

	onMount(() => {
		void stream.refresh()
		return () => stream.dispose()
	})
</script>

<section class="settings-page" aria-labelledby="roblox-notifications-title">
	<AccountPageHeader
		titleId="roblox-notifications-title"
		title="Notifications"
		username={account.username}
		{onBack}>
		<Select
			className="w-36"
			label="Notification type"
			value={stream.category}
			options={categories}
			onChange={(category) => (stream.category = category)} />
		<button
			class="button-ghost"
			type="button"
			aria-label={`Dismiss all ${categoryNoun}`}
			aria-haspopup="dialog"
			data-tooltip={`Dismiss all ${categoryNoun}`}
			disabled={!stream.hasDismissible || stream.loading || stream.dismissingAll}
			onclick={() => (isDismissAllOpen = true)}>
			<Icon svg={broomIcon} size={16} aria-hidden="true" />
		</button>
		<button
			class="button-ghost"
			type="button"
			aria-label="Refresh notifications"
			data-tooltip="Refresh notifications"
			data-tooltip-side="bottom-end"
			aria-busy={stream.loading}
			disabled={stream.loading || stream.dismissingAll}
			onclick={() => void stream.refresh()}>
			<RefreshIcon isRefreshing={stream.loading} />
		</button>
	</AccountPageHeader>

	<div
		class="min-h-0 flex-1 overflow-y-auto"
		data-roblox-notifications-scroll
		aria-busy={stream.loading}>
		{#if stream.loading && stream.notifications.length === 0}
			<div class="empty-state h-full" role="status">
				<Icon
					svg={circleNotchIcon}
					class="spinner"
					size={28}
					aria-hidden="true" />
				<p>Loading notifications…</p>
			</div>
		{:else if stream.notifications.length === 0 && stream.error}
			<div class="empty-state h-full">
				<Icon svg={bellIcon} size={28} aria-hidden="true" />
				<h2>Notifications unavailable</h2>
				<p role="alert">{stream.error}</p>
				<button
					class="button-primary"
					type="button"
					onclick={() => void stream.refresh()}>Try again</button>
			</div>
		{:else if stream.notifications.length === 0}
			<div class="empty-state h-full">
				<Icon svg={bellIcon} size={28} aria-hidden="true" />
				<h2>No notifications</h2>
				<p>
					Invites, friend updates, and community posts from Roblox appear
					here.
				</p>
			</div>
		{:else}
			<div class="flex max-w-2xl flex-col gap-2 p-3">
				{#if stream.error}
					<p class="notice-error" role="alert">{stream.error}</p>
				{/if}
				{#if stream.visibleNotifications.length === 0 && !stream.nextStartIndex}
					<div class="empty-state py-10">
						<Icon svg={funnelIcon} size={28} aria-hidden="true" />
						<h2>No {categoryNoun}</h2>
						<p>Choose All types to see every notification.</p>
						<button
							class="button-primary"
							type="button"
							onclick={() =>
								(stream.category = RobloxNotificationCategory.$zero)}
							>Show all types</button>
					</div>
				{:else}
					<ul class="flex flex-col gap-2" aria-label="Notifications">
						{#each stream.visibleNotifications as notification (notification.id)}
							<RobloxNotificationCard
								{notification}
								isDismissing={stream.dismissingIds.includes(
									notification.id,
								)}
								isDismissAllRunning={stream.dismissingAll}
								onDismiss={() => void stream.dismiss(notification)}
								{onOpenChat}
								onOpenLink={(url) => void openLink(url)}
								{onFollowUser}
								{onSearchPlace} />
						{/each}
					</ul>
				{/if}
				{#if stream.nextStartIndex}
					<div
						class="flex min-h-7 items-center justify-center text-fg-faint"
						use:loadMoreWhenVisible={stream.nextStartIndex}>
						{#if stream.loadMoreFailed}
							<button
								type="button"
								class="button"
								onclick={() => void stream.loadMore()}>
								Load more notifications
							</button>
						{:else}
							<Icon
								svg={circleNotchIcon}
								class="spinner"
								size={16}
								aria-label="Loading more notifications" />
						{/if}
					</div>
				{/if}
			</div>
		{/if}
	</div>
</section>

{#if isDismissAllOpen}
	<DismissAllRobloxNotificationsDialog
		notifications={stream}
		noun={categoryNoun}
		username={account.username}
		onClose={() => (isDismissAllOpen = false)} />
{/if}
