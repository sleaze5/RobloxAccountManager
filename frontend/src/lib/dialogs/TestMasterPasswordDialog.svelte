<script lang="ts">
	import EyeIcon from "phosphor-svelte/lib/EyeIcon"
	import EyeSlashIcon from "phosphor-svelte/lib/EyeSlashIcon"
	import ShieldCheckIcon from "phosphor-svelte/lib/ShieldCheckIcon"
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

<div class="dialog-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<div
		class="relative dialog-card max-w-sm text-center"
		role="dialog"
		aria-modal="true"
		aria-labelledby="test-password-title"
		bind:this={card}
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<button
			class="absolute top-3 right-3 btn-close"
			type="button"
			aria-label="Close test master password dialog"
			disabled={store.busy}
			onclick={onClose}>
			<XIcon size={14} aria-hidden="true" />
		</button>
		<form
			novalidate
			onsubmit={(event) => {
				event.preventDefault()
				void testPassword()
			}}>
			<div
				class="mx-auto mb-3 grid size-8 place-items-center rounded-md border border-line-strong bg-raised text-fg">
				<ShieldCheckIcon size={16} aria-hidden="true" />
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
				{#if visible}<EyeSlashIcon size={16} aria-hidden="true" /> Hide password{:else}<EyeIcon
						size={16}
						aria-hidden="true" /> Show password{/if}
			</button>
			{#if store.vault.passwordHint}
				<div class="mt-3 notice-note">Hint: {store.vault.passwordHint}</div>
			{/if}
			<div class="dialog-actions">
				<button
					class="btn"
					type="button"
					disabled={store.busy}
					onclick={onClose}>Cancel</button>
				<button
					class="btn-primary"
					type="submit"
					disabled={!password || store.busy}>Test password</button>
			</div>
		</form>
	</div>
</div>
