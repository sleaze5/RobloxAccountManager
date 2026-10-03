<script lang="ts">
	import Check from "@lucide/svelte/icons/check"
	import LoaderCircle from "@lucide/svelte/icons/loader-circle"
	import X from "@lucide/svelte/icons/x"
	import type { Account } from "../accounts/account-model"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { accountBackend } from "../backend/bridge"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
	} from "../shared/presence"

	let {
		store,
		account,
		onClose,
	}: {
		store: AccountStore
		account: Account
		onClose: () => void
	} = $props()
	let phase = $state<"confirm" | "pending" | "complete">("confirm"),
		error = $state(""),
		warning = $state("")
	let dialog = $state<HTMLDivElement | undefined>(undefined)
	const pending = $derived(phase === "pending")

	async function renew(): Promise<void> {
		if (phase !== "confirm" || store.busy || !store.vault.unlocked) return
		phase = "pending"
		store.busy = true
		dialog?.focus()
		try {
			const result = await accountBackend.RenewCookie(account.id)
			warning = result.warning ?? ""
		} catch (cause) {
			error = cause instanceof Error ? cause.message : "The renewal failed."
		} finally {
			phase = "complete"
			store.busy = false
			if (store.vault.unlocked) void store.refreshAccounts()
		}
	}
</script>

<div class="modal-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<div
		bind:this={dialog}
		tabindex="-1"
		class="modal-card confirm-card"
		role="dialog"
		aria-modal="true"
		aria-labelledby="renew-cookie-title"
		aria-describedby={phase === "confirm"
			? "renew-cookie-description"
			: "renew-cookie-status"}
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<div class="modal-heading compact-heading">
			<div>
				<h2 id="renew-cookie-title">
					{phase === "confirm" ? "Renew cookie?" : "Renew cookie"}
				</h2>
				<p>@{account.username}</p>
			</div>
			<button
				type="button"
				aria-label="Close cookie renewal"
				disabled={pending}
				onclick={onClose}>
				<X size={14} />
			</button>
		</div>
		{#if phase === "confirm"}
			<p id="renew-cookie-description">
				This replaces the saved session cookie. Any open browser for this
				account will need to be reopened.
			</p>
		{:else}
			<div
				id="renew-cookie-status"
				class="renewal-status"
				role="status"
				aria-live="polite">
				{#if pending}
					<LoaderCircle size={16} class="spinner" aria-hidden="true" />
					<span>Renewing the cookie and saving it…</span>
				{:else if error}
					<span>Renewal could not be confirmed.</span>
				{:else}
					<Check size={16} class="renewal-success" aria-hidden="true" />
					<span>The renewed cookie is saved.</span>
				{/if}
			</div>
		{/if}
		{#if error}
			<div class="modal-error" role="alert">{error}</div>
		{/if}
		{#if warning}
			<p class="renewal-warning" role="status">{warning}</p>
		{/if}
		<div class="modal-actions">
			{#if phase === "confirm"}
				<button type="button" onclick={onClose}>Cancel</button>
				<button
					class="warning-button"
					type="button"
					disabled={store.busy}
					onclick={() => void renew()}>Renew cookie</button>
			{:else}
				<button type="button" disabled={pending} onclick={onClose}
					>{pending ? "Renewing…" : "Close"}</button>
			{/if}
		</div>
	</div>
</div>
