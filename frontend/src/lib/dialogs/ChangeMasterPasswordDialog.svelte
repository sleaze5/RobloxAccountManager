<script lang="ts">
	import { automaticUnlockDialogDescription } from "../shared/automatic-unlock"
	import CheckIcon from "phosphor-svelte/lib/CheckIcon"
	import EyeIcon from "phosphor-svelte/lib/EyeIcon"
	import EyeSlashIcon from "phosphor-svelte/lib/EyeSlashIcon"
	import InfoIcon from "phosphor-svelte/lib/InfoIcon"
	import KeyIcon from "phosphor-svelte/lib/KeyIcon"
	import XIcon from "phosphor-svelte/lib/XIcon"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
		replayValidationNudge,
	} from "../shared/presence"
	import { checkPassword, passwordValid } from "./password-rules"

	let { store, onClose }: { store: AccountStore; onClose: () => void } = $props(),
		currentPassword = $state(""),
		newPassword = $state(""),
		confirmation = $state(""),
		hint = $state(""),
		automaticUnlock = $state(true),
		visible = $state(false),
		form = $state<HTMLFormElement>()
	const rules = $derived(checkPassword(newPassword))

	async function changePassword(): Promise<void> {
		const changed = await store.changeVaultPassword(
			currentPassword,
			newPassword,
			confirmation,
			hint,
			automaticUnlock,
		)
		if (changed) {
			onClose()
		} else {
			replayValidationNudge(form)
		}
	}
</script>

<div class="modal-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<form
		novalidate
		class="modal-card vault-card"
		bind:this={form}
		use:modalFocus
		in:modalCardIn
		out:modalCardOut
		onsubmit={(event) => {
			event.preventDefault()
			void changePassword()
		}}>
		<div class="modal-icon"><KeyIcon size={20} aria-hidden="true" /></div>
		<h2>Change master password</h2>
		<p>
			Changing the password rotates the database encryption key and removes
			backups that contain old secrets.
		</p>
		{#if store.error}<div class="modal-error" role="alert">
				{store.error}
			</div>{/if}

		{#if store.vault.validAutomaticUnlock}
			<div class="modal-note">
				Windows automatic unlock has verified your access. You can change your
				master password without entering the current one.
			</div>
		{:else}
			<label class="modal-field">
				<span>Current master password</span>
				<input
					bind:value={currentPassword}
					type={visible ? "text" : "password"}
					autocomplete="current-password"
					required />
			</label>
		{/if}
		<label class="modal-field">
			<span>New master password</span>
			<input
				bind:value={newPassword}
				type={visible ? "text" : "password"}
				autocomplete="new-password"
				required />
		</label>
		<label class="modal-field">
			<span>Confirm new master password</span>
			<input
				bind:value={confirmation}
				type={visible ? "text" : "password"}
				autocomplete="new-password"
				required />
		</label>
		<button
			class="visibility-control"
			type="button"
			onclick={() => (visible = !visible)}>
			{#if visible}<EyeSlashIcon size={16} aria-hidden="true" /> Hide passwords{:else}<EyeIcon
					size={16}
					aria-hidden="true" /> Show passwords{/if}
		</button>
		<ul class="password-checklist">
			{#each [[rules.length, "8-128 characters long"], [rules.uppercase, "At least one upper-case letter"], [rules.lowercase, "At least one lower-case letter"], [rules.digit, "At least one number"], [rules.special, "At least one special character"]] as requirement}
				<li class:valid={requirement[0]}>
					{#if requirement[0]}<CheckIcon
							size={14}
							aria-hidden="true" />{:else}<XIcon
							size={14}
							aria-hidden="true" />{/if}{requirement[1]}
				</li>
			{/each}
			<li class:valid={newPassword !== "" && newPassword === confirmation}>
				{#if newPassword !== "" && newPassword === confirmation}<CheckIcon
						size={14}
						aria-hidden="true" />{:else}<XIcon
						size={14}
						aria-hidden="true" />{/if}Passwords match
			</li>
		</ul>
		<label class="modal-field"
			><span>Password hint (optional)</span><input
				bind:value={hint}
				maxlength="480"
				autocomplete="off" /></label>
		<div class="modal-warning">
			The hint is stored in plain text. Do not include sensitive information.
		</div>
		<div class="automatic-unlock">
			<label>
				<span class="cookie-selection">
					<input bind:checked={automaticUnlock} type="checkbox" />
					<span aria-hidden="true"
						><CheckIcon size={11} weight="bold" aria-hidden="true" /></span>
				</span>
				Enable automatic unlock
			</label>
			<button
				class="automatic-unlock-info"
				type="button"
				aria-label="About automatic unlock"
				data-tooltip={automaticUnlockDialogDescription}>
				<InfoIcon size={15} aria-hidden="true" />
			</button>
		</div>
		<div class="modal-actions">
			<button type="button" disabled={store.busy} onclick={onClose}
				>Cancel</button>
			<button
				class="primary-action"
				type="submit"
				disabled={!passwordValid(newPassword) ||
					newPassword !== confirmation ||
					(!currentPassword && !store.vault.validAutomaticUnlock) ||
					store.busy}>Change password</button>
		</div>
	</form>
</div>
