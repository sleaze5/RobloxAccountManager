<script lang="ts">
	import checkBoldIcon from "@phosphor-icons/core/bold/check-bold.svg?raw"
	import fileArrowUpIcon from "@phosphor-icons/core/regular/file-arrow-up.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
	} from "../shared/presence"
	import { updateStore } from "../updates/update-store.svelte"
	import ExportVaultDialog from "./ExportVaultDialog.svelte"

	let {
			store,
			notifications,
			onClose,
		}: {
			store: AccountStore
			notifications: NotificationCenter
			onClose: () => void
		} = $props(),
		exporting = $state(false),
		exported = $state(false)
	const version = $derived(updateStore.state.availableVersion ?? "")
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape" && !exporting) {
			event.preventDefault()
			onClose()
		}
	}} />

<div
	class="dialog-backdrop"
	in:modalBackdropIn
	out:modalBackdropOut
	onclick={(event) => {
		if (event.target === event.currentTarget) onClose()
	}}
	role="presentation">
	<div
		class="dialog-card max-w-md"
		role="dialog"
		aria-modal="true"
		aria-labelledby="update-export-title"
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<div class="dialog-heading">
			<div>
				<h2 id="update-export-title">Export your data before you update</h2>
				<p>
					Version {version} uses a new vault format and cannot open this vault.
				</p>
			</div>
			<button type="button" aria-label="Close update dialog" onclick={onClose}>
				<Icon svg={xIcon} size={14} aria-hidden="true" />
			</button>
		</div>
		<ol class="flex list-decimal flex-col gap-1 pl-5 text-xs text-fg-muted">
			<li>Export your data, then keep the file and its passcode safe.</li>
			<li>Download the update and restart.</li>
			<li>
				In version {version}, create a new vault. Then open Add account, choose
				Import export file, and select your file.
			</li>
		</ol>
		<div class="mt-3 notice-note">
			The update does not delete your current vault.
		</div>
		{#if !store.vault.unlocked}
			<div class="mt-3 notice-warning">Unlock the vault to export your data.</div>
		{/if}
		<div class="mt-3 flex items-center justify-between gap-2">
			<label class="flex items-center gap-1 text-xs text-fg-muted">
				<span class="checkbox">
					<input bind:checked={exported} type="checkbox" />
					<span aria-hidden="true"
						><Icon
							svg={checkBoldIcon}
							size={11}
							aria-hidden="true" /></span>
				</span>
				I exported my data and saved the passcode
			</label>
			<button
				class="btn"
				type="button"
				disabled={store.busy || !store.vault.unlocked}
				onclick={() => (exporting = true)}>
				<Icon svg={fileArrowUpIcon} size={16} aria-hidden="true" />
				Export data
			</button>
		</div>
		<div class="dialog-actions">
			<button class="btn" type="button" onclick={onClose}>Cancel</button>
			<button
				class="btn-primary"
				type="button"
				disabled={!exported}
				onclick={() => void updateStore.install(true)}>Download update</button>
		</div>
	</div>
</div>

{#if exporting}
	<ExportVaultDialog {notifications} onClose={() => (exporting = false)} />
{/if}
