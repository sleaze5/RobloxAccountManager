<script lang="ts">
	import tShirtIcon from "@phosphor-icons/core/regular/t-shirt.svg?raw"
	import type { Account } from "../accounts/account-model"
	import type { SavedAccountsItemOwnership } from "../backend/bridge"
	import type { LabeledBadge } from "../shared/labeled-badge"
	import ProfileCard from "../shared/ProfileCard.svelte"

	let {
		otherAccounts,
		ownership,
		canBeWorn,
	}: {
		// The saved accounts other than the one that the dialog acts for.
		otherAccounts: Account[]
		// undefined while it loads, and null when it failed.
		ownership: SavedAccountsItemOwnership | null | undefined
		canBeWorn: boolean
	} = $props()
	const elementIdPrefix = $props.id(),
		wornBadge: LabeledBadge = {
			label: "Worn",
			icon: tShirtIcon,
			tooltip: "This account wears this item now",
			isStrong: true,
		}

	const owners = $derived(
			(ownership?.owners ?? []).flatMap((owner) => {
				const account = otherAccounts.find(
					(otherAccount) => otherAccount.id === owner.accountId,
				)
				return account ? [{ account, isWorn: owner.isWorn }] : []
			}),
		),
		notOwningCount = $derived(
			ownership
				? ownership.checkedAccountCount - (ownership.owners?.length ?? 0)
				: 0,
		)
</script>

{#if ownership && owners.length > 0}
	<section
		class="flex flex-col gap-3"
		aria-labelledby={`${elementIdPrefix}-inventory-item-saved-accounts`}>
		<h3
			id={`${elementIdPrefix}-inventory-item-saved-accounts`}
			class="flex items-center gap-2 text-xs font-semibold text-fg-muted">
			Saved accounts
			<span class="font-mono text-2xs font-normal text-fg-faint"
				>{owners.length}</span>
		</h3>
		<ul
			class="grid animate-enter grid-cols-3 gap-2"
			aria-label="Saved accounts that own this item">
			{#each owners as owner (owner.account.id)}
				<li class="min-w-0 animate-enter">
					<ProfileCard
						kind="User"
						imageUrl={owner.account.avatarUrl}
						title={owner.account.displayName}
						subtitle={`@${owner.account.username}`}
						isFavorite={owner.account.favorite}
						badges={canBeWorn && owner.isWorn ? [wornBadge] : []} />
				</li>
			{/each}
		</ul>
		{#if notOwningCount > 0}
			<p class="text-2xs text-fg-muted">
				{notOwningCount} other saved {notOwningCount === 1
					? "account does"
					: "accounts do"} not own this item.
			</p>
		{/if}
		{#if ownership.failedAccountCount > 0}
			<p class="text-2xs text-fg-muted">
				{ownership.failedAccountCount}
				{ownership.failedAccountCount === 1 ? "account" : "accounts"} not checked.
			</p>
		{/if}
	</section>
{/if}
