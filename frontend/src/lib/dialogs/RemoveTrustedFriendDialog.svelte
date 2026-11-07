<script lang="ts">
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import type { Account } from "../accounts/account-model"
	import type { SocialUser } from "../backend/bridge"
	import { personName } from "../social/social-store.svelte"
	import type { SocialStore } from "../social/social-store.svelte"

	let {
		store,
		account,
		friend,
		onClose,
	}: {
		store: SocialStore
		account: Account
		friend: SocialUser
		onClose: () => void
	} = $props()

	function close(): void {
		if (store.removingTrusted) return
		store.removeTrustedError = ""
		onClose()
	}

	async function remove(): Promise<void> {
		if (await store.removeTrustedFriend(friend)) onClose()
	}
</script>

<Dialog
	class="max-w-sm text-left"
	role="alertdialog"
	aria-labelledby="remove-trusted-title"
	aria-describedby="remove-trusted-description">
	<div class="dialog-heading mb-2">
		<div>
			<h2 id="remove-trusted-title" class="wrap-anywhere">
				Remove {personName(friend)} as a trusted friend?
			</h2>
			<p id="remove-trusted-description">
				They stay friends with @{account.username}, but adding them back takes a
				new request.
			</p>
		</div>
		<button
			type="button"
			aria-label="Close remove trusted friend dialog"
			disabled={store.removingTrusted}
			onclick={close}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>
	{#if store.removeTrustedError}
		<div class="mt-3 notice-error" role="alert">{store.removeTrustedError}</div>
	{/if}
	<div class="dialog-actions">
		<button
			class="button"
			type="button"
			disabled={store.removingTrusted}
			onclick={close}>Cancel</button>
		<button
			class="button-danger"
			type="button"
			disabled={store.removingTrusted}
			onclick={() => void remove()}>
			{store.removingTrusted ? "Removing…" : "Remove trusted friend"}
		</button>
	</div>
</Dialog>
