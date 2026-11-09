<script lang="ts">
	import arrowSquareOutIcon from "@phosphor-icons/core/regular/arrow-square-out.svg?raw"
	import bellIcon from "@phosphor-icons/core/regular/bell.svg?raw"
	import chatsIcon from "@phosphor-icons/core/regular/chats.svg?raw"
	import gameControllerIcon from "@phosphor-icons/core/regular/game-controller.svg?raw"
	import signInIcon from "@phosphor-icons/core/regular/sign-in.svg?raw"
	import userIcon from "@phosphor-icons/core/regular/user.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import { RobloxNotificationImageKind } from "../backend/bridge"
	import type {
		GamePlace,
		RobloxNotificationUser,
		RobloxNotificationView,
	} from "../backend/bridge"
	import { loadGamePlace } from "../games/game-place-cache"
	import { creatorByline, getGamesStore } from "../games/games-store.svelte"
	import { relativeTime } from "../shared/timestamp"

	let {
		notification,
		isDismissing,
		isDismissAllRunning,
		onDismiss,
		onOpenChat,
		onOpenLink,
		onFollowUser,
		onSearchPlace,
	}: {
		notification: RobloxNotificationView
		isDismissing: boolean
		isDismissAllRunning: boolean
		onDismiss: () => void
		onOpenChat: (userId: number) => void
		onOpenLink: (url: string) => void
		onFollowUser: (user: RobloxNotificationUser) => void
		onSearchPlace: (placeId: number) => void
	} = $props()

	const games = getGamesStore(),
		fallbackIcons: Record<RobloxNotificationImageKind, string> = {
			[RobloxNotificationImageKind.$zero]: bellIcon,
			[RobloxNotificationImageKind.RobloxNotificationImageUser]: userIcon,
			[RobloxNotificationImageKind.RobloxNotificationImageGroup]: usersThreeIcon,
			[RobloxNotificationImageKind.RobloxNotificationImageGame]:
				gameControllerIcon,
			[RobloxNotificationImageKind.RobloxNotificationImageOther]: bellIcon,
		},
		action = $derived(notification.action),
		user = $derived(notification.user),
		userName = $derived(
			user ? user.displayName || user.username || `User ${user.userId}` : "",
		),
		canDismiss = $derived(!!notification.dismissAction)

	let place = $state<GamePlace | null>(null)

	$effect(() => {
		const join = action?.join
		place = null
		if (!join || join.universeId <= 0) return
		let isCurrent = true
		void loadGamePlace(join.universeId, join.placeId).then((loaded) => {
			if (isCurrent) place = loaded?.place ?? null
			return loaded
		})
		return () => {
			isCurrent = false
		}
	})
</script>

<li
	class="flex min-w-0 animate-enter gap-3 rounded-lg border border-line p-3"
	aria-labelledby={`roblox-notification-${notification.id}`}>
	<span
		class="relative grid size-9 shrink-0 place-items-center overflow-hidden rounded-md border border-line-strong bg-raised"
		aria-hidden="true">
		<ProfileImage
			url={notification.imageUrl}
			fallbackIcon={fallbackIcons[notification.imageKind]} />
	</span>
	<div class="flex min-w-0 flex-1 flex-col gap-2">
		<div class="flex min-w-0 items-start gap-2">
			<div
				id={`roblox-notification-${notification.id}`}
				class="flex min-w-0 flex-1 flex-col gap-0.5 text-sm wrap-anywhere text-fg">
				{#if user}
					<p class="flex min-w-0 flex-wrap items-baseline gap-x-1">
						<strong class="font-semibold">{userName}</strong>
						{#if user.nickname}<span class="text-fg-muted"
								>(<span class="sr-only"
									>Nickname:
								</span>{user.nickname})</span
							>{/if}
						{#if user.username}<span class="text-xs text-fg-muted"
								>@{user.username}</span
							>{/if}
					</p>
				{:else if notification.title}
					<strong class="font-semibold">{notification.title}</strong>
				{/if}
				{#if notification.body}<p>{notification.body}</p>{/if}
			</div>
			{#if notification.createdAtMs}
				<time
					class="shrink-0 text-xs text-fg-muted"
					datetime={new Date(notification.createdAtMs).toISOString()}
					>{relativeTime(notification.createdAtMs)}</time>
			{/if}
		</div>
		{#if action?.join && place}
			{@const placeId = action.join.placeId}
			<button
				class="flex max-w-full min-w-0 animate-enter items-center gap-2 self-start rounded-lg border border-line-subtle p-1.5 pr-3 text-left transition-colors duration-100 ease-in-out hover:bg-hover"
				type="button"
				aria-label={`Find ${games.displayName(place)} in Games`}
				data-tooltip="Find this place in Games"
				onclick={() => onSearchPlace(placeId)}>
				<span
					class="relative grid size-10 shrink-0 place-items-center overflow-hidden rounded-md border border-line-strong bg-raised"
					aria-hidden="true">
					<ProfileImage
						url={place.iconUrl}
						fallbackIcon={gameControllerIcon} />
				</span>
				<span class="flex min-w-0 flex-col">
					<strong class="truncate text-sm font-medium text-fg"
						>{games.displayName(place)}</strong>
					{#if place.creatorName}<small class="truncate text-xs text-fg-muted"
							>{creatorByline(place)}</small
						>{/if}
				</span>
			</button>
		{/if}
		{#if canDismiss || action}
			<div class="flex flex-wrap items-center gap-2">
				{#if canDismiss}
					<button
						class="button"
						type="button"
						aria-busy={isDismissing}
						disabled={isDismissing || isDismissAllRunning}
						onclick={onDismiss}>
						{isDismissing ? "Dismissing…" : "Dismiss"}
					</button>
				{/if}
				{#if action?.join && user}
					{@const inviter = user}
					<button
						class="button"
						type="button"
						data-tooltip={`Fill launch options to follow ${userName}`}
						onclick={() => onFollowUser(inviter)}>
						<Icon svg={signInIcon} size={14} aria-hidden="true" />
						{action.label || "Join"}
					</button>
				{:else if action?.chatUserId}
					{@const userId = action.chatUserId}
					<button
						class="button"
						type="button"
						onclick={() => onOpenChat(userId)}>
						<Icon svg={chatsIcon} size={14} aria-hidden="true" />
						{action.label || "Chat"}
					</button>
				{:else if action?.linkUrl}
					{@const url = action.linkUrl}
					<button
						class="button"
						type="button"
						onclick={() => onOpenLink(url)}>
						<Icon svg={arrowSquareOutIcon} size={14} aria-hidden="true" />
						{action.label || "Open"}
					</button>
				{/if}
			</div>
		{/if}
	</div>
</li>
