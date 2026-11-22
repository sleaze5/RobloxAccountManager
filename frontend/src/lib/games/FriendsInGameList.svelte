<script lang="ts">
	import eyeSlashIcon from "@phosphor-icons/core/regular/eye-slash.svg?raw"
	import usersIcon from "@phosphor-icons/core/regular/users.svg?raw"
	import type { FriendInServer } from "../backend/bridge"
	import Icon from "../shared/Icon.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import ServerActionsButton from "./ServerActionsButton.svelte"
	import type { FriendsInGameLoader } from "./game-servers-state.svelte"
	import { friendServerKey, openServerMenuFromRow } from "./server-row-menu"
	import type { ServerMenuOpener } from "./server-row-menu"

	let {
		friendsInGame,
		placeId,
		activeKey,
		menuID,
		onMenu,
	}: {
		friendsInGame: FriendsInGameLoader
		placeId: number
		activeKey: string | undefined
		menuID: string
		onMenu: ServerMenuOpener
	} = $props()

	// Four avatars and a count fit the fixed avatar column, so names line up.
	const maxShownAvatars = 4,
		headingID = $props.id(),
		friends = $derived(friendsInGame.friends),
		servers = $derived(friends?.servers ?? []),
		friendsInHiddenServers = $derived(friends?.friendsInHiddenServers ?? []),
		friendCount = $derived(
			servers.reduce(
				(total, server) => total + (server.friends?.length ?? 0),
				0,
			) + friendsInHiddenServers.length,
		)
</script>

{#snippet avatars(shownFriends: FriendInServer[])}
	<div
		class="flex w-24 shrink-0 items-center"
		aria-label={`${shownFriends.length} ${shownFriends.length === 1 ? "friend" : "friends"}`}>
		{#each shownFriends.slice(0, maxShownAvatars) as friend (friend.userId)}
			<span
				class="grid size-6 shrink-0 place-items-center overflow-hidden rounded-full border border-canvas bg-raised text-fg-faint not-first:-ml-1.5"
				><ProfileImage url={friend.imageUrl} /></span>
		{/each}
		{#if shownFriends.length > maxShownAvatars}
			<span class="ml-1 font-mono text-2xs text-fg-muted"
				>+{shownFriends.length - maxShownAvatars}</span>
		{/if}
	</div>
{/snippet}

{#snippet names(shownFriends: FriendInServer[])}
	<span
		class="truncate text-xs font-medium text-fg"
		data-tooltip={shownFriends
			.map((friend) => `${friend.displayName} (@${friend.username})`)
			.join(", ")}
		>{shownFriends.map((friend) => friend.displayName).join(", ")}</span>
{/snippet}

<section
	class="flex max-h-48 shrink-0 flex-col border-b border-line-subtle"
	aria-labelledby={headingID}
	aria-busy={friendsInGame.loading}>
	<h2
		id={headingID}
		class="flex items-center gap-1.5 px-3 pt-2 pb-1.5 text-xs font-semibold text-fg-muted">
		<Icon svg={usersIcon} size={14} aria-hidden="true" />
		Friends in this game
		{#if friendCount > 0}
			<span class="font-mono text-2xs font-normal text-fg-faint"
				>({friendCount} {friendCount === 1 ? "friend" : "friends"})</span>
		{/if}
	</h2>
	{#if friendsInGame.error}
		<div class="mx-3 mb-2 flex items-center gap-3 notice-error" role="alert">
			<span class="flex-1">{friendsInGame.error}</span>
			<button
				class="button"
				type="button"
				onclick={() => void friendsInGame.refresh()}>Try again</button>
		</div>
	{/if}
	{#if !friends}
		{#if !friendsInGame.error}
			<p class="px-3 pb-2 text-xs">
				<Skeleton className="w-48" label="Loading friends in this game…" />
			</p>
		{/if}
	{:else if friendCount === 0}
		<p class="px-3 pb-2 text-xs text-fg-muted" role="status">
			No friends are playing this game.
		</p>
	{:else}
		<ol
			class="min-h-0 scrollbar-stable overflow-y-auto"
			aria-label="Friends in this game by server">
			{#each servers as server (friendServerKey(server))}
				{@const key = friendServerKey(server)}
				{@const serverFriends = server.friends ?? []}
				<li
					class="flex h-11 items-center gap-3 border-t border-line-subtle bg-canvas px-3 transition-colors duration-75 hover:bg-hover data-open:bg-hover"
					data-open={activeKey === key || undefined}
					oncontextmenu={(event) =>
						openServerMenuFromRow(event, key, onMenu)}>
					{@render avatars(serverFriends)}
					<div class="flex min-w-0 flex-1 flex-col">
						{@render names(serverFriends)}
						<span
							class="truncate font-mono text-2xs text-fg-faint select-text"
							>{server.placeId === placeId
								? server.jobId
								: `Place ID: ${server.placeId} · ${server.jobId}`}</span>
					</div>
					<ServerActionsButton
						open={activeKey === key}
						{menuID}
						onMenu={(trigger, toggle) => onMenu(key, trigger, toggle)} />
				</li>
			{/each}
			{#if friendsInHiddenServers.length > 0}
				<li
					class="flex h-11 items-center gap-3 border-t border-line-subtle bg-canvas px-3">
					{@render avatars(friendsInHiddenServers)}
					<div class="flex min-w-0 flex-1 flex-col">
						{@render names(friendsInHiddenServers)}
						<span
							class="flex items-center gap-1 truncate text-2xs text-fg-faint">
							<Icon svg={eyeSlashIcon} size={12} aria-hidden="true" />
							{friendsInHiddenServers.length === 1
								? "Their privacy settings hide their server"
								: "Their privacy settings hide their servers"}
						</span>
					</div>
				</li>
			{/if}
		</ol>
	{/if}
</section>
