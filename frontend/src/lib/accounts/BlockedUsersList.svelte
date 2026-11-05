<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { BlockedUser } from "../backend/bridge"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import type { BlockedUsersState } from "./blocked-users-state.svelte"

	let { blocked }: { blocked: BlockedUsersState } = $props()

	function name(user: BlockedUser): string {
		return user.displayName || user.username || `User ${user.userId}`
	}
</script>

{#if blocked.error}
	<div class="notice-error" role="alert">{blocked.error}</div>
{/if}

{#if !blocked.users}
	{#if blocked.loading}
		<div class="flex items-center gap-2 text-xs text-fg-muted" role="status">
			<Icon svg={circleNotchIcon} class="spinner" size={16} aria-hidden="true" />
			Loading blocked users
		</div>
	{/if}
{:else if blocked.users.length === 0}
	<p class="text-xs text-fg-faint">No blocked users.</p>
{:else}
	<ul class="settings-section settings-rows" aria-label="Blocked users">
		{#each blocked.users as user (user.userId)}
			<li class="settings-row justify-start gap-3">
				<span
					class="relative grid size-7 shrink-0 place-items-center overflow-hidden rounded-md border border-line-strong bg-raised"
					aria-hidden="true">
					<ProfileImage url={user.imageUrl} />
				</span>
				<div class="settings-row-copy">
					<strong>{name(user)}</strong>
					<span>{user.username ? `@${user.username}` : user.userId}</span>
				</div>
				<button
					class="btn"
					type="button"
					aria-label={`Unblock ${name(user)}`}
					disabled={blocked.loading || blocked.unblocking !== null}
					onclick={() => void blocked.unblock(user)}>
					{#if blocked.unblocking === user.userId}
						<Icon
							svg={circleNotchIcon}
							class="spinner"
							size={14}
							aria-hidden="true" />
					{/if}
					Unblock
				</button>
			</li>
		{/each}
	</ul>
{/if}
