<script lang="ts">
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
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
	class="dialog-backdrop"
	in:modalBackdropIn
	out:modalBackdropOut
	onclick={(event) => {
		if (event.target === event.currentTarget) onClose()
	}}
	role="presentation">
	<div
		class="dialog-card max-w-md"
		role="dialog"
		aria-modal="true"
		use:modalFocus
		in:modalCardIn
		out:modalCardOut
		aria-labelledby="launch-data-title">
		<form
			novalidate
			onsubmit={(event) => {
				event.preventDefault()
				handleSave()
			}}>
			<div class="dialog-heading mb-2">
				<div>
					<h2 id="launch-data-title">Launch data</h2>
					<p>
						Custom launch data overrides data in the link. Leave empty to
						use the link's data.
					</p>
				</div>
				<button
					type="button"
					aria-label="Close launch data dialog"
					onclick={onClose}>
					<Icon svg={xIcon} size={14} aria-hidden="true" />
				</button>
			</div>
			<label class="dialog-field">
				<span>Launch data</span>
				<textarea
					bind:this={textarea}
					bind:value={draft}
					class="h-36 break-all whitespace-pre-wrap"
					placeholder="Enter launch data…"
					wrap="soft"
					spellcheck="false"
					autocomplete="off"
					disabled={store.launching}></textarea>
			</label>
			<div class="mt-3 flex items-center justify-between gap-3">
				<span
					class="font-mono text-xs text-fg-faint select-none"
					aria-live="polite">
					{formatCompactBytes(byteSize)}
				</span>
				<div class="dialog-actions mt-0">
					<button
						class="btn"
						type="button"
						disabled={draft.length === 0 || store.launching}
						onclick={handleClear}>
						Clear
					</button>
					<button
						class="btn-primary"
						type="submit"
						disabled={store.launching}>
						Save
					</button>
				</div>
			</div>
		</form>
	</div>
</div>
