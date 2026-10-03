<script lang="ts">
	import { automaticUnlockDialogDescription } from "../shared/automatic-unlock"
	import { onMount } from "svelte"
	import Check from "@lucide/svelte/icons/check"
	import Eye from "@lucide/svelte/icons/eye"
	import EyeOff from "@lucide/svelte/icons/eye-off"
	import Info from "@lucide/svelte/icons/info"
	import LockKeyhole from "@lucide/svelte/icons/lock-keyhole"
	import X from "@lucide/svelte/icons/x"
	import { FileState } from "../backend/bridge"
	import type { BackupInfo } from "../backend/bridge"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import Select from "../shared/Select.svelte"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
		replayValidationNudge,
	} from "../shared/presence"
	import { checkPassword, passwordValid } from "./password-rules"

	let { store }: { store: AccountStore } = $props(),
		password = $state(""),
		confirmation = $state(""),
		hint = $state(""),
		automaticUnlock = $state(true),
		visible = $state(false),
		backups = $state<BackupInfo[]>([]),
		selectedBackup = $state(""),
		resetConfirmation = $state(""),
		recovery = $state(false),
		form = $state<HTMLFormElement>()

	const creating = $derived(store.vault.fileState === FileState.FileStateEmpty),
		incomplete = $derived(store.vault.fileState === FileState.FileStateIncomplete),
		recoveryMode = $derived(incomplete || recovery),
		rules = $derived(checkPassword(password)),
		canCreate = $derived(
			passwordValid(password) && password === confirmation && !store.busy,
		)

	onMount(async () => {
		if (incomplete) {
			backups = await store.listBackups()
			selectedBackup = backups[0]?.name ?? ""
		}
	})

	async function submit(): Promise<void> {
		if (creating) {
			if (
				!(await store.createVault(
					password,
					confirmation,
					hint,
					automaticUnlock,
				))
			) {
				replayValidationNudge(form)
			}
			return
		}
		if (incomplete && store.vault.unlocked && store.vault.dpapiRecovery) {
			if (
				!(await store.changeVaultPassword(
					"",
					password,
					confirmation,
					hint,
					automaticUnlock,
				))
			) {
				replayValidationNudge(form)
			}
			return
		}
		if (!(await store.unlockVault(password, automaticUnlock))) {
			replayValidationNudge(form)
		}
	}

	async function restore(): Promise<void> {
		if (await store.restoreBackup(selectedBackup, password)) {
			password = ""
		}
	}

	async function reset(): Promise<void> {
		if (await store.resetVault(resetConfirmation)) {
			resetConfirmation = ""
			backups = []
		}
	}

	async function showRecovery(): Promise<void> {
		recovery = true
		backups = await store.listBackups()
		selectedBackup = backups[0]?.name ?? ""
	}
</script>

