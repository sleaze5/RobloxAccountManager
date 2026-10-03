<script lang="ts">
	import type { AccountStore } from "../accounts/account-store.svelte"

	let { store }: { store: AccountStore } = $props()
</script>

<footer class="status-bar">
	<span aria-live="polite">
		{#if store.multipleSelected}
			Accounts: {store.selectedAccountIds.length} selected
		{:else}
			Account: {store.selectedAccount.id === 0
				? "None"
				: `@${store.selectedAccount.username}`}
		{/if}
	</span>
	<div>
		<span>{store.accounts.length} accounts</span>
		<span
			><i class="status-dot presence-online"></i>{store.onlineCount} online</span>
		<span>
			<i class="status-dot warning"></i>
			{store.attentionCount} needs attention
		</span>
	</div>
</footer>
