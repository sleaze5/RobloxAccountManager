<script lang="ts">
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { accountPageLabels } from "../accounts/account-pages"
	import StatusTrail from "./StatusTrail.svelte"
	import { getWorkspace } from "./workspace-state.svelte"

	let { store }: { store: AccountStore } = $props()

	const workspace = getWorkspace(),
		selectedPosition = $derived(
			store.accounts.findIndex(
				(account) => account.id === store.selectedAccount.id,
			) + 1,
		)
</script>

<footer
	class="flex h-6 shrink-0 items-center gap-1 border-t border-line-subtle bg-surface pl-3 text-2xs text-fg-faint">
	{#if !store.multipleSelected && selectedPosition > 0}
		<span class="shrink-0" aria-live="polite">Account #{selectedPosition}</span>
		<StatusTrail
			parts={workspace.accountPage === "profile"
				? []
				: [accountPageLabels[workspace.accountPage]]} />
	{/if}
</footer>
