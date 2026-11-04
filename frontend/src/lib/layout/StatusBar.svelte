<script lang="ts">
	import type { AccountStore } from "../accounts/account-store.svelte"

	let { store }: { store: AccountStore } = $props()

	const selectedPosition = $derived(
		store.accounts.findIndex((account) => account.id === store.selectedAccount.id) +
			1,
	)
</script>

<footer
	class="flex h-6 shrink-0 items-center justify-between border-t border-line-subtle bg-surface pl-3 text-2xs text-fg-faint">
	<span aria-live="polite">
		{#if !store.multipleSelected && selectedPosition > 0}
			Account #{selectedPosition}
		{/if}
	</span>
</footer>
