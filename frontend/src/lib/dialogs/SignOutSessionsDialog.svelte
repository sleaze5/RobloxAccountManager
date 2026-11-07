<script lang="ts">
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import type { Account } from "../accounts/account-model"
	import type { AccountSessionsState } from "../accounts/account-sessions-state.svelte"

	let {
		account,
		sessions,
		onClose,
	}: {
		account: Account
		sessions: AccountSessionsState
		onClose: () => void
	} = $props()

	function close(): void {
		if (sessions.signingOut) return
		sessions.signOutError = ""
		onClose()
	}

	async function signOut(): Promise<void> {
		if (await sessions.signOutOthers()) onClose()
	}
</script>

<Dialog
	class="max-w-sm text-left"
	role="alertdialog"
	aria-labelledby="sign-out-sessions-title"
	aria-describedby="sign-out-sessions-description"
	onkeydown={(event) => {
		if (event.key === "Escape") close()
	}}>
	<div class="dialog-heading mb-2">
		<div>
			<h2 id="sign-out-sessions-title">Sign out other sessions?</h2>
			<p id="sign-out-sessions-description">
				Every other browser, device, and app signed in as @{account.username} will
				be signed out. This app stays signed in with a new cookie. This cannot be
				undone.
			</p>
		</div>
		<button
			type="button"
			aria-label="Close sign out dialog"
			disabled={sessions.signingOut}
			onclick={close}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>
	{#if sessions.signOutError}
		<div class="mt-3 notice-error" role="alert">{sessions.signOutError}</div>
	{/if}
	<div class="dialog-actions">
		<button class="btn" type="button" disabled={sessions.signingOut} onclick={close}
			>Cancel</button>
		<button
			class="btn-danger"
			type="button"
			disabled={sessions.signingOut}
			onclick={() => void signOut()}>
			{sessions.signingOut ? "Signing out…" : "Sign out other sessions"}
		</button>
	</div>
</Dialog>
