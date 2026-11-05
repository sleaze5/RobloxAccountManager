<script lang="ts">
	import checkBoldIcon from "@phosphor-icons/core/bold/check-bold.svg?raw"
	import dotsSixVerticalIcon from "@phosphor-icons/core/regular/dots-six-vertical.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { tick } from "svelte"
	import { isMoveShortcut, moveListFocus } from "../shared/move-shortcut"
	import type { WorkspaceState } from "../layout/workspace-state.svelte"
	import AccountAvatar from "./AccountAvatar.svelte"
	import type { AccountStore } from "./account-store.svelte"
	import { presenceDetail, presenceLabel } from "./account-model"
	import PresenceLine from "./PresenceLine.svelte"

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
	class={["min-h-0 scrollbar-stable overflow-y-auto"]}
	aria-busy={store.loadingAccounts || store.loadingMore}
	onscroll={loadNearEnd}>
	{#each store.filteredAccounts as account (account.id)}
		{@const detail = presenceDetail(account.presence)}
		<div
			class={[
				"group/row list-row relative flex w-full items-center gap-2 border-b border-line-subtle text-left",
				detail ? "h-16" : "h-12",
				draggedAccountId === account.id && "opacity-45",
				dropTarget?.accountId === account.id &&
					(dropTarget.after ? "shadow-drop-after" : "shadow-drop-before"),
			]}
			data-selected={store.selectedAccountIds.includes(account.id) || undefined}
			data-account-row
			role="group"
			aria-label={`@${account.username}`}
			oncontextmenu={(event) => openContextMenu(event, account.id)}
			ondragover={(event) => updateDropTarget(event, account.id)}
			ondrop={(event) => dropAccount(event, account.id)}>
			{#if store.multipleSelected}
				<span class={["checkbox ml-2", detail && "mt-4 self-start"]}>
					<input
						type="checkbox"
						aria-label={`Select @${account.username}`}
						checked={store.selectedAccountIds.includes(account.id)}
						onkeydown={(event) => handleContextKey(event, account.id)}
						onchange={(event) => {
							selectAccount(account.id, true)
							if (!store.multipleSelected) {
								event.currentTarget
									.closest("[data-account-row]")
									?.querySelector<HTMLButtonElement>("button")
									?.focus()
							}
						}} />
					<span aria-hidden="true"
						><Icon
							svg={checkBoldIcon}
							size={11}
							aria-hidden="true" /></span>
				</span>
			{:else}
				<span
					class={[
						"grid h-7 w-5 shrink-0 cursor-grab place-items-center pl-2 text-fg-faint active:cursor-grabbing",
						detail && "mt-2.5 self-start",
					]}
					draggable={!store.busy && store.accounts.length > 1}
					aria-hidden="true"
					ondragstart={(event) => startDrag(event, account.id)}
					ondragend={finishDrag}>
					<Icon svg={dotsSixVerticalIcon} size={14} aria-hidden="true" />
				</span>
			{/if}
			<button
				class="flex h-full min-w-0 flex-1 flex-col justify-center gap-1 pr-2 text-left"
				data-account-select
				data-list-item
				type="button"
				aria-pressed={store.selectedAccountIds.includes(account.id)}
				aria-keyshortcuts="Control+Shift+ArrowUp Control+Shift+ArrowDown"
				aria-label={`${account.displayName} (@${account.username}), ${presenceLabel(account.presence)}${account.favorite ? ", favorite" : ""}`}
				onclick={(event) =>
					selectAccount(account.id, event.ctrlKey || event.metaKey)}
				onkeydown={(event) => {
					handleContextKey(event, account.id)
					void moveWithKeyboard(event, account.id)
					moveListFocus(event)
					if (
						(event.key === " " || event.key === "Enter") &&
						(event.ctrlKey || event.metaKey)
					) {
						event.preventDefault()
						selectAccount(account.id, true)
					}
				}}
				ondragstart={(event) => event.preventDefault()}>
				<span class="flex w-full min-w-0 items-center gap-2">
					<AccountAvatar
						{account}
						className="size-9 text-xs"
						showPresence={true} />
					<span class="block min-w-0">
						<strong
							class="mb-0.5 flex min-w-0 items-center gap-1 text-sm font-semibold text-fg">
							<span class="truncate">{account.displayName}</span>
							{#if account.favorite}<Icon
									svg={starFillIcon}
									class="shrink-0 animate-pop text-accent-strong"
									size={12}
									aria-hidden="true" />{/if}
						</strong>
						<small
							class="block truncate text-2xs text-fg-faint group-hover/row:text-fg-muted group-data-selected/row:text-fg-muted"
							>@{account.username}</small>
					</span>
				</span>
				{#if detail}<PresenceLine presence={account.presence} />{/if}
			</button>
		</div>
	{:else}
		<div
			class="flex h-24 flex-col items-center justify-center gap-0.5 text-xs text-fg-faint">
			{#if store.loadingAccounts}
				<span>Loading accounts…</span>
			{:else if !store.vault.unlocked}
				<span>Unlock the vault to see accounts.</span>
			{:else if store.query || store.selectedTagIds.length > 0}
				<strong class="font-medium text-fg-muted">No matching accounts</strong>
			{/if}
		</div>
	{/each}
	{#if store.loadingMore}
		<div class="flex h-24 items-center justify-center text-xs text-fg-faint">
			<span>Loading more accounts…</span>
		</div>
	{/if}
	<span class="sr-only" aria-live="polite">{moveAnnouncement}</span>
</div>
