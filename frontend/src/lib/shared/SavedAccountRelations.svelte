<script lang="ts" module>
	import type { Account } from "../accounts/account-model"
	import type { LabeledBadge } from "./labeled-badge"

	// SavedAccountRelation is another saved account with badges that name how
	// it relates to the profile, such as Friends, Owned, or Favorite.
	export interface SavedAccountRelation {
		account: Account
		badges: LabeledBadge[]
	}
</script>

<script lang="ts">
	import ProfileCard from "./ProfileCard.svelte"

	// SavedAccountRelations lists the other saved accounts that relate to a
	// user, a game, or an item. It shows nothing when no account relates.
	// failedAccountCount counts the accounts that Roblox did not answer for.
	let {
		relations,
		failedAccountCount = 0,
	}: { relations: SavedAccountRelation[]; failedAccountCount?: number } = $props()
	const headingId = $props.id()
</script>

{#if relations.length > 0}
	<section class="@container flex flex-col gap-2" aria-labelledby={headingId}>
		<h3
			id={headingId}
			class="flex items-center gap-2 text-xs font-semibold text-fg-muted">
			Saved accounts
			<span class="font-mono text-2xs font-normal text-fg-faint"
				>{relations.length}</span>
		</h3>
		<ul
			class="grid grid-cols-1 gap-2 @md:grid-cols-2 @3xl:grid-cols-3"
			aria-label="Saved accounts">
			{#each relations as relation (relation.account.id)}
				<li class="min-w-0 animate-enter">
					<ProfileCard
						kind="User"
						imageUrl={relation.account.avatarUrl}
						title={relation.account.displayName}
						subtitle={`@${relation.account.username}`}
						isFavorite={relation.account.favorite}
						badges={relation.badges} />
				</li>
			{/each}
		</ul>
		{#if failedAccountCount > 0}
			<p class="text-2xs text-fg-muted">
				{failedAccountCount}
				{failedAccountCount === 1 ? "account" : "accounts"} not fully checked
			</p>
		{/if}
	</section>
{/if}
