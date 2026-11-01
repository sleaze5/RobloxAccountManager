<script lang="ts">
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
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

<div class="dialog-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<div
		class="dialog-card max-w-sm text-left"
		role="alertdialog"
		aria-modal="true"
		aria-labelledby="remove-account-title"
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<div class="dialog-heading mb-2">
			<div>
				<h2
					id="remove-account-title"
					class="inline-flex min-w-0 items-center gap-1">
					<span class="truncate">Remove {account.displayName}</span>
					{#if account.favorite}<Icon
							svg={starFillIcon}
							class="shrink-0 animate-pop text-accent-strong"
							size={14}
							aria-hidden="true" />{/if}
					<span>(@{account.username})?</span>
				</h2>
				<p>This removes the account and its encrypted session.</p>
			</div>
			<button
				type="button"
				aria-label="Close remove account dialog"
				disabled={store.busy}
				onclick={close}>
				<Icon svg={xIcon} size={14} aria-hidden="true" />
			</button>
		</div>
		{#if store.error}
			<div class="mt-3 notice-error" role="alert">
				{store.error}
			</div>
		{/if}
		<div class="dialog-actions">
			<button class="btn" type="button" disabled={store.busy} onclick={close}
				>Cancel</button>
			<button
				class="btn-danger"
				type="button"
				disabled={store.busy}
				onclick={() => void remove()}>
				{store.busy ? "Removing…" : "Remove account"}
			</button>
		</div>
	</div>
</div>
