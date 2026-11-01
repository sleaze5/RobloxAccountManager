<script lang="ts">
	import cursorClickIcon from "@phosphor-icons/core/regular/cursor-click.svg?raw"
	import dotsThreeIcon from "@phosphor-icons/core/regular/dots-three.svg?raw"
	import plusIcon from "@phosphor-icons/core/regular/plus.svg?raw"
	import userPlusIcon from "@phosphor-icons/core/regular/user-plus.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { WorkspaceState } from "../layout/workspace-state.svelte"
	import type { AccountStore } from "./account-store.svelte"
	import AddAccountMenu from "./AddAccountMenu.svelte"

	let {
		store,
		workspace,
		onAddAccount,
		onBrowser,
	}: {
		store: AccountStore
		workspace: WorkspaceState
		onAddAccount: () => void
		onBrowser: () => void
	} = $props()
	const menuOpen = $derived(workspace.activeAccountMenu === "empty-add-account")
	let trigger = $state<HTMLButtonElement>()

	function pasteCookie(): void {
		workspace.closeAccountMenus()
		onAddAccount()
	}

	function openBrowser(): void {
		workspace.closeAccountMenus()
		onBrowser()
	}
	const noAccounts = $derived(
		store.accounts.length === 0 &&
			!store.query &&
			store.selectedTagIds.length === 0,
	)
</script>

{#if store.multipleSelected}
	<div class="empty-state min-h-full animate-enter">
		<Icon svg={usersThreeIcon} size={28} aria-hidden="true" />
		<h2>{store.selectedAccountIds.length} accounts selected</h2>
		<p>Launch starts every selected account.</p>
		<div class="flex gap-2">
			<button
				class="btn"
				type="button"
				aria-expanded={workspace.accountContextMenu !== null}
				onclick={(event) =>
					workspace.openAccountContextMenu(
						event,
						store.selectedAccountIds[0],
					)}>
				<Icon svg={dotsThreeIcon} size={16} aria-hidden="true" />Edit tags or
				copy
			</button>
			<button class="btn" type="button" onclick={() => store.clearSelection()}>
				Clear selection
			</button>
		</div>
	</div>
{:else if store.vault.unlocked && !store.loadingAccounts}
	<div class="empty-state min-h-full animate-enter">
		{#if noAccounts}
			<Icon svg={userPlusIcon} size={28} aria-hidden="true" />
			<h2>No accounts yet</h2>
			<p>Sign in with a browser or paste a cookie.</p>
			<div class="relative" data-account-menu>
				<button
					bind:this={trigger}
					class="btn-primary"
					type="button"
					aria-expanded={menuOpen}
					disabled={store.busy}
					onclick={() => workspace.toggleAccountMenu("empty-add-account")}>
					<Icon svg={plusIcon} size={16} aria-hidden="true" />Add account
				</button>
				{#if menuOpen}
					<AddAccountMenu
						disabled={store.busy}
						anchor={trigger}
						onPasteCookie={pasteCookie}
						onBrowser={openBrowser} />
				{/if}
			</div>
		{:else}
			<Icon svg={cursorClickIcon} size={28} aria-hidden="true" />
			<h2>No account selected</h2>
			<p>Select an account in the list to see its profile.</p>
		{/if}
	</div>
{/if}
