<script lang="ts">
	import type { AccountStore } from "../accounts/account-store.svelte"

	let { store }: { store: AccountStore } = $props()

	const selectedPosition = $derived(
		store.accounts.findIndex((account) => account.id === store.selectedAccount.id) +
			1,
	)
	const accountName = $derived(
		store.selectedAccount.displayName === store.selectedAccount.username
			? store.selectedAccount.username
			: `${store.selectedAccount.displayName} @${store.selectedAccount.username}`,
	)
</script>

<footer class="status-bar">
	<span aria-live="polite">
		{#if store.multipleSelected}
			Accounts: {store.selectedAccountIds.length} selected
		{:else}
			{store.selectedAccount.id === 0
				? "Account: None"
				: `Account #${selectedPosition}: ${accountName}`}
		{/if}
	</span>
</footer>
