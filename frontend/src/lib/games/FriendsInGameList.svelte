<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import eyeSlashIcon from "@phosphor-icons/core/regular/eye-slash.svg?raw"
	import usersIcon from "@phosphor-icons/core/regular/users.svg?raw"
	import type { FriendInServer, FriendPlace, GameServer } from "../backend/bridge"
	import Icon from "../shared/Icon.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import ServerFriend from "./ServerFriend.svelte"
	import ServerTile from "./ServerTile.svelte"
	import type {
		FriendsInGameLoader,
		ServerDetailsLoader,
	} from "./game-servers-state.svelte"
	import { friendServerKey } from "./server-row-menu"
	import type { ServerMenuOpener } from "./server-row-menu"

	// FriendsInGameList shows the servers of the shown account's friends as
	// server tiles, by place. A server that the Roblox list also loaded shows
	// its players, ping, and FPS.
	let {
		friendsInGame,
		details,
		placeId,
		listedServerById,
		newestVersion,
		activeKey,
		menuID,
		onMenu,
	}: {
		friendsInGame: FriendsInGameLoader
		details: ServerDetailsLoader
		placeId: number
		listedServerById: Map<string, GameServer>
		newestVersion: number
		activeKey: string | undefined
		menuID: string
		onMenu: ServerMenuOpener
	} = $props()

	const headingID = $props.id(),
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

{#snippet hiddenFriends(hidden: FriendInServer[], reason: string)}
	<div
		class="flex flex-col gap-2 rounded-lg border border-line-subtle bg-surface p-3">
		<p class="flex items-center gap-1.5 text-2xs text-fg-muted">
			<Icon svg={eyeSlashIcon} size={12} aria-hidden="true" />
			{reason}
		</p>
		<ul
			class="flex min-w-0 flex-wrap gap-1.5"
			aria-label={friendCountLabel(hidden.length)}>
			{#each hidden as friend (friend.userId)}
				<li class="min-w-0"><ServerFriend {friend} /></li>
			{/each}
		</ul>
	</div>
{/snippet}

{#snippet placeHeading(id: string, name: string, detail: string, count: number)}
	<h3 {id} class="flex min-w-0 items-center gap-2 text-2xs text-fg-muted">
		<span class="min-w-0 truncate font-semibold text-fg">{name}</span>
		{#if detail}<span class="shrink-0">{detail}</span>{/if}
		<span class="ml-auto shrink-0 font-mono text-fg-faint"
			>{friendCountLabel(count)}</span>
	</h3>
{/snippet}

<section
	class="flex flex-col gap-2 border-b border-line-subtle p-3"
	aria-labelledby={headingID}
	aria-busy={friendsInGame.loading}>
	<h2
		id={headingID}
		class="flex items-center gap-1.5 text-xs font-semibold text-fg-muted">
		<Icon svg={usersIcon} size={14} aria-hidden="true" />
		Friends in this game
		{#if friendCount > 0}
			<span class="font-mono text-2xs font-normal text-fg-faint"
				>({friendCountLabel(friendCount)})</span>
		{/if}
	</h2>
	{#if !friends}
		{#if !friendsInGame.error}
			<p class="text-xs">
				<Skeleton className="w-48" label="Loading friends in this game…" />
			</p>
		{/if}
	{:else if friendCount === 0}
		<p class="text-xs text-fg-muted" role="status">
			No friends are playing this game.
		</p>
	{:else}
		{#each places as friendPlace (friendPlace.placeId)}
			{@const groupID = `${headingID}-place-${friendPlace.placeId}`}
			{@const hidden = friendPlace.friendsInHiddenServers ?? []}
			<section class="flex flex-col gap-2" aria-labelledby={groupID}>
				{@render placeHeading(
					groupID,
					friendPlace.name,
					friendPlace.placeId === placeId
						? "This place"
						: friendPlace.isRootPlace
							? "Root place"
							: "Subplace",
					placeFriendCount(friendPlace),
				)}
				{#if friendPlace.servers?.length}
					<ol
						class="flex flex-col gap-2"
						aria-label={`Servers in ${friendPlace.name}`}>
						{#each friendPlace.servers as server (server.serverId)}
							{@const listed =
								friendPlace.placeId === placeId
									? listedServerById.get(server.serverId)
									: undefined}
							<ServerTile
								serverId={server.serverId}
								menuKey={friendServerKey(
									friendPlace.placeId,
									server.serverId,
								)}
								friends={server.friends ?? []}
								server={listed}
								record={listed?.record ?? null}
								details={details.get(server.serverId)}
								{newestVersion}
								isMenuOpen={activeKey ===
									friendServerKey(
										friendPlace.placeId,
										server.serverId,
									)}
								{menuID}
								{onMenu}
								onVisible={() =>
									details.request(
										friendPlace.placeId,
										server.serverId,
									)} />
						{/each}
					</ol>
				{/if}
				{#if hidden.length > 0}
					{@render hiddenFriends(
						hidden,
						hidden.length === 1
							? "Their privacy settings hide their server"
							: "Their privacy settings hide their servers",
					)}
				{/if}
			</section>
		{/each}
		{#if friendsInHiddenPlaces.length > 0}
			{@const groupID = `${headingID}-hidden-places`}
			<section class="flex flex-col gap-2" aria-labelledby={groupID}>
				{@render placeHeading(
					groupID,
					"Place not shared by Roblox",
					"",
					friendsInHiddenPlaces.length,
				)}
				{@render hiddenFriends(
					friendsInHiddenPlaces,
					friendsInHiddenPlaces.length === 1
						? "Their privacy settings hide their place and server"
						: "Their privacy settings hide their places and servers",
				)}
			</section>
		{/if}
	{/if}
</section>
