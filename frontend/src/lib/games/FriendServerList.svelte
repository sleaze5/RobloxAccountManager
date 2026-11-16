<script lang="ts">
	import usersIcon from "@phosphor-icons/core/regular/users.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import ServerActionsButton from "./ServerActionsButton.svelte"
	import type { FriendServers } from "./game-servers-state.svelte"
	import { openServerMenuFromRow } from "./server-row-menu"
	import type { ServerMenuOpener } from "./server-row-menu"

	let {
		friendServers,
		accountUsername,
		activeKey,
		menuID,
		onMenu,
	}: {
		friendServers: FriendServers
		accountUsername: string
		activeKey: string | undefined
		menuID: string
		onMenu: ServerMenuOpener
	} = $props()

	const maxShownAvatars = 5,
		headingID = $props.id(),
		servers = $derived(friendServers.servers),
		friendCount = $derived(
			(servers ?? []).reduce(
				(total, server) => total + (server.friends?.length ?? 0),
				0,
			),
		)
</script>

<section
	class="flex max-h-48 shrink-0 flex-col border-b border-line-subtle"
	aria-labelledby={headingID}
	aria-busy={friendServers.loading}>
	<h2
		id={headingID}
		class="flex items-center gap-1.5 px-3 pt-2 pb-1.5 text-xs font-semibold text-fg-muted">
		<Icon svg={usersIcon} size={14} aria-hidden="true" />
		Friends' servers
		{#if servers?.length}
			<span class="font-mono text-2xs font-normal text-fg-faint"
				>({friendCount} {friendCount === 1 ? "friend" : "friends"})</span>
		{/if}
	</h2>
	{#if friendServers.error && !servers}
		<div class="mx-3 mb-2 flex items-center gap-3 notice-error" role="alert">
			<span class="flex-1">{friendServers.error}</span>
			<button
				class="button"
				type="button"
				onclick={() => void friendServers.refresh()}>Try again</button>
		</div>
	{:else if !servers}
		<p class="px-3 pb-2 text-xs">
			<Skeleton className="w-48" label="Loading friends' servers" />
		</p>
	{:else if servers.length === 0}
		<p class="px-3 pb-2 text-xs text-fg-muted" role="status">
			No friends of @{accountUsername} are in a server you can see.
		</p>
	{:else}
		<ol
			class="min-h-0 scrollbar-stable overflow-y-auto"
			aria-label="Servers with friends">
			{#each servers as server (server.jobId)}
				{@const key = `friend-${server.jobId}`}
				{@const friends = server.friends ?? []}
				{@const names = friends.map((friend) => friend.displayName).join(", ")}
				<li
					class="flex h-11 items-center gap-3 border-t border-line-subtle bg-canvas px-3 transition-colors duration-75 hover:bg-hover data-open:bg-hover"
					data-open={activeKey === key || undefined}
					oncontextmenu={(event) =>
						openServerMenuFromRow(event, key, onMenu)}>
					<div
						class="flex shrink-0 items-center"
						aria-label={`${friends.length} ${friends.length === 1 ? "friend" : "friends"} in this server`}>
						{#each friends.slice(0, maxShownAvatars) as friend (friend.userId)}
							<span
								class="grid size-6 shrink-0 place-items-center overflow-hidden rounded-full border border-canvas bg-raised text-fg-faint not-first:-ml-1.5"
								><ProfileImage url={friend.imageUrl} /></span>
						{/each}
						{#if friends.length > maxShownAvatars}
							<span class="ml-1 font-mono text-2xs text-fg-muted"
								>+{friends.length - maxShownAvatars}</span>
						{/if}
					</div>
					<div class="flex min-w-0 flex-1 flex-col">
						<span
							class="truncate text-xs font-medium text-fg"
							data-tooltip={friends
								.map(
									(friend) =>
										`${friend.displayName} (@${friend.username})`,
								)
								.join(", ")}>{names}</span>
						<span
							class="truncate font-mono text-2xs text-fg-faint select-text"
							>{server.jobId}</span>
					</div>
					<ServerActionsButton
						open={activeKey === key}
						{menuID}
						onMenu={(trigger, toggle) => onMenu(key, trigger, toggle)} />
				</li>
			{/each}
		</ol>
	{/if}
</section>
