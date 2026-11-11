<script lang="ts">
	import AutomaticUnlockCheckbox from "./AutomaticUnlockCheckbox.svelte"
	import checkIcon from "@phosphor-icons/core/regular/check.svg?raw"
	import eyeIcon from "@phosphor-icons/core/regular/eye.svg?raw"
	import eyeSlashIcon from "@phosphor-icons/core/regular/eye-slash.svg?raw"
	import keyIcon from "@phosphor-icons/core/regular/key.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { replayValidationNudge } from "../shared/presence"
	import Dialog from "../shared/Dialog.svelte"
	import { passwordRequirements, passwordValid } from "./password-rules"

	let { store, onClose }: { store: AccountStore; onClose: () => void } = $props(),
		currentPassword = $state(""),
		newPassword = $state(""),
		confirmation = $state(""),
		hint = $state(""),
		automaticUnlock = $state(true),
		visible = $state(false),
		card = $state<HTMLDivElement>()
	const requirements = $derived(passwordRequirements(newPassword))

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
			replayValidationNudge(card)
		}
	}
</script>

<Dialog
	class="relative max-w-md text-center"
	aria-labelledby="change-password-title"
	bind:card>
	<button
		class="absolute top-3 right-3 button-close"
		type="button"
		aria-label="Close change master password dialog"
		disabled={store.busy}
		onclick={onClose}>
		<Icon svg={xIcon} size={14} aria-hidden="true" />
	</button>
	<form
		novalidate
		onsubmit={(event) => {
			event.preventDefault()
			void changePassword()
		}}>
		<div
			class="mx-auto mb-3 grid size-8 place-items-center rounded-md border border-line-strong bg-raised text-fg">
			<Icon svg={keyIcon} size={16} aria-hidden="true" />
		</div>
		<h2 id="change-password-title">Change master password</h2>
		<p>
			Changing the password rotates the database encryption key and removes
			backups that contain old secrets.
		</p>
		{#if store.error}<div class="mt-3 notice-error" role="alert">
				{store.error}
			</div>{/if}

		{#if store.vault.validAutomaticUnlock}
			<div class="mt-3 notice-note">
				Automatic unlock has verified your access. You can change your master
				password without entering the current one.
			</div>
		{:else}
			<label class="dialog-field">
				<span>Current master password</span>
				<input
					bind:value={currentPassword}
					type={visible ? "text" : "password"}
					autocomplete="current-password"
					required />
			</label>
		{/if}
		<label class="dialog-field">
			<span>New master password</span>
			<input
				bind:value={newPassword}
				type={visible ? "text" : "password"}
				autocomplete="new-password"
				required />
		</label>
		<label class="dialog-field">
			<span>Confirm new master password</span>
			<input
				bind:value={confirmation}
				type={visible ? "text" : "password"}
				autocomplete="new-password"
				required />
		</label>
		<button
			class="mt-2 flex items-center gap-1 py-1 text-xs text-fg-muted transition-colors duration-100 *:animate-pop"
			type="button"
			onclick={() => (visible = !visible)}>
			{#if visible}<Icon svg={eyeSlashIcon} size={16} aria-hidden="true" /> Hide passwords{:else}<Icon
					svg={eyeIcon}
					size={16}
					aria-hidden="true" /> Show passwords{/if}
		</button>
		<ul class="checklist" data-touched={newPassword !== "" || undefined}>
			{#each requirements as requirement}
				<li data-valid={requirement[0] || undefined}>
					{#if requirement[0]}<Icon
							svg={checkIcon}
							size={14}
							aria-hidden="true" />{:else}<Icon
							svg={xIcon}
							size={14}
							aria-hidden="true" />{/if}{requirement[1]}
				</li>
			{/each}
			<li
				data-valid={(newPassword !== "" && newPassword === confirmation) ||
					undefined}>
				{#if newPassword !== "" && newPassword === confirmation}<Icon
						svg={checkIcon}
						size={14}
						aria-hidden="true" />{:else}<Icon
						svg={xIcon}
						size={14}
						aria-hidden="true" />{/if}Passwords match
			</li>
		</ul>
		<label class="dialog-field"
			><span>Password hint (optional)</span><input
				bind:value={hint}
				maxlength="480"
				autocomplete="off" /></label>
		<div class="mt-3 notice-warning">
			The hint is stored in plain text. Do not include sensitive information.
		</div>
		<AutomaticUnlockCheckbox bind:checked={automaticUnlock} />
		<div class="dialog-actions">
			<button class="button" type="button" disabled={store.busy} onclick={onClose}
				>Cancel</button>
			<button
				class="button-primary"
				type="submit"
				disabled={!passwordValid(newPassword) ||
					newPassword !== confirmation ||
					(!currentPassword && !store.vault.validAutomaticUnlock) ||
					store.busy}>Change password</button>
		</div>
	</form>
</Dialog>
