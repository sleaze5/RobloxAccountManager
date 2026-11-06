<script lang="ts">
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import trashIcon from "@phosphor-icons/core/regular/trash.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { tick } from "svelte"
	import Icon from "../shared/Icon.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { TagNameInputState } from "../accounts/tag-name-input.svelte"
	import {
		isValidTagName,
		maxTagNameLength,
		tagNameError,
	} from "../accounts/tag-name"
	import { replayValidationNudge } from "../shared/presence"
	import Dialog from "../shared/Dialog.svelte"

	let {
			store,
			accountId,
			onClose,
		}: {
			store: AccountStore
			accountId: number | null
			onClose: () => void
		} = $props(),
		form = $state<HTMLFormElement>(),
		creating = $state(false),
		confirmingTagId = $state<number | null>(null),
		deleting = $state(false)
	const nameInput = new TagNameInputState(() => store.clearError()),
		invalid = $derived(
			nameInput.value.length > 0 && !isValidTagName(nameInput.value),
		),
		validationMessage = $derived(invalid ? tagNameError(nameInput.value) : "")

	function close(): void {
		if (store.busy) {
			return
		}

		store.clearError()
		onClose()
	}

	async function confirmDelete(tagId: number): Promise<void> {
		store.clearError()
		confirmingTagId = tagId
		await tick()
		document.querySelector<HTMLButtonElement>("[data-cancel-tag-delete]")?.focus()
	}

	async function cancelDelete(): Promise<void> {
		const tagId = confirmingTagId
		confirmingTagId = null
		await tick()
		document
			.querySelector<HTMLButtonElement>(`[data-delete-tag="${tagId}"]`)
			?.focus()
	}

	function cancelOnEscape(event: KeyboardEvent): void {
		if (event.key !== "Escape") return
		event.stopPropagation()
		if (!store.busy) void cancelDelete()
	}

	async function deleteTag(tagId: number): Promise<void> {
		deleting = true
		try {
			await store.removeTag(tagId)
		} finally {
			deleting = false
		}
		if (store.error) return
		confirmingTagId = null
		await tick()
		document.querySelector<HTMLInputElement>("#managed-tag-name")?.focus()
	}

	async function submit(): Promise<void> {
		if (store.busy) return
		nameInput.resetSeparator()
		if (!isValidTagName(nameInput.value)) {
			replayValidationNudge(form)
			return
		}

		creating = true
		try {
			if (await store.createTag(nameInput.value, accountId)) {
				nameInput.value = ""
			} else {
				replayValidationNudge(form)
			}
		} finally {
			creating = false
		}
	}
</script>

<Dialog class="max-w-sm" aria-labelledby="manage-tags-title">
	<div class="dialog-heading mb-2">
		<div>
			<h2 id="manage-tags-title">Manage tags</h2>
			<p>
				{accountId === null
					? "Create and delete reusable account tags."
					: "New tags are also added to this account."}
			</p>
		</div>
		<button
			type="button"
			aria-label="Close manage tags dialog"
			disabled={store.busy}
			onclick={close}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>
	{#if store.error}
		<div class="mt-3 notice-error" role="alert">{store.error}</div>
	{/if}
	<form
		novalidate
		class="tag-manager-create"
		bind:this={form}
		onsubmit={(event) => {
			event.preventDefault()
			void submit()
		}}>
		<label class="dialog-field" for="managed-tag-name">
			<span>New tag</span>
		</label>
		<div class="flex gap-2">
			<input
				class="field text-sm"
				id="managed-tag-name"
				value={nameInput.value}
				oninput={(event) => nameInput.update(event)}
				onkeydown={(event) => nameInput.keydown(event)}
				onpointerdown={() => nameInput.resetSeparator()}
				onblur={() => nameInput.finish()}
				type="text"
				autocomplete="off"
				placeholder="alt-account"
				maxlength={maxTagNameLength}
				disabled={store.busy}
				aria-invalid={invalid}
				aria-describedby={invalid ? "managed-tag-validation" : undefined} />
			<button
				class="btn-primary"
				type="submit"
				disabled={store.busy || !isValidTagName(nameInput.value)}>
				{creating ? "Creating…" : "Create tag"}
			</button>
		</div>
	</form>
	{#if invalid}
		<p
			class="mt-1 text-xs leading-snug text-danger-muted"
			id="managed-tag-validation">
			{validationMessage}
		</p>
	{:else}
		<p class="mt-1">Use letters and numbers. Spaces become hyphens.</p>
	{/if}
	<div class="mt-4 flex items-center justify-between pb-2 text-xs text-fg-muted">
		<strong class="font-semibold">Tags</strong>
		<span class="font-mono text-2xs text-fg-faint">{store.customTags.length}</span>
	</div>
	<div
		class="max-h-48 overflow-y-auto rounded-md border border-line bg-fill *:not-first:border-t *:not-first:border-line-subtle">
		{#each store.customTags as tag (tag.id)}
			{#if confirmingTagId === tag.id}
				<div
					class="flex animate-enter flex-col gap-2 bg-danger/5 p-2 text-xs text-fg"
					role="group"
					aria-labelledby="delete-tag-prompt">
					<p id="delete-tag-prompt">
						Delete the <strong class="font-semibold">{tag.name}</strong> tag?
						It is removed from every account and cannot be undone.
					</p>
					<div class="flex justify-end gap-2">
						<button
							class="btn"
							type="button"
							data-cancel-tag-delete
							disabled={store.busy}
							onkeydown={cancelOnEscape}
							onclick={() => void cancelDelete()}>Cancel</button>
						<button
							class="btn-danger"
							type="button"
							disabled={store.busy}
							onkeydown={cancelOnEscape}
							onclick={() => void deleteTag(tag.id)}
							>{deleting ? "Deleting…" : "Delete tag"}</button>
					</div>
				</div>
			{:else}
				<div class="flex min-h-8 items-center gap-2 pl-2 text-xs text-fg">
					<Icon
						svg={hashIcon}
						class="shrink-0 text-fg-faint"
						size={16}
						aria-hidden="true" />
					<span class="min-w-0 flex-1 truncate">{tag.name}</span>
					<button
						class="grid size-8 shrink-0 place-items-center text-fg-faint transition-colors duration-100 hover:bg-danger/10 hover:text-danger"
						type="button"
						aria-label={`Delete ${tag.name} tag`}
						data-tooltip={`Delete ${tag.name}`}
						data-tooltip-side="left"
						data-delete-tag={tag.id}
						disabled={store.busy}
						onclick={() => void confirmDelete(tag.id)}>
						<Icon svg={trashIcon} size={16} aria-hidden="true" />
					</button>
				</div>
			{/if}
		{:else}
			<div class="px-2 py-3 text-center text-xs text-fg-faint">
				No custom tags
			</div>
		{/each}
	</div>
	<div class="dialog-actions">
		<button class="btn" type="button" disabled={store.busy} onclick={close}
			>Done</button>
	</div>
</Dialog>
