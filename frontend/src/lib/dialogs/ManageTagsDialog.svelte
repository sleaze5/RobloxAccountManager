<script lang="ts">
	import Hash from "@lucide/svelte/icons/hash"
	import Trash2 from "@lucide/svelte/icons/trash-2"
	import X from "@lucide/svelte/icons/x"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { TagNameInputState } from "../accounts/tag-name-input.svelte"
	import {
		isValidTagName,
		maxTagNameLength,
		tagNameError,
	} from "../accounts/tag-name"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
		replayValidationNudge,
	} from "../shared/presence"

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
		creating = $state(false)
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

<div class="modal-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<div
		class="modal-card tag-manager-card"
		role="dialog"
		aria-modal="true"
		aria-labelledby="manage-tags-title"
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<div class="modal-heading compact-heading">
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
				<X size={15} />
			</button>
		</div>
		{#if store.error}
			<div class="modal-error" role="alert">{store.error}</div>
		{/if}
		<form
			novalidate
			class="tag-manager-create"
			bind:this={form}
			onsubmit={(event) => {
				event.preventDefault()
				void submit()
			}}>
			<label class="modal-field tag-name-field" for="managed-tag-name">
				<span>New tag</span>
			</label>
			<div class="tag-manager-create-row">
				<input
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
					class="primary-action"
					type="submit"
					disabled={store.busy || !isValidTagName(nameInput.value)}>
					{creating ? "Creating..." : "Create tag"}
				</button>
			</div>
		</form>
		{#if invalid}
			<p class="tag-name-validation" id="managed-tag-validation">
				{validationMessage}
			</p>
		{:else}
			<p class="tag-name-hint">Use letters and numbers. Spaces become hyphens.</p>
		{/if}
		<div class="tag-manager-heading">
			<strong>Tags</strong>
			<span>{store.customTags.length}</span>
		</div>
		<div class="tag-manager-list">
			{#each store.customTags as tag (tag.id)}
				<div class="tag-manager-row">
					<Hash size={14} aria-hidden="true" />
					<span>{tag.name}</span>
					<button
						type="button"
						aria-label={`Delete ${tag.name} tag`}
						data-tooltip={`Delete ${tag.name}`}
						data-tooltip-side="left"
						disabled={store.busy}
						onclick={() => void store.removeTag(tag.id)}>
						<Trash2 size={14} />
					</button>
				</div>
			{:else}
				<div class="tag-manager-empty">No custom tags</div>
			{/each}
		</div>
		<div class="modal-actions">
			<button type="button" disabled={store.busy} onclick={close}>Done</button>
		</div>
	</div>
</div>
