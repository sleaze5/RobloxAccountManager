<script lang="ts">
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import type { Account } from "../accounts/account-model"
	import {
		bulkRemovalCopy,
		isBulkRemovalList,
		personName,
	} from "../social/social-store.svelte"
	import type { SocialStore } from "../social/social-store.svelte"

	let {
		store,
		account,
		onClose,
	}: { store: SocialStore; account: Account; onClose: () => void } = $props()

	const copy = $derived(
			isBulkRemovalList(store.list) ? bulkRemovalCopy[store.list] : null,
		),
		people = $derived(
			(store.page?.people ?? []).filter((person) =>
				store.selectedUserIds.includes(person.userId),
			),
		),
		count = $derived(store.selectedUserIds.length),
		previewLimit = 5

	function close(): void {
		if (store.removingSelected) return
		store.removeSelectedError = ""
		onClose()
	}

	async function remove(): Promise<void> {
		if (await store.removeSelected()) onClose()
	}
</script>

{#if copy}
	<Dialog
		class="max-w-sm text-left"
		role="alertdialog"
		aria-labelledby="remove-connections-title"
		aria-describedby="remove-connections-description">
		<div class="dialog-heading mb-2">
			<div>
				<h2 id="remove-connections-title">
					{copy.action}
					{count}
					{count === 1 ? "person" : "people"}?
				</h2>
				<p id="remove-connections-description">
					@{account.username} will {copy.action.toLowerCase()} the people below.
					{copy.consequence}
				</p>
			</div>
			<button
				type="button"
				aria-label="Close dialog"
				disabled={store.removingSelected}
				onclick={close}>
				<Icon svg={xIcon} size={14} aria-hidden="true" />
			</button>
		</div>
		<ul
			class="mt-1 flex flex-col overflow-hidden rounded-md border border-line bg-fill">
			{#each people.slice(0, previewLimit) as person (person.userId)}
				<li
					class="flex h-9 items-center gap-2 px-2 text-xs not-first:border-t not-first:border-line-subtle">
					<span
						class="relative grid size-6 shrink-0 place-items-center overflow-hidden rounded-md border border-line-strong bg-raised"
						aria-hidden="true"><ProfileImage url={person.imageUrl} /></span>
					<span class="min-w-0 truncate font-medium text-fg"
						>{personName(person)}</span>
					{#if person.username}<span class="shrink-0 truncate text-fg-faint"
							>@{person.username}</span
						>{/if}
				</li>
			{/each}
			{#if count > previewLimit}
				<li
					class="flex h-8 items-center border-t border-line-subtle px-2 text-xs text-fg-muted">
					and {count - previewLimit} more
				</li>
			{/if}
		</ul>
		{#if store.removeSelectedError}
			<div class="mt-3 notice-error" role="alert">
				{store.removeSelectedError}
			</div>
		{/if}
		<div class="dialog-actions">
			<button
				class="button"
				type="button"
				disabled={store.removingSelected}
				onclick={close}>Cancel</button>
			<button
				class="button-danger"
				type="button"
				aria-busy={store.removingSelected}
				disabled={store.removingSelected || count === 0}
				onclick={() => void remove()}>
				{store.removingSelected ? copy.progress : `${copy.action} ${count}`}
			</button>
		</div>
	</Dialog>
{/if}
