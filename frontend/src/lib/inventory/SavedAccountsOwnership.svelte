<script lang="ts">
	import tShirtIcon from "@phosphor-icons/core/regular/t-shirt.svg?raw"
	import AccountAvatar from "../accounts/AccountAvatar.svelte"
	import type { Account } from "../accounts/account-model"
	import type { SavedAccountsItemOwnership } from "../backend/bridge"
	import Icon from "../shared/Icon.svelte"

	let {
		otherAccounts,
		ownership,
		canBeWorn,
		onRetry,
	}: {
		// The saved accounts other than the one that the dialog acts for.
		otherAccounts: Account[]
		// undefined while it loads, and null when it failed.
		ownership: SavedAccountsItemOwnership | null | undefined
		canBeWorn: boolean
		onRetry: () => void
	} = $props()
	const elementIdPrefix = $props.id()

	const owners = $derived(
			(ownership?.owners ?? []).flatMap((owner) => {
				const account = otherAccounts.find(
					(otherAccount) => otherAccount.id === owner.accountId,
				)
				return account ? [{ account, isWorn: owner.isWorn }] : []
			}),
		),
		skeletonCardIndexes = $derived(
			Array.from(
				{ length: Math.min(otherAccounts.length, 3) },
				(_unused, index) => index,
			),
		),
		notOwningCount = $derived(
			ownership
				? ownership.checkedAccountCount - (ownership.owners?.length ?? 0)
				: 0,
		),
		checkedLabel = $derived(
			ownership
				? `${ownership.checkedAccountCount} other saved ${ownership.checkedAccountCount === 1 ? "account" : "accounts"}`
				: "",
		)
</script>

<section
	class="flex flex-col gap-3"
	aria-labelledby={`${elementIdPrefix}-inventory-item-saved-accounts`}>
	<h3
		id={`${elementIdPrefix}-inventory-item-saved-accounts`}
		class="flex items-center gap-2 text-xs font-semibold text-fg-muted">
		Saved accounts
		{#if owners.length > 0}
			<span class="font-mono text-2xs font-normal text-fg-faint"
				>{owners.length}</span>
		{/if}
	</h3>
	{#if ownership === undefined}
		<ul class="grid grid-cols-3 gap-2" aria-busy="true">
			{#each skeletonCardIndexes as index (index)}
				<li class="flex items-center gap-2 rounded-lg border border-line p-2">
					<span class="size-8 shrink-0 skeleton rounded-md"></span>
					<span class="flex min-w-0 flex-1 flex-col">
						<span class="text-xs"
							><span class="skeleton-line w-24"></span></span>
						<span class="text-2xs"
							><span class="skeleton-line w-16"></span></span>
					</span>
				</li>
			{/each}
		</ul>
		<span class="sr-only" role="status">Checking your other accounts…</span>
	{:else if ownership === null}
		<div class="flex items-center gap-2 notice-error" role="alert">
			<span class="min-w-0 flex-1">Could not check your other accounts.</span>
			<button class="button" type="button" onclick={onRetry}>Try again</button>
		</div>
	{:else}
		{#if owners.length === 0}
			<p class="text-xs text-fg-muted">
				{ownership.checkedAccountCount === 0
					? "No other saved account could be checked."
					: `None of your ${checkedLabel} own this item.`}
			</p>
		{:else}
			<ul
				class="grid animate-enter grid-cols-3 gap-2"
				aria-label="Saved accounts that own this item">
				{#each owners as owner (owner.account.id)}
					<li
						class="flex min-w-0 animate-enter items-center gap-2 rounded-lg border border-line p-2">
						<AccountAvatar account={owner.account} className="size-8" />
						<span class="flex min-w-0 flex-1 flex-col leading-snug">
							<strong class="truncate text-xs font-semibold text-fg"
								>{owner.account.displayName}</strong>
							<small class="truncate text-2xs text-fg-muted"
								>@{owner.account.username}</small>
						</span>
						{#if canBeWorn && owner.isWorn}
							<span
								class="grid size-5 shrink-0 animate-pop place-items-center rounded-sm border border-line-strong bg-overlay text-success"
								data-tooltip="Wears this item now"
								><Icon
									svg={tShirtIcon}
									size={12}
									aria-hidden="true" /><span class="sr-only"
									>Wears this item now</span
								></span>
						{/if}
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
		{/if}
		{#if ownership.failedAccountCount > 0}
			<p class="text-2xs text-fg-muted" role="status">
				Could not check {ownership.failedAccountCount}
				{ownership.failedAccountCount === 1 ? "account" : "accounts"}. Open the
				item again to try again.
			</p>
		{/if}
	{/if}
</section>
