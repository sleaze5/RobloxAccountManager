<script lang="ts">
	import type { FriendConnection } from "../backend/bridge"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import RelationshipBadges from "./RelationshipBadges.svelte"
	import { relationshipBadges } from "./relationship"

	let {
		connections,
		accounts,
	}: { connections: FriendConnection[]; accounts: AccountStore } = $props()

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
				<li
					class="flex min-w-0 animate-enter flex-col gap-2 rounded-lg border border-line p-2">
					<div class="flex min-w-0 items-center gap-2">
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
					</div>
					<div
						class="flex flex-wrap items-center gap-1"
						aria-label={`How @${connection.username} is connected`}>
						<RelationshipBadges badges={relationshipBadges(connection)} />
					</div>
				</li>
			{/each}
		</ul>
	{/if}
</section>
