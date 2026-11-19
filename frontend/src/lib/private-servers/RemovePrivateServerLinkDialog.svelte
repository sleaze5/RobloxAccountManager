<script lang="ts">
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
</script>

<ConfirmationDialog
	title={`Remove ${server.name || "Saved server"}?`}
	description="The link is removed from the vault. This cannot be undone. The private server does not change."
	confirmLabel="Remove link"
	progressLabel="Removing…"
	isWorking={saved.removing === server.id}
	error={saved.actionError}
	onConfirm={() => void remove()}
	onCancel={close} />
