<script lang="ts">
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import type { Account } from "../accounts/account-model"

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

<Dialog
	class="max-w-sm text-left"
	role="alertdialog"
	aria-labelledby="remove-account-title">
	<div class="dialog-heading mb-2">
		<div>
			<h2
				id="remove-account-title"
				class="flex min-w-0 flex-wrap items-center gap-x-1 wrap-anywhere">
				<span class="min-w-0">Remove {account.displayName}</span>
				{#if account.favorite}<Icon
						svg={starFillIcon}
						class="shrink-0 animate-pop text-accent-strong"
						size={14}
						aria-hidden="true" />{/if}
				<span class="min-w-0">(@{account.username})?</span>
			</h2>
			<p>
				The Roblox account stays, but its saved session and every older backup
				are deleted, and this cannot be undone.
			</p>
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
		<button class="button" type="button" disabled={store.busy} onclick={close}
			>Cancel</button>
		<button
			class="button-danger"
			type="button"
			disabled={store.busy}
			onclick={() => void remove()}>
			{store.busy ? "Removing…" : "Remove account"}
		</button>
	</div>
</Dialog>
