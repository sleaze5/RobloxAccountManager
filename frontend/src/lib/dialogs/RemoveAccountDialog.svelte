<script lang="ts">
	import StarIcon from "phosphor-svelte/lib/StarIcon"
	import XIcon from "phosphor-svelte/lib/XIcon"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import type { Account } from "../accounts/account-model"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
	} from "../shared/presence"

	let {
		store,
		account,
		onClose,
	}: { store: AccountStore; account: Account; onClose: () => void } = $props()

	function close(): void {
		if (store.busy) {
			return
		}

		store.clearError()
		onClose()
	}

	async function remove(): Promise<void> {
		if (await store.removeAccount(account.id)) {
			onClose()
		}
	}
</script>

<div class="modal-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<div
		class="modal-card confirm-card"
		role="alertdialog"
		aria-modal="true"
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<div class="modal-heading compact-heading">
			<div>
				<h2 class="account-display-name">
					<span>Remove {account.displayName}</span>
					{#if account.favorite}<StarIcon
							class="favorite-name-star"
							size={14}
							weight="fill"
							aria-hidden="true" />{/if}
					<span>?</span>
				</h2>
				<p>This removes the account and its encrypted session.</p>
			</div>
			<button
				type="button"
				aria-label="Close remove account dialog"
				disabled={store.busy}
				onclick={close}>
				<XIcon size={16} aria-hidden="true" />
			</button>
		</div>
		{#if store.error}
			<div class="modal-error" role="alert">
				{store.error}
			</div>
		{/if}
		<div class="modal-actions">
			<button type="button" disabled={store.busy} onclick={close}>Cancel</button>
			<button
				class="danger-button"
				type="button"
				disabled={store.busy}
				onclick={() => void remove()}>
				{store.busy ? "Removing..." : "Remove"}
			</button>
		</div>
	</div>
</div>
