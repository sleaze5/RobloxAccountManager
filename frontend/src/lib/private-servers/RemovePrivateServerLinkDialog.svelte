<script lang="ts">
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Dialog from "../shared/Dialog.svelte"
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

	const id = $props.id(),
		removing = $derived(saved.removing === server.id)

	function close(): void {
		if (removing) return
		saved.actionError = ""
		onClose()
	}

	async function remove(): Promise<void> {
		if (await saved.remove(server)) onClose()
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") close()
	}} />

<Dialog
	class="max-w-sm text-left"
	role="alertdialog"
	aria-labelledby={`${id}-title`}
	aria-describedby={`${id}-description`}>
	<div class="dialog-heading mb-2">
		<div class="min-w-0">
			<h2 id={`${id}-title`} class="wrap-anywhere">
				Remove {server.name || "Saved server"}?
			</h2>
			<p id={`${id}-description`}>
				The link is deleted from the vault and can't be restored. The private
				server itself doesn't change.
			</p>
		</div>
		<button
			type="button"
			aria-label="Close remove private server link"
			disabled={removing}
			onclick={close}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>
	{#if saved.actionError}
		<div class="mt-3 notice-error" role="alert">{saved.actionError}</div>
	{/if}
	<div class="dialog-actions">
		<button class="button" type="button" disabled={removing} onclick={close}
			>Cancel</button>
		<button
			class="button-danger"
			type="button"
			disabled={removing}
			onclick={() => void remove()}>
			{removing ? "Removing…" : "Remove link"}
		</button>
	</div>
</Dialog>
