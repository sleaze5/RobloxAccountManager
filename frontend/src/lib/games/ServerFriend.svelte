<script lang="ts">
	import type { FriendInServer } from "../backend/bridge"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import { getProfilePreviews } from "../shared/profile-preview"

	// ServerFriend is a friend of the shown account who plays a game, as a
	// button that opens their preview.
	let { friend }: { friend: FriendInServer } = $props()

	const previews = getProfilePreviews()
</script>

<button
	class="flex h-8 max-w-56 min-w-0 shrink-0 items-center gap-2 rounded-md border border-line py-1 pr-2.5 pl-1 text-left transition-colors duration-100 hover:border-line-strong hover:bg-hover"
	type="button"
	aria-label={`${friend.displayName} (@${friend.username}). Show profile`}
	data-tooltip="Show profile"
	onclick={() =>
		previews.show({
			kind: "User",
			userId: friend.userId,
			username: friend.username,
			displayName: friend.displayName,
		})}>
	<span
		class="grid size-6 shrink-0 place-items-center overflow-hidden rounded-full bg-raised text-fg-faint"
		><ProfileImage url={friend.imageUrl} /></span>
	<span class="flex min-w-0 flex-col leading-tight">
		<span class="truncate text-xs font-medium text-fg">{friend.displayName}</span>
		<span class="truncate text-2xs text-fg-muted">@{friend.username}</span>
	</span>
</button>
