<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import eyeSlashIcon from "@phosphor-icons/core/regular/eye-slash.svg?raw"
	import usersIcon from "@phosphor-icons/core/regular/users.svg?raw"
	import type { FriendInServer, FriendPlace } from "../backend/bridge"
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
		places = $derived(friends?.places ?? []),
		friendsInHiddenPlaces = $derived(friends?.friendsInHiddenPlaces ?? []),
		friendCount = $derived(
			places.reduce(
				(total, friendPlace) => total + placeFriendCount(friendPlace),
				0,
			) + friendsInHiddenPlaces.length,
		)

	function placeFriendCount(friendPlace: FriendPlace): number {
		return (friendPlace.servers ?? []).reduce(
			(total, server) => total + (server.friends?.length ?? 0),
			friendPlace.friendsInHiddenServers?.length ?? 0,
		)
	}

	function friendCountLabel(count: number): string {
		return `${count} ${count === 1 ? "friend" : "friends"}`
	}

	notifyErrors("Friends not loaded", () => friendsInGame.error)
</script>

{#snippet avatars(shownFriends: FriendInServer[])}
	<div
		class="flex w-24 shrink-0 items-center"
		aria-label={friendCountLabel(shownFriends.length)}>
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

{#snippet hiddenRow(hiddenFriends: FriendInServer[], reason: string)}
	<li class="flex h-11 items-center gap-3 border-t border-line-subtle bg-canvas px-3">
		{@render avatars(hiddenFriends)}
		<div class="flex min-w-0 flex-1 flex-col">
			{@render names(hiddenFriends)}
			<span class="flex items-center gap-1 truncate text-2xs text-fg-faint">
				<Icon svg={eyeSlashIcon} size={12} aria-hidden="true" />
				{reason}
			</span>
		</div>
	</li>
{/snippet}

{#snippet groupHeading(
	id: string,
	name: string,
	kind: string,
	count: number,
	detail: string,
)}
	<h3
		{id}
		class="flex h-7 items-center gap-2 border-t border-line-subtle bg-raised px-3 text-2xs text-fg-muted">
		<span class="min-w-0 truncate font-semibold text-fg">{name}</span>
		{#if kind}<span class="shrink-0">{kind}</span>{/if}
		<span class="ml-auto shrink-0 font-mono text-fg-faint"
			>{detail ? `${detail} · ` : ""}{friendCountLabel(count)}</span>
	</h3>
{/snippet}

<section
	class="flex max-h-64 shrink-0 flex-col border-b border-line-subtle"
	aria-labelledby={headingID}
	aria-busy={friendsInGame.loading}>
	<h2
		id={headingID}
		class="flex items-center gap-1.5 px-3 pt-2 pb-1.5 text-xs font-semibold text-fg-muted">
		<Icon svg={usersIcon} size={14} aria-hidden="true" />
		Friends in this game
		{#if friendCount > 0}
			<span class="font-mono text-2xs font-normal text-fg-faint"
				>({friendCountLabel(friendCount)})</span>
		{/if}
	</h2>
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
		<div class="min-h-0 scrollbar-stable overflow-y-auto">
			{#each places as friendPlace (friendPlace.placeId)}
				{@const groupID = `${headingID}-place-${friendPlace.placeId}`}
				{@const hiddenFriends = friendPlace.friendsInHiddenServers ?? []}
				<section aria-labelledby={groupID}>
					{@render groupHeading(
						groupID,
						friendPlace.name,
						friendPlace.isRootPlace ? "Root place" : "Subplace",
						placeFriendCount(friendPlace),
						friendPlace.placeId === placeId
							? "This place"
							: `Place ID: ${friendPlace.placeId}`,
					)}
					<ol aria-label={`Servers in ${friendPlace.name}`}>
						{#each friendPlace.servers ?? [] as server (server.jobId)}
							{@const key = friendServerKey(
								friendPlace.placeId,
								server.jobId,
							)}
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
										>{server.jobId}</span>
								</div>
								<ServerActionsButton
									open={activeKey === key}
									{menuID}
									onMenu={(trigger, toggle) =>
										onMenu(key, trigger, toggle)} />
							</li>
						{/each}
						{#if hiddenFriends.length > 0}
							{@render hiddenRow(
								hiddenFriends,
								hiddenFriends.length === 1
									? "Their privacy settings hide their server"
									: "Their privacy settings hide their servers",
							)}
						{/if}
					</ol>
				</section>
			{/each}
			{#if friendsInHiddenPlaces.length > 0}
				{@const groupID = `${headingID}-hidden-places`}
				<section aria-labelledby={groupID}>
					{@render groupHeading(
						groupID,
						"Place not shared by Roblox",
						"",
						friendsInHiddenPlaces.length,
						"",
					)}
					<ol aria-label="Friends in places that Roblox does not share">
						{@render hiddenRow(
							friendsInHiddenPlaces,
							friendsInHiddenPlaces.length === 1
								? "Their privacy settings hide their place and server"
								: "Their privacy settings hide their places and servers",
						)}
					</ol>
				</section>
			{/if}
		</div>
	{/if}
</section>
