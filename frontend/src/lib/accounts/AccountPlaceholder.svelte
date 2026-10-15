<script lang="ts">
	import CursorClickIcon from "phosphor-svelte/lib/CursorClickIcon"
	import DotsThreeIcon from "phosphor-svelte/lib/DotsThreeIcon"
	import PlusIcon from "phosphor-svelte/lib/PlusIcon"
	import UserPlusIcon from "phosphor-svelte/lib/UserPlusIcon"
	import UsersThreeIcon from "phosphor-svelte/lib/UsersThreeIcon"
	import type { WorkspaceState } from "../layout/workspace-state.svelte"
	import type { AccountStore } from "./account-store.svelte"

	let {
		store,
		workspace,
		onAddAccount,
	}: {
		store: AccountStore
		workspace: WorkspaceState
		onAddAccount: () => void
	} = $props()
	const noAccounts = $derived(
		store.accounts.length === 0 &&
			!store.query &&
			store.selectedTagIds.length === 0,
	)
</script>

{#if store.multipleSelected}
	<div class="account-content empty-state">
		<UsersThreeIcon size={27} aria-hidden="true" />
		<h2>{store.selectedAccountIds.length} accounts selected</h2>
		<p>Launch starts every selected account.</p>
		<div class="account-placeholder-actions">
			<button
				class="control-button text-button"
				type="button"
				aria-expanded={workspace.accountContextMenu !== null}
				onclick={(event) =>
					workspace.openAccountContextMenu(
						event,
						store.selectedAccountIds[0],
					)}>
				<DotsThreeIcon size={16} aria-hidden="true" />Edit tags or copy
			</button>
			<button
				class="control-button text-button"
				type="button"
				onclick={() => store.clearSelection()}>
				Clear selection
			</button>
		</div>
	</div>
{:else if store.vault.unlocked && !store.loadingAccounts}
	<div class="account-content empty-state">
		{#if noAccounts}
			<UserPlusIcon size={27} aria-hidden="true" />
			<h2>No accounts yet</h2>
			<p>Add an account with its .ROBLOSECURITY cookie to get started.</p>
			<button class="primary-action" type="button" onclick={onAddAccount}>
				<PlusIcon size={14} aria-hidden="true" />Add account
			</button>
		{:else}
			<CursorClickIcon size={27} aria-hidden="true" />
			<h2>No account selected</h2>
			<p>Select an account in the list to see its profile.</p>
		{/if}
	</div>
{/if}

<style>
	.account-placeholder-actions {
		display: flex;
		gap: var(--space-2);
	}

	.primary-action {
		display: flex;
		align-items: center;
		gap: var(--space-1);
	}
</style>
