<script lang="ts">
	import CheckIcon from "phosphor-svelte/lib/CheckIcon"
	import DotsSixVerticalIcon from "phosphor-svelte/lib/DotsSixVerticalIcon"
	import StarIcon from "phosphor-svelte/lib/StarIcon"
	import { tick } from "svelte"
	import { isMoveShortcut } from "../shared/move-shortcut"
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
		dropTarget = $state<DropTarget | null>(null),
		moveAnnouncement = $state("")

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

	async function moveWithKeyboard(
		event: KeyboardEvent,
		accountId: number,
	): Promise<void> {
		if (!isMoveShortcut(event)) return
		event.preventDefault()
		if (store.busy || store.multipleSelected) return
		const button = event.currentTarget as HTMLElement,
			down = event.key === "ArrowDown",
			index = store.accounts.findIndex((account) => account.id === accountId),
			target = store.accounts[index + (down ? 1 : -1)]
		if (!target) return
		const moving = store.moveAccount(accountId, target.id, down)
		await tick()
		button.focus()
		await moving
		const position =
			store.accounts.findIndex((account) => account.id === accountId) + 1
		if (position > 0) {
			moveAnnouncement = `Moved to position ${position} of ${store.accountTotal}`
		}
	}

	function loadNearEnd(event: Event): void {
		const list = event.currentTarget as HTMLElement
		if (list.scrollHeight - list.scrollTop - list.clientHeight < 120) {
			void store.loadMore()
		}
	}
</script>

<div
	class="account-list"
	aria-busy={store.loadingAccounts || store.loadingMore}
	onscroll={loadNearEnd}>
	{#each store.filteredAccounts as account (account.id)}
		<div
			class:selected={store.selectedAccountIds.includes(account.id)}
			class:dragging={draggedAccountId === account.id}
			class:drop-before={dropTarget?.accountId === account.id &&
				!dropTarget.after}
			class:drop-after={dropTarget?.accountId === account.id && dropTarget.after}
			class="account-row list-row"
			role="group"
			aria-label={`@${account.username}`}
			oncontextmenu={(event) => openContextMenu(event, account.id)}
			ondragover={(event) => updateDropTarget(event, account.id)}
			ondrop={(event) => dropAccount(event, account.id)}>
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
						><CheckIcon size={11} weight="bold" aria-hidden="true" /></span>
				</span>
			{:else}
				<span
					class="account-drag-handle"
					draggable={!store.busy && store.accounts.length > 1}
					aria-hidden="true"
					ondragstart={(event) => startDrag(event, account.id)}
					ondragend={finishDrag}>
					<DotsSixVerticalIcon size={14} aria-hidden="true" />
				</span>
			{/if}
			<button
				class="account-row-select"
				type="button"
				aria-pressed={store.selectedAccountIds.includes(account.id)}
				aria-keyshortcuts="Control+Shift+ArrowUp Control+Shift+ArrowDown"
				aria-label={`${account.displayName} (@${account.username}), ${presenceLabel(account.presence)}${account.favorite ? ", favorite" : ""}`}
				onclick={(event) =>
					selectAccount(account.id, event.ctrlKey || event.metaKey)}
				onkeydown={(event) => {
					handleContextKey(event, account.id)
					void moveWithKeyboard(event, account.id)
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
						{#if account.favorite}<StarIcon
								class="favorite-name-star"
								size={12}
								weight="fill"
								aria-hidden="true" />{/if}
					</strong>
					<small>@{account.username}</small>
				</span>
			</button>
		</div>
	{:else}
		<div class="empty-list">
			{#if store.loadingAccounts}
				<span>Loading accounts…</span>
			{:else if !store.query && store.selectedTagIds.length === 0}
				<strong>No accounts yet</strong>
				<span>Use the plus button to add one.</span>
			{:else}
				<strong>No matching accounts</strong>
				<span>Change the search or tag filter.</span>
			{/if}
		</div>
	{/each}
	{#if store.loadingMore}
		<div class="empty-list"><span>Loading more accounts…</span></div>
	{/if}
	<span class="visually-hidden" aria-live="polite">{moveAnnouncement}</span>
</div>
