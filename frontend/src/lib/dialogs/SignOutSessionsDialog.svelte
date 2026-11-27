<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
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

	notifyErrors("Sessions not signed out", () => sessions.signOutError)
</script>

<ConfirmationDialog
	title="Sign out other sessions?"
	description={`This signs @${account.username} out of every other browser, device, and app. This app stays signed in with a new cookie. This cannot be undone.`}
	confirmLabel="Sign out other sessions"
	progressLabel="Signing out…"
	isWorking={sessions.signingOut}
	onConfirm={() => void signOut()}
	onCancel={close} />
