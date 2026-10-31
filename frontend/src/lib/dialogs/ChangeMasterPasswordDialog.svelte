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
		card = $state<HTMLDivElement>()
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
			replayValidationNudge(card)
		}
	}
</script>

<div class="dialog-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<div
		class="relative dialog-card max-w-sm text-center"
		role="dialog"
		aria-modal="true"
		aria-labelledby="change-password-title"
		bind:this={card}
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<button
			class="absolute top-3 right-3 btn-close"
			type="button"
			aria-label="Close change master password dialog"
			disabled={store.busy}
			onclick={onClose}>
			<XIcon size={14} aria-hidden="true" />
		</button>
		<form
			novalidate
			onsubmit={(event) => {
				event.preventDefault()
				void changePassword()
			}}>
			<div
				class="mx-auto mb-3 grid size-8 place-items-center rounded-md border border-line-strong bg-raised text-fg">
				<KeyIcon size={16} aria-hidden="true" />
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
					Automatic unlock has verified your access. You can change your
					master password without entering the current one.
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
				{#if visible}<EyeSlashIcon size={16} aria-hidden="true" /> Hide passwords{:else}<EyeIcon
						size={16}
						aria-hidden="true" /> Show passwords{/if}
			</button>
			<ul class="checklist" data-touched={newPassword !== "" || undefined}>
				{#each [[rules.length, "8-128 characters long"], [rules.uppercase, "At least one upper-case letter"], [rules.lowercase, "At least one lower-case letter"], [rules.digit, "At least one number"], [rules.special, "At least one special character"]] as requirement}
					<li data-valid={requirement[0] || undefined}>
						{#if requirement[0]}<CheckIcon
								size={14}
								aria-hidden="true" />{:else}<XIcon
								size={14}
								aria-hidden="true" />{/if}{requirement[1]}
					</li>
				{/each}
				<li
					data-valid={(newPassword !== "" && newPassword === confirmation) ||
						undefined}>
					{#if newPassword !== "" && newPassword === confirmation}<CheckIcon
							size={14}
							aria-hidden="true" />{:else}<XIcon
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
			<div
				class="mt-2 flex items-center justify-start gap-1 text-xs text-fg-muted">
				<label class="flex items-center gap-1">
					<span class="checkbox">
						<input bind:checked={automaticUnlock} type="checkbox" />
						<span aria-hidden="true"
							><CheckIcon
								size={11}
								weight="bold"
								aria-hidden="true" /></span>
					</span>
					Enable automatic unlock
				</label>
				<button
					class="flex size-5 cursor-help items-center justify-center text-fg-faint transition-colors duration-100 hover:text-fg focus-visible:text-fg"
					type="button"
					aria-label="About automatic unlock"
					data-tooltip={automaticUnlockDialogDescription}>
					<InfoIcon size={14} aria-hidden="true" />
				</button>
			</div>
			<div class="dialog-actions">
				<button
					class="btn"
					type="button"
					disabled={store.busy}
					onclick={onClose}>Cancel</button>
				<button
					class="btn-primary"
					type="submit"
					disabled={!passwordValid(newPassword) ||
						newPassword !== confirmation ||
						(!currentPassword && !store.vault.validAutomaticUnlock) ||
						store.busy}>Change password</button>
			</div>
		</form>
	</div>
</div>
