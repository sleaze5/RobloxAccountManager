<script lang="ts">
	import X from "@lucide/svelte/icons/x"
	import { untrack } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { formatCompactBytes } from "../shared/bytes"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
	} from "../shared/presence"

	let {
		store,
		onClose,
	}: {
		store: AccountStore
		onClose: () => void
	} = $props()

	let draft = $state(untrack(() => store.launchInput.launchData))
	let textarea = $state<HTMLTextAreaElement>()

	const byteSize = $derived(new TextEncoder().encode(draft).length)

	function handleClear(): void {
		draft = ""
		textarea?.focus()
	}

	function handleSave(): void {
		store.launchInput.launchData = draft
		onClose()
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (event.key === "Escape") {
			event.preventDefault()
			onClose()
		} else if (event.key === "Enter" && (event.ctrlKey || event.metaKey)) {
			event.preventDefault()
			handleSave()
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<div
	class="modal-backdrop"
	in:modalBackdropIn
	out:modalBackdropOut
	onclick={(event) => {
		if (event.target === event.currentTarget) onClose()
	}}
	role="presentation">
	<form
		novalidate
		class="modal-card launch-data-card"
		use:modalFocus
		in:modalCardIn
		out:modalCardOut
		aria-labelledby="launch-data-title"
		onsubmit={(event) => {
			event.preventDefault()
			handleSave()
		}}>
		<div class="modal-heading compact-heading">
			<div>
				<h2 id="launch-data-title">Launch data</h2>
				<p>
					Custom launch data overrides data in the link. Leave empty to use
					the link's data.
				</p>
			</div>
			<button
				type="button"
				aria-label="Close launch data dialog"
				onclick={onClose}>
				<X size={15} />
			</button>
		</div>
		<label class="modal-field launch-data-field">
			<span>Launch data</span>
			<textarea
				bind:this={textarea}
				bind:value={draft}
				class="launch-data-textarea"
				placeholder="Enter launch data..."
				wrap="soft"
				spellcheck="false"
				autocomplete="off"
				disabled={store.launching}></textarea>
		</label>
		<div class="launch-data-footer">
			<span class="launch-data-size" aria-live="polite">
				{formatCompactBytes(byteSize)}
			</span>
			<div class="modal-actions">
				<button
					type="button"
					disabled={draft.length === 0 || store.launching}
					onclick={handleClear}>
					Clear
				</button>
				<button class="primary-action" type="submit" disabled={store.launching}>
					Save
				</button>
			</div>
		</div>
	</form>
</div>
