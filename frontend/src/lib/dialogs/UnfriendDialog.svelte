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
		if (store.unfriending) return
		store.unfriendError = ""
		onClose()
	}

	async function unfriend(): Promise<void> {
		if (await store.unfriend(friend)) onClose()
	}
</script>

<Dialog
	class="max-w-sm text-left"
	role="alertdialog"
	aria-labelledby="unfriend-title"
	aria-describedby="unfriend-description">
	<div class="dialog-heading mb-2">
		<div>
			<h2 id="unfriend-title" class="inline-flex min-w-0 items-center gap-1">
				<span class="truncate">Unfriend {personName(friend)}</span>
				<span>{friend.username ? `(@${friend.username})?` : "?"}</span>
			</h2>
			<p id="unfriend-description">
				This ends their Roblox friendship with @{account.username}. To be
				friends again, one of you must send a new request.
			</p>
		</div>
		<button
			type="button"
			aria-label="Close unfriend dialog"
			disabled={store.unfriending}
			onclick={close}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>
	{#if store.unfriendError}
		<div class="mt-3 notice-error" role="alert">{store.unfriendError}</div>
	{/if}
	<div class="dialog-actions">
		<button
			class="button"
			type="button"
			disabled={store.unfriending}
			onclick={close}>Cancel</button>
		<button
			class="button-danger"
			type="button"
			disabled={store.unfriending}
			onclick={() => void unfriend()}>
			{store.unfriending ? "Unfriending…" : "Unfriend"}
		</button>
	</div>
</Dialog>
