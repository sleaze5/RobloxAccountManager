<script lang="ts">
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Dialog from "../shared/Dialog.svelte"
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
		if (notifications.dismissingAll) return
		notifications.dismissAllError = ""
		onClose()
	}

	async function dismissAll(): Promise<void> {
		if (await notifications.dismissAll()) onClose()
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") close()
	}} />

<Dialog
	class="max-w-sm text-left"
	role="alertdialog"
	aria-labelledby="dismiss-roblox-notifications-title"
	aria-describedby="dismiss-roblox-notifications-description"
	onBackdropClick={close}>
	<div class="dialog-heading mb-2">
		<div>
			<h2 id="dismiss-roblox-notifications-title">Dismiss all {noun}?</h2>
			<p id="dismiss-roblox-notifications-description">
				Roblox removes all {noun} of @{username} on every device. This cannot be undone.
			</p>
		</div>
		<button
			type="button"
			aria-label="Close dialog"
			disabled={notifications.dismissingAll}
			onclick={close}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>
	{#if notifications.dismissAllError}
		<div class="mt-3 notice-error" role="alert">
			{notifications.dismissAllError}
		</div>
	{/if}
	<div class="dialog-actions">
		<button
			class="button"
			type="button"
			disabled={notifications.dismissingAll}
			onclick={close}>Cancel</button>
		<button
			class="button-danger"
			type="button"
			aria-busy={notifications.dismissingAll}
			disabled={notifications.dismissingAll}
			onclick={() => void dismissAll()}>
			{notifications.dismissingAll ? "Dismissing…" : "Dismiss all"}
		</button>
	</div>
</Dialog>
