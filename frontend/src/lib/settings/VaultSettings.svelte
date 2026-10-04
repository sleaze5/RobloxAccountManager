<script lang="ts">
	import KeyIcon from "phosphor-svelte/lib/KeyIcon"
	import LockKeyIcon from "phosphor-svelte/lib/LockKeyIcon"
	import ShieldCheckIcon from "phosphor-svelte/lib/ShieldCheckIcon"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import ChangeMasterPasswordDialog from "../dialogs/ChangeMasterPasswordDialog.svelte"
	import TestMasterPasswordDialog from "../dialogs/TestMasterPasswordDialog.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
	import Select from "../shared/Select.svelte"
	import SettingsStatus from "./SettingsStatus.svelte"
	import { automaticUnlockDescription } from "../shared/automatic-unlock"

	type ActiveDialog = "test" | "change" | null
	const reminderOptions = [
		{ value: 7, label: "Every 7 days" },
		{ value: 30, label: "Every 30 days" },
		{ value: 90, label: "Every 90 days" },
		{ value: 0, label: "Never" },
	]

	let {
			store,
			query = "",
			onLock,
		}: { store: AccountStore; query?: string; onLock: () => void } = $props(),
		activeDialog = $state<ActiveDialog>(null),
		normalizedQuery = $derived(query.trim().toLowerCase()),
		searchTerms = $derived(normalizedQuery.split(/\s+/).filter(Boolean))

	function matches(name: string, description: string): boolean {
		const text = `vault security ${name} ${description}`.toLowerCase()
		return searchTerms.every((term) => text.includes(term))
	}

	function openDialog(dialog: Exclude<ActiveDialog, null>): void {
		store.clearError()
		activeDialog = dialog
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (event.key !== "Escape") {
			return
		}
		if (activeDialog !== null && !store.busy) {
			activeDialog = null
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<section class="settings-section" aria-labelledby="vault-security-title">
	<h3 id="vault-security-title">Security</h3>
	{#if store.error && activeDialog === null}<div
			class="settings-inline-error"
			role="alert">
			{store.error}
		</div>{/if}
	<div class="settings-rows">
		{#if matches("Vault status", "Unlocked lock vault test master password last tested not tested yet")}
			<div class="settings-row">
				<div class="settings-row-copy">
					<strong>Vault status</strong>
					<span>
						{#if store.vault.lastPasswordTestedAt}
							Master password last tested <Timestamp
								value={Date.parse(store.vault.lastPasswordTestedAt)} />
						{:else}
							The master password has not been tested yet.
						{/if}
					</span>
					<SettingsStatus value="Unlocked" tone="success" />
				</div>
				<div class="settings-row-actions">
					<button
						class="settings-row-action"
						type="button"
						disabled={store.busy}
						onclick={() => openDialog("test")}>
						<ShieldCheckIcon size={16} aria-hidden="true" />
						Test password
					</button>
					<button
						class="settings-row-action"
						type="button"
						disabled={store.busy || !store.vault.unlocked}
						onclick={onLock}>
						<LockKeyIcon size={16} aria-hidden="true" />
						Lock
					</button>
				</div>
			</div>
		{/if}

		{#if matches("Automatic unlock", "Unlock vault when the app starts without master password enable disable on off operating system user account")}
			<div class="settings-row">
				<div class="settings-row-copy">
					<strong id="automatic-unlock-title">Automatic unlock</strong>
					<span id="automatic-unlock-description"
						>Unlock the vault without the master password when the app
						starts. Locking the vault turns it off.</span>
				</div>
				<label
					class="settings-toggle"
					data-tooltip={automaticUnlockDescription}>
					<input
						type="checkbox"
						role="switch"
						aria-labelledby="automatic-unlock-title"
						aria-describedby="automatic-unlock-description"
						checked={store.vault.automaticUnlock}
						disabled={store.busy || !store.vault.unlocked}
						onchange={(event) => {
							const enabled = event.currentTarget.checked
							event.currentTarget.checked = store.vault.automaticUnlock
							void store.setAutomaticUnlock(enabled)
						}} />
					<span aria-hidden="true"></span>
				</label>
			</div>
		{/if}

		{#if matches("Change master password", "Rotate vault encryption key update automatic unlock")}
			<div class="settings-row">
				<div class="settings-row-copy">
					<strong>Change master password</strong>
					<span
						>Rotate the vault encryption key and update automatic unlock.</span>
				</div>
				<button
					class="settings-row-action"
					type="button"
					disabled={store.busy}
					onclick={() => openDialog("change")}>
					<KeyIcon size={16} aria-hidden="true" />
					Change password
				</button>
			</div>
		{/if}

		{#if matches("Password test reminder", "Choose how often automatic unlock asks confirm master password")}
			<div class="settings-row">
				<div class="settings-row-copy">
					<strong>Password test reminder</strong>
					<span
						>Choose how often automatic unlock asks you to confirm the
						master password.</span>
				</div>
				<Select
					label="Password test reminder interval"
					value={store.vault.passwordTestIntervalDays}
					options={reminderOptions}
					disabled={store.busy}
					onChange={(days) => void store.setPasswordTestIntervalDays(days)} />
			</div>
		{/if}
	</div>
</section>

{#if activeDialog === "test"}
	<TestMasterPasswordDialog {store} onClose={() => (activeDialog = null)} />
{/if}

{#if activeDialog === "change"}
	<ChangeMasterPasswordDialog {store} onClose={() => (activeDialog = null)} />
{/if}
