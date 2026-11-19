<script lang="ts">
	import ConfirmationDialog from "../shared/ConfirmationDialog.svelte"
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
		sessions.signOutError = ""
		onClose()
	}

	async function signOut(): Promise<void> {
		if (await sessions.signOutOthers()) onClose()
	}
</script>

<ConfirmationDialog
	title="Sign out other sessions?"
	description={`Every other browser, device, and app signed in as @${account.username} will be signed out. This app stays signed in with a new cookie. This cannot be undone.`}
	confirmLabel="Sign out other sessions"
	progressLabel="Signing out…"
	isWorking={sessions.signingOut}
	error={sessions.signOutError}
	onConfirm={() => void signOut()}
	onCancel={close} />
