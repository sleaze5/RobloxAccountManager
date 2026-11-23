<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import ConfirmationDialog from "../shared/ConfirmationDialog.svelte"
	import type { RobloxNotificationState } from "./roblox-notification-state.svelte"

	let {
		notifications,
		noun,
		username,
		onClose,
	}: {
		notifications: RobloxNotificationState
		noun: string
		username: string
		onClose: () => void
	} = $props()

	function close(): void {
		notifications.dismissAllError = ""
		onClose()
	}

	async function dismissAll(): Promise<void> {
		if (await notifications.dismissAll()) onClose()
	}

	notifyErrors("Notifications not dismissed", () => notifications.dismissAllError)
</script>

<ConfirmationDialog
	title={`Dismiss all ${noun}?`}
	description={`Roblox removes all ${noun} of @${username} on every device. This cannot be undone.`}
	confirmLabel="Dismiss all"
	progressLabel="Dismissing…"
	isWorking={notifications.dismissingAll}
	onConfirm={() => void dismissAll()}
	onCancel={close} />
