<script lang="ts">
	import EyeIcon from "phosphor-svelte/lib/EyeIcon"
	import EyeSlashIcon from "phosphor-svelte/lib/EyeSlashIcon"
	import ShieldCheckIcon from "phosphor-svelte/lib/ShieldCheckIcon"
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
		form = $state<HTMLFormElement>()

	async function testPassword(): Promise<void> {
		if (await store.testVaultPassword(password)) {
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
			void testPassword()
		}}>
		<div class="modal-icon"><ShieldCheckIcon size={20} aria-hidden="true" /></div>
		<h2>Test master password</h2>
		<p>Confirm that you still know the portable vault password.</p>
		{#if store.error}<div class="modal-error" role="alert">
				{store.error}
			</div>{/if}

		<label class="modal-field">
			<span>Master password</span>
			<input
				bind:value={password}
				type={visible ? "text" : "password"}
				autocomplete="current-password"
				required />
		</label>
		<button
			class="visibility-control"
			type="button"
			onclick={() => (visible = !visible)}>
			{#if visible}<EyeSlashIcon size={16} aria-hidden="true" /> Hide password{:else}<EyeIcon
					size={16}
					aria-hidden="true" /> Show password{/if}
		</button>
		{#if store.vault.passwordHint}
			<div class="modal-note">Hint: {store.vault.passwordHint}</div>
		{/if}
		<div class="modal-actions">
			<button type="button" disabled={store.busy} onclick={onClose}
				>Cancel</button>
			<button
				class="primary-action"
				type="submit"
				disabled={!password || store.busy}>Test password</button>
		</div>
	</form>
</div>
