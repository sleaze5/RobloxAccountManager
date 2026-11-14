<script lang="ts">
	import type { AccountStore } from "../accounts/account-store.svelte"
	import Select from "../shared/Select.svelte"
	import Skeleton from "../shared/Skeleton.svelte"

	let { store }: { store: AccountStore } = $props()

	const account = $derived(store.selectedAccount),
		accountOptions = $derived(
			store.accounts.map((savedAccount) => ({
				value: savedAccount.id,
				label: savedAccount.displayName,
				description: `@${savedAccount.username}`,
				image: savedAccount.avatarUrl,
			})),
		),
		isFiltered = $derived(store.query !== "" || store.selectedTagIds.length > 0)
</script>

{#if store.vault.unlocked}
	{#if accountOptions.length > 0}
		<Select
			className="-ml-1.5 min-w-0 max-w-56"
			variant="status-strip"
			label={`Account: ${account.displayName} (@${account.username})`}
			value={account.id}
			options={accountOptions}
			onChange={(accountId) => store.selectAccount(accountId)} />
	{:else if store.loadingAccounts}
		<Skeleton label="Loading accounts" />
	{:else}
		<span class="truncate"
			>{isFiltered ? "No matching accounts" : "No accounts"}</span>
	{/if}
{/if}