<div class="modal-backdrop vault-gate" in:modalBackdropIn out:modalBackdropOut>
	<form
		novalidate
		class="modal-card vault-card"
		bind:this={form}
		use:modalFocus
		in:modalCardIn
		out:modalCardOut
		onsubmit={(event) => {
			event.preventDefault()
			void submit()
		}}>
		<div class="modal-icon"><LockKeyhole size={18} /></div>
		<h2>
			{creating
				? "Create account vault"
				: recoveryMode
					? "Recover account vault"
					: "Unlock vault"}
		</h2>
		<p>
			{creating
				? "Create a master password for the encrypted account vault."
				: recoveryMode
					? "The active vault files are missing or damaged. Repair the portable key, restore a backup, or reset the vault."
					: "Enter the master password to unlock Roblox account sessions."}
		</p>

		{#if store.vault.startupError}
			<div class="modal-warning">{store.vault.startupError}</div>
			{#if !creating && !recoveryMode}
				<button
					class="visibility-control"
					type="button"
					onclick={() => void showRecovery()}>Recovery options</button>
			{/if}
		{/if}
		{#if store.error}
			<div class="modal-error" role="alert">
				{store.error}
			</div>
			{#if !recoveryMode && !creating}
				<button
					class="visibility-control"
					type="button"
					onclick={() => void showRecovery()}>Recovery options</button>
			{/if}
		{/if}

		{#if recoveryMode && !store.vault.unlocked}
			<div class="modal-note">
				vault.key and autounlock.key are security files. Use the built-in backup
				and restore operations to copy them, and do not edit them.
			</div>
			{#if backups.length > 0}
				<div class="modal-field">
					<span>Verified backup candidate</span>
					<Select
						label="Verified backup candidate"
						value={selectedBackup}
						options={backups.map((backup) => ({
							value: backup.name,
							label: new Date(backup.createdAtMs).toLocaleString(),
						}))}
						disabled={store.busy}
						onChange={(name) => (selectedBackup = name)} />
				</div>
				<label class="modal-field">
					<span>Backup master password</span>
					<input
						bind:value={password}
						type="password"
						autocomplete="current-password" />
				</label>
				<button
					class="primary-action"
					type="button"
					disabled={!password || store.busy}
					onclick={() => void restore()}>
					Restore selected backup
				</button>
			{/if}
			<div class="modal-warning">
				Reset deletes vault.db, vault.key, and autounlock.key. Stored accounts
				cannot be recovered without a valid backup.
			</div>
			<label class="modal-field">
				<span>Type RESET to confirm</span>
				<input bind:value={resetConfirmation} autocomplete="off" />
			</label>
			<button
				class="danger-button vault-danger"
				type="button"
				disabled={resetConfirmation !== "RESET" || store.busy}
				onclick={() => void reset()}>
				Reset vault
			</button>
		{:else}
			<label class="modal-field">
				<span
					>{creating || incomplete
						? "New master password"
						: "Master password"}</span>
				<input
					bind:value={password}
					type={visible ? "text" : "password"}
					autocomplete={creating || incomplete
						? "new-password"
						: "current-password"}
					required />
			</label>
			<button
				class="visibility-control"
				type="button"
				onclick={() => (visible = !visible)}>
				{#if visible}<EyeOff size={14} /> Hide passwords{:else}<Eye size={14} /> Show
					passwords{/if}
			</button>
			{#if !creating && !incomplete && store.vault.passwordHint}
				<div class="modal-note">Hint: {store.vault.passwordHint}</div>
			{/if}
			{#if creating || incomplete}
				<label class="modal-field">
					<span>Confirm master password</span>
					<input
						bind:value={confirmation}
						type={visible ? "text" : "password"}
						autocomplete="new-password"
						required />
				</label>
				<ul class="password-checklist" aria-label="Password requirements">
					{#each [[rules.length, "8-128 characters long"], [rules.uppercase, "At least one upper-case letter"], [rules.lowercase, "At least one lower-case letter"], [rules.digit, "At least one number"], [rules.special, "At least one special character"]] as requirement}
						<li class:valid={requirement[0]}>
							{#if requirement[0]}<Check size={12} />{:else}<X
									size={12} />{/if}
							{requirement[1]}
						</li>
					{/each}
					<li class:valid={password !== "" && password === confirmation}>
						{#if password !== "" && password === confirmation}<Check
								size={12} />{:else}<X size={12} />{/if}
						Passwords match
					</li>
				</ul>
				<label class="modal-field">
					<span>Password hint (optional)</span>
					<input bind:value={hint} maxlength="480" autocomplete="off" />
				</label>
				<div class="modal-warning">
					The hint is stored in plain text. Do not include sensitive
					information.
				</div>
			{/if}
			<div class="automatic-unlock">
				<label>
					<span class="cookie-selection">
						<input bind:checked={automaticUnlock} type="checkbox" />
						<span aria-hidden="true"
							><Check size={10} strokeWidth={2.4} /></span>
					</span>
					Enable automatic unlock
				</label>
				<button
					class="automatic-unlock-info"
					type="button"
					aria-label="About automatic unlock"
					data-tooltip={automaticUnlockDialogDescription}>
					<Info size={13} />
				</button>
			</div>
			<button
				class="primary-action"
				type="submit"
				disabled={creating || incomplete
					? !canCreate
					: !password || store.busy}>
				{store.busy
					? "Working..."
					: creating
						? "Create vault"
						: incomplete
							? "Repair portable key"
							: "Unlock"}
			</button>
		{/if}
	</form>
</div>
