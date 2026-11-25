<script lang="ts">
	import type { FriendConnection } from "../backend/bridge"
	import type { Account } from "../accounts/account-model"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import ProfileCard from "../shared/ProfileCard.svelte"
	import { relationshipBadges } from "./relationship"

	let {
		connections,
		accounts,
	}: { connections: FriendConnection[]; accounts: AccountStore } = $props()

	function findSavedAccount(accountId: number): Account | undefined {
		return accounts.accounts.find((account) => account.id === accountId)
	}
</script>

{#if connections.length > 0}
	<section class="flex flex-col gap-2" aria-labelledby="social-connections-heading">
		<h3 id="social-connections-heading" class="text-xs font-semibold text-fg-muted">
			Saved accounts
		</h3>
		<ul
			class="grid max-w-2xl grid-cols-1 gap-2 @md:grid-cols-2 @3xl:grid-cols-3"
			aria-label="Saved accounts">
			{#each connections as connection (connection.accountId)}
				{@const account = findSavedAccount(connection.accountId)}
				<li class="min-w-0 animate-enter">
					<ProfileCard
						kind="User"
						imageUrl={account?.avatarUrl ?? ""}
						title={connection.displayName}
						subtitle={`@${connection.username}`}
						isFavorite={account?.favorite ?? false}
						badges={relationshipBadges(connection)} />
				</li>
			{/each}
		</ul>
	</section>
{/if}
