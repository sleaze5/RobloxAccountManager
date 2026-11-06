<script lang="ts">
	import eyeIcon from "@phosphor-icons/core/regular/eye.svg?raw"
	import eyeSlashIcon from "@phosphor-icons/core/regular/eye-slash.svg?raw"
	import shieldCheckIcon from "@phosphor-icons/core/regular/shield-check.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { replayValidationNudge } from "../shared/presence"
	import Dialog from "../shared/Dialog.svelte"

	let { store, onClose }: { store: AccountStore; onClose: () => void } = $props(),
		password = $state(""),
		visible = $state(false),
		card = $state<HTMLDivElement>()

	async function testPassword(): Promise<void> {
		if (await store.testVaultPassword(password)) {
			onClose()
		} else {
			replayValidationNudge(card)
		}
	}
</script>

<Dialog
	class="relative max-w-sm text-center"
	aria-labelledby="test-password-title"
	bind:card>
	<button
		class="absolute top-3 right-3 btn-close"
		type="button"
		aria-label="Close test master password dialog"
		disabled={store.busy}
		onclick={onClose}>
		<Icon svg={xIcon} size={14} aria-hidden="true" />
	</button>
	<form
		novalidate
		onsubmit={(event) => {
			event.preventDefault()
			void testPassword()
		}}>
		<div
			class="mx-auto mb-3 grid size-8 place-items-center rounded-md border border-line-strong bg-raised text-fg">
			<Icon svg={shieldCheckIcon} size={16} aria-hidden="true" />
		</div>
		<h2 id="test-password-title">Test master password</h2>
		<p>Confirm that you still know the portable vault password.</p>
		{#if store.error}<div class="mt-3 notice-error" role="alert">
				{store.error}
			</div>{/if}

		<label class="dialog-field">
			<span>Master password</span>
			<input
				bind:value={password}
				type={visible ? "text" : "password"}
				autocomplete="current-password"
				required />
		</label>
		<button
			class="mt-2 flex items-center gap-1 py-1 text-xs text-fg-muted transition-colors duration-100 *:animate-pop"
			type="button"
			onclick={() => (visible = !visible)}>
			{#if visible}<Icon svg={eyeSlashIcon} size={16} aria-hidden="true" /> Hide password{:else}<Icon
					svg={eyeIcon}
					size={16}
					aria-hidden="true" /> Show password{/if}
		</button>
		{#if store.vault.passwordHint}
			<div class="mt-3 notice-note">Hint: {store.vault.passwordHint}</div>
		{/if}
		<div class="dialog-actions">
			<button class="btn" type="button" disabled={store.busy} onclick={onClose}
				>Cancel</button>
			<button class="btn-primary" type="submit" disabled={!password || store.busy}
				>Test password</button>
		</div>
	</form>
</Dialog>
