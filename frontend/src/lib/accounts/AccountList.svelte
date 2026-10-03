<script lang="ts">
	import Check from "@lucide/svelte/icons/check"
	import GripVertical from "@lucide/svelte/icons/grip-vertical"
	import Star from "@lucide/svelte/icons/star"
	import type { WorkspaceState } from "../layout/workspace-state.svelte"
	import AccountAvatar from "./AccountAvatar.svelte"
	import type { AccountStore } from "./account-store.svelte"
	import { presenceLabel } from "./account-model"

	interface DropTarget {
		accountId: number
		after: boolean
	}

	let {
			store,
			workspace,
		}: {
			store: AccountStore
			workspace: WorkspaceState
		} = $props(),
		draggedAccountId = $state<number | null>(null),
		dropTarget = $state<DropTarget | null>(null)

	function selectAccount(accountId: number, toggle = false): void {
		if (toggle) store.toggleAccount(accountId)
		else store.selectAccount(accountId)
		workspace.closeAccountMenus()
	}

	function openContextMenu(
		event: MouseEvent | KeyboardEvent,
		accountId: number,
	): void {
		if (!store.selectedAccountIds.includes(accountId)) {
			store.selectAccount(accountId)
		}
		workspace.openAccountContextMenu(event, accountId)
	}

	function handleContextKey(event: KeyboardEvent, accountId: number): void {
		if (event.key === "ContextMenu" || (event.shiftKey && event.key === "F10")) {
			openContextMenu(event, accountId)
		}
	}

	function startDrag(event: DragEvent, accountId: number): void {
		if (store.busy || store.multipleSelected || store.accounts.length < 2) {
			event.preventDefault()
			return
		}
		draggedAccountId = accountId
		dropTarget = null
		workspace.closeAccountMenus()
		if (event.dataTransfer) {
			event.dataTransfer.effectAllowed = "move"
			event.dataTransfer.setData("text/plain", String(accountId))
		}
	}

	function updateDropTarget(event: DragEvent, accountId: number): void {
		if (
			store.multipleSelected ||
			draggedAccountId === null ||
			draggedAccountId === accountId
		) {
			return
		}
		if (!(event.currentTarget instanceof HTMLElement)) {
			return
		}

		event.preventDefault()
		if (event.dataTransfer) {
			event.dataTransfer.dropEffect = "move"
		}
		const bounds = event.currentTarget.getBoundingClientRect()
		dropTarget = {
			accountId,
			after: event.clientY >= bounds.top + bounds.height / 2,
		}
	}

	function finishDrag(): void {
		draggedAccountId = null
		dropTarget = null
	}

	function dropAccount(event: DragEvent, accountId: number): void {
		event.preventDefault()
		const sourceID = draggedAccountId,
			after = dropTarget?.accountId === accountId && dropTarget.after
		finishDrag()
		if (store.multipleSelected || sourceID === null || sourceID === accountId) {
			return
		}
		void store.moveAccount(sourceID, accountId, after)
	}

	function loadNearEnd(event: Event): void {
		const list = event.currentTarget as HTMLElement
		if (list.scrollHeight - list.scrollTop - list.clientHeight < 120) {
			void store.loadMore()
		}
	}
</script>

<div class="account-list" onscroll={loadNearEnd}>
	{#each store.filteredAccounts as account (account.id)}
		<div
			class:selected={store.selectedAccountIds.includes(account.id)}
			class:dragging={draggedAccountId === account.id}
			class:drop-before={dropTarget?.accountId === account.id &&
				!dropTarget.after}
			class:drop-after={dropTarget?.accountId === account.id && dropTarget.after}
			class="account-row"
			role="group"
			aria-label={`@${account.username}`}
			oncontextmenu={(event) => openContextMenu(event, account.id)}
			ondragover={(event) => updateDropTarget(event, account.id)}
			ondrop={(event) => dropAccount(event, account.id)}>
			<span class="selection-mark"></span>
			{#if store.multipleSelected}
				<span class="cookie-selection account-checkbox">
					<input
						type="checkbox"
						aria-label={`Select @${account.username}`}
						checked={store.selectedAccountIds.includes(account.id)}
						onkeydown={(event) => handleContextKey(event, account.id)}
						onchange={(event) => {
							selectAccount(account.id, true)
							if (!store.multipleSelected) {
								event.currentTarget
									.closest(".account-row")
									?.querySelector<HTMLButtonElement>("button")
									?.focus()
							}
						}} />
					<span aria-hidden="true"
						><Check size={10} strokeWidth={2.4} /></span>
				</span>
			{:else}
				<span
					class="account-drag-handle"
					draggable={!store.busy && store.accounts.length > 1}
					aria-hidden="true"
					ondragstart={(event) => startDrag(event, account.id)}
					ondragend={finishDrag}>
					<GripVertical size={12} />
				</span>
			{/if}
			<button
				class="account-row-select"
				type="button"
				aria-pressed={store.selectedAccountIds.includes(account.id)}
				aria-label={`${account.displayName} (@${account.username}), ${presenceLabel(account.presence)}${account.favorite ? ", favorite" : ""}`}
				onclick={(event) =>
					selectAccount(account.id, event.ctrlKey || event.metaKey)}
				onkeydown={(event) => {
					handleContextKey(event, account.id)
					if (
						(event.key === " " || event.key === "Enter") &&
						(event.ctrlKey || event.metaKey)
					) {
						event.preventDefault()
						selectAccount(account.id, true)
					}
				}}
				ondragstart={(event) => event.preventDefault()}>
				<AccountAvatar
					{account}
					className="account-avatar"
					showPresence={true} />
				<span class="account-name">
					<strong class="account-display-name">
						<span>{account.displayName}</span>
						{#if account.favorite}<Star
								class="favorite-name-star"
								size={11}
								fill="currentColor"
								aria-hidden="true" />{/if}
					</strong>
					<small>@{account.username}</small>
				</span>
			</button>
		</div>
	{:else}
		<div class="empty-list">
			{#if store.accounts.length === 0}
				<strong>No accounts yet</strong>
				<span>Use the plus button to add one.</span>
			{:else}
				<strong>No matching accounts</strong>
				<span>Change the search or tag filter.</span>
			{/if}
		</div>
	{/each}
	{#if store.loadingMore}
		<div class="empty-list"><span>Loading more accounts...</span></div>
	{/if}
</div>
