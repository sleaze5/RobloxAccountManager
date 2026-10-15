<script lang="ts">
	import CircleNotchIcon from "phosphor-svelte/lib/CircleNotchIcon"
	import type { BlockedUser } from "../backend/bridge"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import type { BlockedUsersState } from "./blocked-users-state.svelte"

	let { blocked }: { blocked: BlockedUsersState } = $props()

	function name(user: BlockedUser): string {
		return user.displayName || user.username || `User ${user.userId}`
	}
</script>

{#if blocked.error}
	<div class="settings-inline-error" role="alert">{blocked.error}</div>
{/if}

{#if !blocked.users}
	{#if blocked.loading}
		<div class="account-settings-loading" role="status">
			<CircleNotchIcon class="spinner" size={16} aria-hidden="true" />
			Loading blocked users
		</div>
	{/if}
{:else if blocked.users.length === 0}
	<p class="blocked-users-empty">No blocked users.</p>
{:else}
	<ul class="settings-rows blocked-users" aria-label="Blocked users">
		{#each blocked.users as user (user.userId)}
			<li class="settings-row blocked-user">
				<span class="blocked-user-avatar avatar-image" aria-hidden="true">
					<ProfileImage url={user.imageUrl} />
				</span>
				<div class="settings-row-copy">
					<strong>{name(user)}</strong>
					<span>{user.username ? `@${user.username}` : user.userId}</span>
				</div>
				<button
					class="control-button text-button"
					type="button"
					aria-label={`Unblock ${name(user)}`}
					disabled={blocked.loading || blocked.unblocking !== null}
					onclick={() => void blocked.unblock(user)}>
					{#if blocked.unblocking === user.userId}
						<CircleNotchIcon class="spinner" size={14} aria-hidden="true" />
					{/if}
					Unblock
				</button>
			</li>
		{/each}
	</ul>
{/if}
