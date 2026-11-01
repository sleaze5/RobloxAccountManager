<script lang="ts">
	import handshakeIcon from "@phosphor-icons/core/regular/handshake.svg?raw"
	import paperPlaneTiltIcon from "@phosphor-icons/core/regular/paper-plane-tilt.svg?raw"
	import userPlusIcon from "@phosphor-icons/core/regular/user-plus.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { FriendStatus } from "../backend/bridge"
	import type { FriendConnection } from "../backend/bridge"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"

	let {
		connections,
		accounts,
	}: { connections: FriendConnection[]; accounts: AccountStore } = $props()

	const statuses: Partial<Record<FriendStatus, { label: string; icon: string }>> = {
		[FriendStatus.FriendStatusFriends]: { label: "Friends", icon: handshakeIcon },
		[FriendStatus.FriendStatusRequestSent]: {
			label: "Request sent",
			icon: paperPlaneTiltIcon,
		},
		[FriendStatus.FriendStatusRequestReceived]: {
			label: "Request received",
			icon: userPlusIcon,
		},
	}

	function avatar(accountId: number): string {
		return (
			accounts.accounts.find((account) => account.id === accountId)?.avatarUrl ??
			""
		)
	}
</script>

<section class="flex flex-col gap-2" aria-labelledby="social-connections-heading">
	<h3 id="social-connections-heading" class="text-xs font-semibold text-fg-muted">
		Saved account connections
	</h3>
	{#if connections.length === 0}
		<p class="text-xs text-fg-faint">
			None of your other saved accounts are connected to them.
		</p>
	{:else}
		<ul
			class="grid max-w-2xl grid-cols-1 gap-2 @md:grid-cols-2 @3xl:grid-cols-3"
			aria-label="Saved account connections">
			{#each connections as connection (connection.accountId)}
				{@const status = statuses[connection.status]}
				<li
					class="flex min-w-0 animate-enter items-center gap-2 rounded-lg border border-line px-2 py-2">
					<span
						class="relative grid size-8 shrink-0 place-items-center overflow-hidden rounded-md border border-line-strong bg-raised"
						aria-hidden="true">
						<ProfileImage url={avatar(connection.accountId)} />
					</span>
					<span class="flex min-w-0 flex-1 flex-col leading-snug">
						<strong class="truncate text-xs font-semibold text-fg"
							>{connection.displayName}</strong>
						<small class="truncate text-2xs text-fg-muted"
							>@{connection.username}</small>
					</span>
					{#if status}
						<span
							class="inline-flex h-5 shrink-0 items-center gap-1 rounded-sm border border-line-strong bg-raised px-1 text-2xs font-medium text-fg-muted">
							<Icon
								svg={status.icon}
								size={12}
								aria-hidden="true" />{status.label}
						</span>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
</section>
