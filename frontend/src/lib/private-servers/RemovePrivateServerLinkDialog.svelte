<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import ConfirmationDialog from "../shared/ConfirmationDialog.svelte"
	import type { SavedPrivateServer } from "../backend/bridge"
	import type { SavedServers } from "../games/game-servers-state.svelte"

	let {
		saved,
		server,
		onClose,
	}: {
		saved: SavedServers
		server: SavedPrivateServer
		onClose: () => void
	} = $props()

	function close(): void {
		saved.actionError = ""
		onClose()
	}

	async function remove(): Promise<void> {
		if (await saved.remove(server)) onClose()
	}

	notifyErrors("Link not removed", () => saved.actionError)
</script>

<ConfirmationDialog
	title={`Remove ${server.name || "Saved server"}?`}
	description="This removes the link from the vault. The private server does not change. This cannot be undone."
	confirmLabel="Remove link"
	progressLabel="Removing…"
	isWorking={saved.removing === server.id}
	onConfirm={() => void remove()}
	onCancel={close} />
